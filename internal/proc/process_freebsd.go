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
	pidStr := strconv.Itoa(pid)

	// Format: pid(0) ppid(1) uid(2) jid(3) state(4) pcpu(5) rss(6) lstart(7-11) comm(12) args(13+)
	// args= MUST be last because it is variable-width.
	cmd := exec.Command("ps", "-p", pidStr,
		"-o", "pid=", "-o", "ppid=", "-o", "uid=", "-o", "jid=",
		"-o", "state=", "-o", "pcpu=", "-o", "rss=",
		"-o", "lstart=", "-o", "comm=", "-o", "args=")
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
	if len(fields) < 13 {
		return model.Process{}, fmt.Errorf("unexpected ps output format for pid %d: got %d fields in %q", pid, len(fields), line)
	}

	ppid, _ := strconv.Atoi(fields[1])
	uid, _ := strconv.Atoi(fields[2])
	jid := fields[3]
	state := fields[4]
	cpuPct, _ := strconv.ParseFloat(fields[5], 64)
	rssKB, _ := strconv.ParseFloat(fields[6], 64)

	lstartStr := strings.Join(fields[7:12], " ")
	startedAt := parseLstart(lstartStr)
	if startedAt.IsZero() {
		startedAt = time.Now().UTC()
	}

	comm := fields[12]

	cmdline := comm
	if len(fields) > 13 {
		cmdline = strings.Join(fields[13:], " ")
	}

	cwd, binPath := getCwdAndBinaryPath(pid)
	env := getEnvironment(pid)

	health := "healthy"
	forked := "unknown"

	// FreeBSD states can be multi-character like "Is", "Ss", "R", "Z", "T"
	if len(state) > 0 {
		switch state[0] {
		case 'Z':
			health = "zombie"
		case 'T':
			health = "stopped"
		}
	}

	if cpuPct > 90 {
		health = "high-cpu"
	}
	rssMB := rssKB / 1024
	if rssMB > 1024 {
		health = "high-mem"
	}

	if ppid != 1 && comm != "init" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	user := readUserByUID(uid)
	container := detectContainer(jid, cmdline)
	displayName := deriveDisplayCommand(comm, cmdline)
	if displayName == "" {
		displayName = comm
	}

	if comm == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

	service := detectRcService(pid)
	gitRepo, gitBranch := detectGitInfo(cwd)
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

	out, err := exec.Command("procstat", "-f", strconv.Itoa(pid)).Output()
	if err != nil {
		return cwd, ""
	}

	// procstat -f output format: PID COMM FD TYPE FLAGS ... PATH
	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		switch fields[2] {
		case "cwd":
			cwd = fields[len(fields)-1]
		case "text":
			if binPath == "" {
				binPath = fields[len(fields)-1]
			}
		}
	}

	return cwd, binPath
}

func parseLstart(lstartStr string) time.Time {
	if lstartStr == "" {
		return time.Time{}
	}
	// FreeBSD lstart format with LC_ALL=C: "Thu Jan  2 10:26:00 2025"
	// strings.Fields collapses double spaces, so try the standard format.
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

// detectContainer checks for jail membership and container runtime patterns.
func detectContainer(jid, cmdline string) string {
	// JID = 0 means running on host, JID > 0 means running in a jail
	if jid != "" && jid != "0" {
		if name := resolveJailName(jid); name != "" {
			return "jail: " + name
		}
		return "jail (" + jid + ")"
	}

	// Check command line for container patterns
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

func resolveJailName(jid string) string {
	out, err := exec.Command("jls", "-j", jid, "name").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
