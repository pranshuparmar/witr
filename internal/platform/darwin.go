//go:build darwin

package platform

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func init() {
	Current = &darwinPlatform{}
}

type darwinPlatform struct{}

func (d *darwinPlatform) ReadProcess(pid int) (model.Process, error) {
	// Get basic process info via ps
	// Format: ppid, comm, user, lstart (multi-word), args (rest)
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "ppid=,comm=,user=,lstart=,args=")
	out, err := cmd.Output()
	if err != nil {
		return model.Process{}, fmt.Errorf("process %d not found", pid)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) == 0 {
		return model.Process{}, fmt.Errorf("process %d not found", pid)
	}

	line := strings.TrimSpace(lines[0])
	fields := strings.Fields(line)
	if len(fields) < 8 {
		return model.Process{}, fmt.Errorf("unexpected ps output format")
	}

	ppid, _ := strconv.Atoi(fields[0])
	comm := fields[1]
	// comm= on macOS may return full path - extract basename
	if idx := strings.LastIndex(comm, "/"); idx != -1 {
		comm = comm[idx+1:]
	}
	user := fields[2]

	// lstart format: "Mon Dec 26 10:30:00 2025" (5 fields)
	lstartStr := strings.Join(fields[3:8], " ")
	startTime, err := time.Parse("Mon Jan 2 15:04:05 2006", lstartStr)
	if err != nil {
		startTime = time.Now()
	}

	// args is everything after lstart
	cmdline := ""
	if len(fields) > 8 {
		cmdline = strings.Join(fields[8:], " ")
	} else {
		cmdline = comm
	}

	// Get working directory via lsof
	cwd := d.getWorkingDir(pid)

	// Git detection (same logic as Linux)
	gitRepo, gitBranch := detectGit(cwd)

	// Listening ports via lsof
	ports, addrs := d.getListeningPorts(pid)

	return model.Process{
		PID:            pid,
		PPID:           ppid,
		Command:        comm,
		Cmdline:        cmdline,
		StartedAt:      startTime,
		User:           user,
		WorkingDir:     cwd,
		GitRepo:        gitRepo,
		GitBranch:      gitBranch,
		Container:      "", // Skip on macOS
		Service:        "", // No systemd on macOS
		ListeningPorts: ports,
		BindAddresses:  addrs,
		Health:         "healthy", // Simplified for macOS
		Forked:         "unknown", // Skip on macOS
		Env:            nil,       // Skip on macOS
	}, nil
}

func (d *darwinPlatform) getWorkingDir(pid int) string {
	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "cwd") {
			fields := strings.Fields(line)
			// lsof format: COMMAND PID USER FD TYPE DEVICE SIZE/OFF NODE NAME
			// cwd line has NAME as the last field
			if len(fields) >= 9 {
				return fields[len(fields)-1]
			}
		}
	}
	return "unknown"
}

func (d *darwinPlatform) getListeningPorts(pid int) ([]int, []string) {
	// Use lsof with -a (AND) to require both -p and -i conditions
	cmd := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-i", "-P", "-n")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	var ports []int
	var addrs []string
	seen := make(map[string]bool) // dedup

	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "LISTEN") {
			continue
		}

		fields := strings.Fields(line)
		// Find the field that looks like *:PORT or IP:PORT
		for _, f := range fields {
			if !strings.Contains(f, ":") || strings.HasPrefix(f, "(") {
				continue
			}
			parts := strings.Split(f, ":")
			if len(parts) < 2 {
				continue
			}
			port, err := strconv.Atoi(parts[len(parts)-1])
			if err != nil {
				continue
			}
			addr := strings.Join(parts[:len(parts)-1], ":")
			if addr == "*" {
				addr = "0.0.0.0"
			}
			key := fmt.Sprintf("%s:%d", addr, port)
			if !seen[key] {
				seen[key] = true
				ports = append(ports, port)
				addrs = append(addrs, addr)
			}
		}
	}

	return ports, addrs
}

