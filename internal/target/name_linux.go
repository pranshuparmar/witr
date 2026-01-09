//go:build linux

package target

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
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

	// If all matches are filtered out, treat as no result
	if len(procPIDs) == 0 {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}

	// Service detection (systemd)
	servicePID, serviceErr := resolveSystemdServiceMainPID(name)

	// Ambiguity: both process and service, but only if there are at least two unique PIDs
	uniquePIDs := map[int]bool{}
	if servicePID > 0 {
		uniquePIDs[servicePID] = true
	}
	for _, pid := range procPIDs {
		uniquePIDs[pid] = true
	}
	if len(uniquePIDs) > 1 {
		safeName := output.SanitizeTerminal(name)

		fmt.Printf("Ambiguous target: \"%s\"\n\n", safeName)
		fmt.Println("The name matches multiple entities:")
		fmt.Println()
		// Service entry first
		fmt.Printf("[1] PID %d   %s: master process   (service)\n", servicePID, safeName)
		// Process entries (skip if PID matches servicePID)
		idx := 2
		for _, pid := range procPIDs {
			if pid == servicePID {
				continue
			}
			fmt.Printf("[%d] PID %d   %s: worker process   (manual)\n", idx, pid, safeName)
			idx++
		}
		fmt.Println()
		fmt.Println("witr cannot determine intent safely.")
		fmt.Println("Please re-run with an explicit PID:")
		fmt.Println("  witr --pid <pid>")
		os.Exit(1)
	}

	// Service only
	if servicePID > 0 {
		return []int{servicePID}, nil
	}

	// Process only
	if len(procPIDs) > 0 {
		return procPIDs, nil
	}

	// Neither found
	if serviceErr != nil {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}
	return nil, fmt.Errorf("no running process or service named %q", name)
}

// resolveSystemdServiceMainPID tries to resolve a systemd service and returns its MainPID if running.
func resolveSystemdServiceMainPID(name string) (int, error) {
	// Accept both foo and foo.service
	svcName := name
	if !strings.HasSuffix(svcName, ".service") {
		svcName += ".service"
	}
	out, err := exec.Command("systemctl", "show", svcName, "-p", "MainPID", "--value").Output()
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
