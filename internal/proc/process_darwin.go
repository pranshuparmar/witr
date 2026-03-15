//go:build darwin

package proc

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	pidStr := strconv.Itoa(pid)

	// Format: pid(0) ppid(1) uid(2) lstart(3-7) state(8) ucomm(9) pcpu(10) rss(11) args(12+)
	// args= MUST be last because it is variable-width.
	cmd := exec.Command("ps", "-p", pidStr, "-o", "pid=,ppid=,uid=,lstart=,state=,ucomm=,pcpu=,rss=,args=")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		return model.Process{}, fmt.Errorf("process %d not found: %w", pid, err)
	}

	line := strings.TrimSpace(string(out))
	if line == "" {
		return model.Process{}, fmt.Errorf("process %d not found", pid)
	}

	fields := strings.Fields(line)
	if len(fields) < 12 {
		return model.Process{}, fmt.Errorf("unexpected ps output format for pid %d", pid)
	}

	ppid, _ := strconv.Atoi(fields[1])
	uid, _ := strconv.Atoi(fields[2])

	lstartStr := strings.Join(fields[3:8], " ")
	startedAt, _ := time.Parse("Mon Jan 2 15:04:05 2006", lstartStr)
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	state := fields[8]
	comm := fields[9]

	cpuPct, _ := strconv.ParseFloat(fields[10], 64)
	rssKB, _ := strconv.ParseFloat(fields[11], 64)

	rawCmdline := ""
	if len(fields) > 12 {
		rawCmdline = strings.Join(fields[12:], " ")
	}
	cmdline := rawCmdline
	if cmdline == "" {
		cmdline = comm
	}

	env := getEnvironment(pid)
	cwd, binPath := getCwdAndBinaryPath(pid)

	health := "healthy"
	forked := "unknown"

	switch state {
	case "Z":
		health = "zombie"
	case "T":
		health = "stopped"
	}

	if cpuPct > 90 {
		health = "high-cpu"
	}
	rssMB := rssKB / 1024
	if rssMB > 1024 {
		health = "high-mem"
	}

	if ppid != 1 && comm != "launchd" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	user := readUserByUID(uid)
	container := detectContainer(cmdline)

	if comm == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

	service := detectLaunchdService(pid)
	gitRepo, gitBranch := detectGitInfo(cwd)
	inodes := socketsForPID(pid)
	var ports []int
	var addrs []string

	for _, inode := range inodes {
		addrPort := strings.SplitN(inode, ":", 2)
		if len(addrPort) < 2 {
			continue
		}
		port, _ := strconv.Atoi(addrPort[1])
		ports = append(ports, port)
		addrs = append(addrs, addrPort[0])
	}

	displayName := deriveDisplayCommand(comm, rawCmdline)
	if displayName == "" {
		displayName = comm
	}

	exeDeleted := false
	if binPath != "" {
		_, statErr := os.Stat(binPath)
		exeDeleted = os.IsNotExist(statErr)
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
		ExeDeleted:     exeDeleted,
	}, nil
}

// getCwdAndBinaryPath returns the working directory and executable path for a process.
func getCwdAndBinaryPath(pid int) (cwd string, binPath string) {
	cwd = "unknown"

	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd,txt", "-F", "fn").Output()
	if err != nil {
		return cwd, ""
	}

	// lsof -F fn output has lines like:
	//   p<pid>
	//   fcwd
	//   n/path/to/cwd
	//   ftxt
	//   n/path/to/binary
	currentFD := ""
	for line := range strings.Lines(string(out)) {
		if len(line) < 2 {
			continue
		}
		switch line[0] {
		case 'f':
			currentFD = line[1:]
		case 'n':
			path := strings.TrimSpace(line[1:])
			switch currentFD {
			case "cwd":
				cwd = path
			case "txt":
				if binPath == "" {
					binPath = path
				}
			}
		}
	}

	return cwd, binPath
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

// detectContainer checks the command line for container runtime patterns.
func detectContainer(cmdline string) string {
	lowerCmd := strings.ToLower(cmdline)

	switch {
	case strings.Contains(lowerCmd, "docker"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "docker: " + name
		}
		return "docker"
	case strings.Contains(lowerCmd, "podman"), strings.Contains(lowerCmd, "libpod"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "podman: " + name
		}
		return "podman"
	case strings.Contains(lowerCmd, "minikube"):
		if profile := extractFlagValue(cmdline, "-p", "--profile"); profile != "" {
			return "k8s: " + profile
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "kind"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "k8s: " + name
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "kubepods"):
		if id := findLongHexID(cmdline); id != "" {
			if name := resolveContainerName(id, "crictl"); name != "" {
				return "k8s: " + name
			}
			return "k8s (" + shortID(id) + ")"
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "colima"):
		if profile := extractFlagValue(cmdline, "-p", "--profile"); profile != "" {
			return "colima: " + profile
		}
		return "colima: default"
	case strings.Contains(lowerCmd, "nerdctl"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "containerd: " + name
		}
		return "containerd"
	case strings.Contains(lowerCmd, "containerd"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "containerd: " + name
		}
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
	for depth := 0; depth < 10 && searchDir != "/" && searchDir != "." && searchDir != ""; depth++ {
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
					gitBranch = strings.TrimPrefix(ref, "refs/heads/")
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