func detectGit(cwd string) (string, string) {
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
			var gitBranch string
			if head, err := os.ReadFile(gitDir + "/HEAD"); err == nil {
				headStr := strings.TrimSpace(string(head))
				if strings.HasPrefix(headStr, "ref: ") {
					ref := strings.TrimPrefix(headStr, "ref: ")
					refParts := strings.Split(ref, "/")
					gitBranch = refParts[len(refParts)-1]
				}
			}
			return gitRepo, gitBranch
		}

		idx := strings.LastIndex(searchDir, "/")
		if idx <= 0 {
			break
		}
		searchDir = searchDir[:idx]
	}

	return "", ""
}

func (d *darwinPlatform) ListPIDs() ([]int, error) {
	cmd := exec.Command("ps", "-axo", "pid=")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, line := range strings.Split(string(out), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func (d *darwinPlatform) BootTime() time.Time {
	cmd := exec.Command("sysctl", "-n", "kern.boottime")
	out, err := cmd.Output()
	if err != nil {
		return time.Now()
	}

	// Parse "{ sec = 1735200000, usec = 0 } ..."
	re := regexp.MustCompile(`sec\s*=\s*(\d+)`)
	matches := re.FindStringSubmatch(string(out))
	if len(matches) >= 2 {
		if sec, err := strconv.ParseInt(matches[1], 10, 64); err == nil {
			return time.Unix(sec, 0)
		}
	}
	return time.Now()
}

// ResolvePort finds PIDs listening on a port via lsof.
func (d *darwinPlatform) ResolvePort(port int) ([]int, error) {
	cmd := exec.Command("lsof", "-i", fmt.Sprintf(":%d", port), "-t")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	var pids []int
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		pid, err := strconv.Atoi(strings.TrimSpace(line))
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	// Return lowest PID (main listener)
	sort.Ints(pids)
	return []int{pids[0]}, nil
}

// ResolveName finds PIDs by process name via ps + launchctl.
func (d *darwinPlatform) ResolveName(name string) ([]int, error) {
	var procPIDs []int
	lowerName := strings.ToLower(name)
	selfPid := os.Getpid()
	parentPid := os.Getppid()

	// Try launchctl first for services
	if pid := d.resolveLaunchdService(name); pid > 0 {
		return []int{pid}, nil
	}

	// Scan via ps -axo pid=,comm=,args=
	cmd := exec.Command("ps", "-axo", "pid=,comm=,args=")
	out, _ := cmd.Output()

	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil || pid == selfPid || pid == parentPid {
			continue
		}

		comm := strings.ToLower(fields[1])
		args := strings.ToLower(strings.Join(fields[2:], " "))

		if strings.Contains(comm, lowerName) || strings.Contains(args, lowerName) {
			if !strings.Contains(args, "grep") && !strings.Contains(args, "witr") {
				procPIDs = append(procPIDs, pid)
			}
		}
	}

	if len(procPIDs) == 0 {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}

	// Handle ambiguity
	if len(procPIDs) > 1 {
		fmt.Printf("Ambiguous target: \"%s\"\n\n", name)
		fmt.Println("The name matches multiple processes:")
		fmt.Println()
		for i, pid := range procPIDs {
			fmt.Printf("[%d] PID %d\n", i+1, pid)
		}
		fmt.Println()
		fmt.Println("witr cannot determine intent safely.")
		fmt.Println("Please re-run with an explicit PID:")
		fmt.Println("  witr --pid <pid>")
		os.Exit(1)
	}

	return procPIDs, nil
}

func (d *darwinPlatform) resolveLaunchdService(name string) int {
	// launchctl list outputs: PID, Status, Label
	cmd := exec.Command("launchctl", "list")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	lowerName := strings.ToLower(name)
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		label := strings.ToLower(fields[2])
		if strings.Contains(label, lowerName) {
			pid, err := strconv.Atoi(fields[0])
			if err == nil && pid > 0 {
				return pid
			}
		}
	}
	return 0
}
