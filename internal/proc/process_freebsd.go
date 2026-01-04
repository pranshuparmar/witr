//go:build freebsd

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
	// Read process info using ps command on FreeBSD
	// FreeBSD ps uses different syntax: -o pid -o ppid (separate -o for each field)
	// Note: FreeBSD ps always outputs headers, we need to skip them

	pidStr := strconv.Itoa(pid)

	// Get basic process info: pid, ppid, uid, state, comm
	cmd := exec.Command("ps", "-p", pidStr, "-o", "pid", "-o", "ppid", "-o", "uid", "-o", "state", "-o", "comm")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		return model.Process{}, fmt.Errorf("process %d not found: %w", pid, err)
	}

	// Skip header line and get data line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return model.Process{}, fmt.Errorf("process %d not found", pid)
	}

	// Parse second line (skip header): pid ppid uid state comm
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return model.Process{}, fmt.Errorf("unexpected ps output format for pid %d: got %d fields in %q", pid, len(fields), lines[1])
	}

	ppid, _ := strconv.Atoi(fields[1])
	uid, _ := strconv.Atoi(fields[2])
	state := fields[3]
	comm := fields[4]

	// Get start time separately
	startedAt := getProcessStartTime(pid)
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	// Get full command line
	cmdline := getCommandLine(pid)
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

	// FreeBSD states can be multi-character like "Is", "Ss", "R", "Z", "T"
	// Check first character for main state
	if len(state) > 0 {
		switch state[0] {
		case 'Z':
			health = "zombie"
		case 'T':
			health = "stopped"
		}
	}

	// Fork detection
	if ppid != 1 && comm != "init" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	// Get user from UID
	user := readUserByUID(uid)

	// Container detection on FreeBSD (jails)
	container := detectContainer(pid)

	if comm == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

	// Service detection (rc.d)
	service := detectRcService(pid)

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

	return model.Process{
		PID:            pid,
		PPID:           ppid,
		Command:        comm,
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
	}, nil
}

func getProcessStartTime(pid int) time.Time {
	// Get start time using ps lstart
	// FreeBSD syntax: ps -p <pid> -o lstart
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "lstart")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		return time.Time{}
	}

	// Skip header line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return time.Time{}
	}

	lstartStr := strings.TrimSpace(lines[1])
	if lstartStr == "" {
		return time.Time{}
	}

	// FreeBSD lstart format with LC_ALL=C: "Thu Jan  2 10:26:00 2025"
	// Try multiple formats
	formats := []string{
		"Mon Jan 2 15:04:05 2006",
		"Mon Jan  2 15:04:05 2006",
		"Mon Jan 02 15:04:05 2006",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, lstartStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

func getCommandLine(pid int) string {
	// Use ps to get full command line
	// FreeBSD syntax: ps -p <pid> -o args
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args").Output()
	if err != nil {
		return ""
	}

	// Skip header line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return ""
	}
	return strings.TrimSpace(lines[1])
}

func getEnvironment(pid int) []string {
	var env []string

	// Use procstat -e to get environment variables
	// procstat does not require procfs to be mounted
	out, err := exec.Command("procstat", "-e", strconv.Itoa(pid)).Output()
	if err != nil {
		return env
	}

	// Parse procstat -e output
	// Format: PID COMM ENVVAR=VALUE ...
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		// Skip header and PID/COMM columns
		for _, field := range fields[2:] {
			if strings.Contains(field, "=") {
				env = append(env, field)
			}
		}
	}

	return env
}

func getWorkingDirectory(pid int) string {
	// Use procstat -f to get current working directory
	// procstat -f shows file descriptors, including a special "cwd" entry
	// procstat does not require procfs to be mounted
	out, err := exec.Command("procstat", "-f", strconv.Itoa(pid)).Output()
	if err != nil {
		return "unknown"
	}

	// Parse procstat -f output for cwd
	// Output format: PID COMM FD TYPE FLAGS ... PATH
	// The cwd line has "cwd" in the FD column (typically 3rd column)
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		// Look for the line where FD column is "cwd"
		// Typical format has at least: PID COMM FD TYPE ... PATH
		if len(fields) >= 4 && fields[2] == "cwd" {
			// The path is in the last column
			return fields[len(fields)-1]
		}
	}

	return "unknown"
}

func detectContainer(pid int) string {
	// On FreeBSD, check if running inside a jail by checking the jail ID
	// JID = 0 means running on host, JID > 0 means running in a jail
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "jid=").Output()
	if err == nil {
		jid := strings.TrimSpace(string(out))
		if jid != "" && jid != "0" {
			return "jail"
		}
	}

	// Check command line for container patterns
	cmdline := getCommandLine(pid)
	lowerCmd := strings.ToLower(cmdline)

	switch {
	case strings.Contains(lowerCmd, "docker"):
		return "docker"
	case strings.Contains(lowerCmd, "podman"), strings.Contains(lowerCmd, "libpod"):
		return "podman"
	case strings.Contains(lowerCmd, "kubepods"):
		return "kubernetes"
	case strings.Contains(lowerCmd, "containerd"):
		return "containerd"
	}

	return ""
}

func detectRcService(pid int) string {
	// FreeBSD uses rc.d for service management
	// Try to find the service by checking /var/run/*.pid files
	pidStr := strconv.Itoa(pid)

	entries, err := os.ReadDir("/var/run")
	if err != nil {
		return ""
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".pid") {
			continue
		}

		pidFile := "/var/run/" + entry.Name()
		content, err := os.ReadFile(pidFile)
		if err != nil {
			continue
		}

		if strings.TrimSpace(string(content)) == pidStr {
			// Found matching PID file
			serviceName := strings.TrimSuffix(entry.Name(), ".pid")
			return serviceName
		}
	}

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
	// FreeBSD syntax: ps -p <pid> -o pcpu -o rss
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pcpu", "-o", "rss").Output()
	if err != nil {
		return currentHealth
	}

	// Skip header line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return currentHealth
	}

	fields := strings.Fields(lines[1])
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
