//go:build linux

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

// isValidSymlinkTarget validates that a symlink target is safe and reasonable
func isValidSymlinkTarget(target string) bool {
	if target == "" {
		return false
	}

	// Reject absolute paths that seem suspicious
	if strings.HasPrefix(target, "/") {
		// Allow normal absolute paths but reject system-critical ones
		suspiciousPaths := []string{"/proc", "/sys", "/dev", "/boot", "/root"}
		for _, suspicious := range suspiciousPaths {
			if strings.HasPrefix(target, suspicious) {
				return false
			}
		}
	}

	// Reject relative paths that could escape
	if strings.Contains(target, "../") {
		return false
	}

	return true
}

func ReadProcess(pid int) (model.Process, error) {
	// Verify process still exists before reading
	if _, err := os.Stat(fmt.Sprintf("/proc/%d", pid)); os.IsNotExist(err) {
		return model.Process{}, fmt.Errorf("process %d does not exist", pid)
	}

	// Read all proc files in a logical order to minimize TOCTOU issues
	// Start with stat file which is most likely to fail if process disappears
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	stat, err := os.ReadFile(statPath)
	if err != nil {
		return model.Process{}, fmt.Errorf("process %d disappeared during read", pid)
	}

	// Read environment variables
	env := []string{}
	envBytes, errEnv := os.ReadFile(fmt.Sprintf("/proc/%d/environ", pid))
	if errEnv == nil {
		for _, e := range strings.Split(string(envBytes), "\x00") {
			if e != "" {
				env = append(env, e)
			}
		}
	}
	// Health status
	health := "healthy"

	// Working directory
	var cwd, cwdErr = os.Readlink(fmt.Sprintf("/proc/%d/cwd", pid))
	if cwdErr != nil {
		cwd = "unknown"
	} else {
		// Validate symlink target is reasonable
		if !isValidSymlinkTarget(cwd) {
			cwd = "invalid"
		}
	}

	// Container detection
	container := ""
	cgroupFile := fmt.Sprintf("/proc/%d/cgroup", pid)
	if cgroupData, err := os.ReadFile(cgroupFile); err == nil {
		cgroupStr := string(cgroupData)
		switch {
		case strings.Contains(cgroupStr, "docker"):
			container = "docker"
		case strings.Contains(cgroupStr, "podman"), strings.Contains(cgroupStr, "libpod"):
			container = "podman"
		case strings.Contains(cgroupStr, "kubepods"):
			container = "kubernetes"
		case strings.Contains(cgroupStr, "colima"):
			container = "colima"
		case strings.Contains(cgroupStr, "containerd"):
			container = "containerd"
		}
	}

	// Service detection (try systemctl show for this PID)
	service := ""
	svcOut, err := exec.Command("systemctl", "status", fmt.Sprintf("%d", pid)).CombinedOutput()
	if err == nil && strings.Contains(string(svcOut), "Loaded: loaded") {
		// Try to extract service name from output
		for line := range strings.Lines(string(svcOut)) {
			if strings.HasPrefix(line, "Loaded:") && strings.Contains(line, ".service") {
				parts := strings.Fields(line)
				for _, part := range parts {
					if strings.HasSuffix(part, ".service") {
						service = part
						break
					}
				}
			}
		}
	}

	// Git repo/branch detection (walk up to find .git)
	gitRepo := ""
	gitBranch := ""
	if cwd != "unknown" {
		searchDir := cwd
		for searchDir != "/" && searchDir != "." && searchDir != "" {
			gitDir := searchDir + "/.git"
			if fi, err := os.Stat(gitDir); err == nil && fi.IsDir() {
				// Repo name is the base dir
				parts := strings.Split(strings.TrimRight(searchDir, "/"), "/")
				gitRepo = parts[len(parts)-1]
				// Try to read HEAD for branch
				headFile := gitDir + "/HEAD"
				if head, err := os.ReadFile(headFile); err == nil {
					headStr := strings.TrimSpace(string(head))
					if strings.HasPrefix(headStr, "ref: ") {
						ref := strings.TrimPrefix(headStr, "ref: ")
						refParts := strings.Split(ref, "/")
						gitBranch = refParts[len(refParts)-1]
					}
				}
				break
			}
			// Move up one directory
			idx := strings.LastIndex(searchDir, "/")
			if idx <= 0 {
				break
			}
			searchDir = searchDir[:idx]
		}
	}

	// stat format is evil, command is inside ()
	raw := string(stat)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open == -1 || close == -1 {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	comm := raw[open+1 : close]
	fields := strings.Fields(raw[close+2:])

	ppid, _ := strconv.Atoi(fields[1])
	state := processState(fields)
	startTicks, _ := strconv.ParseInt(fields[19], 10, 64)

	// Fork detection: if ppid != 1 and not systemd, likely forked; also check for vfork/fork/clone flags if possible
	var forked string
	if ppid != 1 && comm != "systemd" {
		forked = "forked"
	} else {
		forked = "not-forked"
	}

	startedAt := bootTime().Add(time.Duration(startTicks) * time.Second / ticksPerSecond())

	// Health: zombie/stopped
	switch state {
	case "Z":
		health = "zombie"
	case "T":
		health = "stopped"
	}

	// High CPU/memory (simple: >80% of total)
	utime, _ := strconv.ParseFloat(fields[11], 64)
	stime, _ := strconv.ParseFloat(fields[12], 64)
	rssPages, _ := strconv.ParseFloat(fields[21], 64)
	clkTck := float64(ticksPerSecond())
	totalCPU := (utime + stime) / clkTck
	if totalCPU > 60*60*2 { // >2h CPU time
		health = "high-cpu"
	}
	pageSize := float64(os.Getpagesize())
	memBytes := rssPages * pageSize
	memMB := memBytes / (1024 * 1024)
	if memMB > 1024 {
		health = "high-mem"
	}

	user := readUser(pid)

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
	// Full command line
	cmdline := ""
	cmdlineBytes, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err == nil {
		cmd := strings.ReplaceAll(string(cmdlineBytes), "\x00", " ")
		cmdline = strings.TrimSpace(cmd)
	}

	if comm == "docker-proxy" && container == "" {
		container = resolveDockerProxyContainer(cmdline)
	}

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

// The kernel emits the state immediately after the command, so fields[0] always carries it.
func processState(fields []string) string {
	if len(fields) == 0 {
		return ""
	}
	state := fields[0]
	if len(state) == 0 {
		return ""
	}
	return state[:1]
}
