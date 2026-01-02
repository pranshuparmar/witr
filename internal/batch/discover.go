//go:build linux || darwin

package batch

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// DiscoverPIDs finds all PIDs matching a pattern.
// This is fast (~20ms) - just a single ps call with pattern matching.
// Unlike target.ResolveName, this returns ALL matches without ambiguity checks.
func DiscoverPIDs(pattern string) ([]int, error) {
	var pids []int

	lowerPattern := strings.ToLower(pattern)
	selfPid := os.Getpid()
	parentPid := os.Getppid()

	// Use ps to list all processes
	out, err := exec.Command("ps", "-axo", "pid=,comm=,args=").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	for line := range strings.Lines(string(out)) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		// Exclude self and parent (witr, go run, etc.)
		if pid == selfPid || pid == parentPid {
			continue
		}

		comm := strings.ToLower(fields[1])
		args := ""
		if len(fields) > 2 {
			args = strings.ToLower(strings.Join(fields[2:], " "))
		}

		// Match against command name
		if strings.Contains(comm, lowerPattern) {
			// Exclude grep-like processes
			if !strings.Contains(comm, "grep") && !strings.Contains(comm, "witr") {
				pids = append(pids, pid)
				continue
			}
		}

		// Match against full command line
		if strings.Contains(args, lowerPattern) &&
			!strings.Contains(args, "grep") &&
			!strings.Contains(args, "witr") {
			pids = append(pids, pid)
		}
	}

	return pids, nil
}
