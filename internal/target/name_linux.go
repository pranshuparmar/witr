//go:build linux

package target

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func ResolveName(name string) ([]int, error) {
	var procPIDs []int

	// Process name and command line matching (case-insensitive, substring)
	entries, _ := os.ReadDir("/proc")
	lowerName := strings.ToLower(name)
	selfPid := os.Getpid()
	parentPid := os.Getppid()
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		// Prevent matching the PID itself as a name
		if lowerName == strconv.Itoa(pid) {
			continue
		}

		// Exclude self and parent (witr, go run, etc.)
		if pid == selfPid || pid == parentPid {
			continue
		}

		comm, err := os.ReadFile("/proc/" + e.Name() + "/comm")
		if err == nil {
			if strings.Contains(strings.ToLower(strings.TrimSpace(string(comm))), lowerName) {
				// Exclude grep-like processes
				if !strings.Contains(strings.ToLower(string(comm)), "grep") {
					procPIDs = append(procPIDs, pid)
				}
				continue
			}
		}

		cmdline, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err == nil {
			// cmdline is null-separated
			cmd := strings.ReplaceAll(string(cmdline), "\x00", " ")
			// Exclude self, parent, and grep
			if strings.Contains(strings.ToLower(cmd), lowerName) &&
				!strings.Contains(strings.ToLower(cmd), "grep") {
				procPIDs = append(procPIDs, pid)
			}
		}
	}

	// Service detection (systemd)
	servicePID, _ := resolveSystemdServiceMainPID(name)

	// Merge and dedupe matches, keeping service PID first.
	seen := map[int]bool{}
	var procUnique []int
	for _, pid := range procPIDs {
		if pid == servicePID || seen[pid] {
			continue
		}
		seen[pid] = true
		procUnique = append(procUnique, pid)
	}
	sort.Ints(procUnique)

	var pids []int
	if servicePID > 0 {
		pids = append(pids, servicePID)
	}
	pids = append(pids, procUnique...)

	if len(pids) == 0 {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}
	return pids, nil
}

// resolveSystemdServiceMainPID tries to resolve a systemd service and returns its MainPID if running.
func resolveSystemdServiceMainPID(name string) (int, error) {
	// Accept both foo and foo.service
	svcName := name
	if !strings.HasSuffix(svcName, ".service") {
		svcName += ".service"
	}
	out, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", "--", svcName).Output()
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(out))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid == 0 {
		return 0, fmt.Errorf("service %q not running", svcName)
	}
	return pid, nil
}
