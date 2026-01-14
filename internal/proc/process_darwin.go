//go:build darwin

package proc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	// Read process info using ps command on macOS
	// LC_ALL=C TZ=UTC ps -p <pid> -o pid=,ppid=,uid=,lstart=,state=,ucomm=
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid=,ppid=,uid=,lstart=,state=,ucomm=")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		return model.Process{}, fmt.Errorf("process %d not found: %w", pid, err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) == "" {
		return model.Process{}, fmt.Errorf("process %d not found", pid)
	}

	// Parse the first line
	fields := strings.Fields(lines[0])
	if len(fields) < 9 {
		// lstart is 5 fields: Mon Dec 25 12:00:00 2024
		return model.Process{}, fmt.Errorf("unexpected ps output format for pid %d", pid)
	}

	ppid, _ := strconv.Atoi(fields[1])
	uid, _ := strconv.Atoi(fields[2])

	// lstart is 5 fields: Mon Dec 25 12:00:00 2024
	lstartStr := strings.Join(fields[3:8], " ")
	startedAt, _ := time.Parse("Mon Jan 2 15:04:05 2006", lstartStr)
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	state := fields[8]
	comm := ""
	if len(fields) > 9 {
		comm = fields[9]
	}

	// Get full command line
	rawCmdline := getCommandLine(pid)
	cmdline := rawCmdline
	if cmdline == "" {
		cmdline = comm
	}

	// Get environment variables
	env := getEnvironment(pid)

	// Get working directory
	cwd := getWorkingDirectory(pid)

	// Health status
	health := "healthy"
	forked := "unknown"

	switch state {
	case "Z":
		health = "zombie"
	case "T":
		health = "stopped"
	}

	// Fork detection
	if ppid != 1 && comm != "launchd" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	// Get user from UID
	user := readUserByUID(uid)

	// Container detection on macOS (Docker for Mac)
	container := detectContainer(pid)

	if comm == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

	// Service detection (launchd)
	service := detectLaunchdService(pid)

	// Git repo/branch detection
	gitRepo, gitBranch := detectGitInfo(cwd)

	// Get listening ports for this process
	sockets, _ := readListeningSockets()
	inodes := socketsForPID(pid)

	var ports []int
	var addrs []string

	for _, inode := range inodes {
		if s, ok := sockets[inode]; ok {
			ports = append(ports, s.Port)
			addrs = append(addrs, s.Address)
		}
	}

	// Check for high resource usage
	health = checkResourceUsage(pid, health)

	displayName := deriveDisplayCommand(comm, rawCmdline)
	if displayName == "" {
		displayName = comm
	}

	return model.Process{
		PID:            pid,
		PPID:           ppid,
		Command:        displayName,
		Cmdline:        cmdline,
		StartedAt:      startedAt,
		User:           user,
		WorkingDir:     cwd,
		GitRepo:        gitRepo,
		GitBranch:      gitBranch,
		Container:      container,
		Service:        service,
		ListeningPorts: ports,
		BindAddresses:  addrs,
		Health:         health,
		Forked:         forked,
		Env:            env,
		ExeDeleted:     isBinaryDeleted(pid),
	}, nil
}

func isBinaryDeleted(pid int) bool {
	// Use lsof to get the executable path (txt)
	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "txt", "-F", "n").Output()
	if err != nil {
		return false
	}

	path := ""
	for line := range strings.Lines(string(out)) {
		if len(line) > 1 && line[0] == 'n' {
			path = line[1:]
			break
		}
	}

	if path == "" {
		return false
	}

	_, err = os.Stat(path)
	return os.IsNotExist(err)
}

// deriveDisplayCommand returns a human-readable command name that avoids macOS
// ps(1)"ucomm" truncation by falling back to the executable extracted from the
// full command line when the short name looks clipped.
func deriveDisplayCommand(comm, cmdline string) string {
	trimmedComm := strings.TrimSpace(comm)
	exe := extractExecutableName(cmdline)
	if trimmedComm == "" {
		return exe
	}
	if exe == "" {
		return trimmedComm
	}
	if strings.HasPrefix(exe, trimmedComm) && len(trimmedComm) < len(exe) {
		return exe
	}
	return trimmedComm
}

func extractExecutableName(cmdline string) string {
	args := splitCmdline(cmdline)
	for _, arg := range args {
		if arg == "" {
			continue
		}
		if strings.Contains(arg, "=") && !strings.Contains(arg, "/") {
			// Skip leading environment assignments.
			continue
		}
		clean := strings.Trim(arg, "\"'")
		if clean == "" {
			continue
		}
		base := filepath.Base(clean)
		if base == "." || base == "" || base == "/" {
			continue
		}
		return base
	}
	return ""
}

func splitCmdline(cmdline string) []string {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range cmdline {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"' || r == '\'':
			if quote == 0 {
				quote = r
				continue
			}
			if quote == r {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case unicode.IsSpace(r) && quote == 0:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

func getCommandLine(pid int) string {
	// Use ps to get full command line
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args=").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func getEnvironment(pid int) []string {
	var env []string

	// On macOS, getting environment of another process requires elevated privileges
	// or using the proc_pidinfo syscall. For simplicity, we use ps -E when available
	// Note: This might not work for all processes due to SIP restrictions

	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-E", "-o", "command=").Output()
	if err != nil {
		return env
	}

	// The -E output appends environment to the command
	// This is a simplified approach; full env parsing would need libproc
	output := string(out)

	// Look for common environment variable patterns
	for _, part := range strings.Fields(output) {
		if strings.Contains(part, "=") && !strings.HasPrefix(part, "-") {
			// Basic validation - should look like VAR=value
			eqIdx := strings.Index(part, "=")
			if eqIdx > 0 {
				varName := part[:eqIdx]
				// Check if it looks like an env var name (uppercase or common patterns)
				if isEnvVarName(varName) {
					env = append(env, part)
				}
			}
		}
	}

	return env
}

func isEnvVarName(name string) bool {
	if len(name) == 0 {
		return false
	}
	// Common env var patterns
	for _, c := range name {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '_') {
			return false
		}
	}
	return true
}

func getWorkingDirectory(pid int) string {
	// Use lsof to get current working directory
	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-F", "n").Output()
	if err != nil {
		return "unknown"
	}

	for line := range strings.Lines(string(out)) {
		if len(line) > 1 && line[0] == 'n' {
			return line[1:]
		}
	}

	return "unknown"
}

func detectContainer(pid int) string {
	// On macOS, check if running inside Docker for Mac
	// Docker for Mac runs processes inside a Linux VM, but we can check
	// if the process has Docker-related environment or parent processes

	cmdline := getCommandLine(pid)
	lowerCmd := strings.ToLower(cmdline)

	switch {
	case strings.Contains(lowerCmd, "docker"):
		return "docker"
	case strings.Contains(lowerCmd, "podman"), strings.Contains(lowerCmd, "libpod"):
		return "podman"
	case strings.Contains(lowerCmd, "kubepods"):
		return "kubernetes"
	case strings.Contains(lowerCmd, "colima"):
		return "colima"
	case strings.Contains(lowerCmd, "containerd"):
		return "containerd"
	}

	return ""
}

func detectLaunchdService(pid int) string {
	// Try to find the launchd service managing this process
	// Use launchctl blame on macOS 10.10+

	out, err := exec.Command("launchctl", "blame", strconv.Itoa(pid)).Output()
	if err == nil {
		blame := strings.TrimSpace(string(out))
		if blame != "" && !strings.Contains(blame, "unknown") {
			return blame
		}
	}

	// Fallback: check if process is a known launchd service
	// by looking at the parent chain or service database
	return ""
}

func detectGitInfo(cwd string) (string, string) {
	if cwd == "unknown" || cwd == "" {
		return "", ""
	}

	searchDir := cwd
	for searchDir != "/" && searchDir != "." && searchDir != "" {
		gitDir := searchDir + "/.git"
		if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
			// Repo name is the base dir
			parts := strings.Split(strings.TrimRight(searchDir, "/"), "/")
			gitRepo := parts[len(parts)-1]

			// Try to read HEAD for branch
			gitBranch := ""
			headFile := gitDir + "/HEAD"
			if head, err := os.ReadFile(headFile); err == nil {
				headStr := strings.TrimSpace(string(head))
				if strings.HasPrefix(headStr, "ref: ") {
					ref := strings.TrimPrefix(headStr, "ref: ")
					refParts := strings.Split(ref, "/")
					gitBranch = refParts[len(refParts)-1]
				}
			}

			return gitRepo, gitBranch
		}

		// Move up one directory
		idx := strings.LastIndex(searchDir, "/")
		if idx <= 0 {
			break
		}
		searchDir = searchDir[:idx]
	}

	return "", ""
}

// buildEnvForPS returns environment variables with LC_ALL=C and TZ=UTC,
// removing any existing LC_ALL or TZ to ensure consistent output format.
func buildEnvForPS() []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "LC_ALL=") && !strings.HasPrefix(e, "TZ=") {
			env = append(env, e)
		}
	}
	env = append(env, "LC_ALL=C", "TZ=UTC")
	return env
}

func checkResourceUsage(pid int, currentHealth string) string {
	// Use ps to get CPU and memory usage
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pcpu=,rss=").Output()
	if err != nil {
		return currentHealth
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return currentHealth
	}

	// Check CPU percentage
	cpuPct, _ := strconv.ParseFloat(fields[0], 64)
	if cpuPct > 90 {
		return "high-cpu"
	}

	// Check RSS memory in KB
	rssKB, _ := strconv.ParseFloat(fields[1], 64)
	rssMB := rssKB / 1024
	if rssMB > 1024 { // > 1GB
		return "high-mem"
	}

	return currentHealth
}

func resolveDockerProxyContainer(cmdline string) string {
	var containerIP string
	parts := strings.Fields(cmdline)
	for i, part := range parts {
		if part == "-container-ip" && i+1 < len(parts) {
			containerIP = parts[i+1]
			break
		}
	}
	if containerIP == "" {
		return ""
	}

	out, err := exec.Command("docker", "network", "inspect", "bridge",
		"--format", "{{range .Containers}}{{.Name}}:{{.IPv4Address}}{{\"\\n\"}}{{end}}").Output()
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx == -1 {
			continue
		}
		name := line[:colonIdx]
		ip := strings.Split(line[colonIdx+1:], "/")[0]
		if ip == containerIP {
			return "target: " + name
		}
	}
	return ""
}
