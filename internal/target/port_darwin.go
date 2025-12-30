//go:build darwin

package target

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/proc"
)

func ResolvePort(port int) ([]int, error) {
	// Use lsof to find the process listening on this port
	// -i TCP:<port> = specific TCP port
	// -s TCP:LISTEN = only LISTEN state
	// -n = no hostname resolution
	// -P = no port name resolution
	// -t = terse output (PIDs only)
	out, err := proc.Run("lsof", "-i", fmt.Sprintf("TCP:%d", port), "-s", "TCP:LISTEN", "-n", "-P", "-t")
	if err != nil {
		// Try alternative: netstat + grep
		return resolvePortNetstat(port)
	}

	pidStrs := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(pidStrs) == 0 || pidStrs[0] == "" {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	pidSet := make(map[int]bool)
	for _, pidStr := range pidStrs {
		pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
		if err == nil && pid > 0 {
			pidSet[pid] = true
		}
	}

	// Return the lowest PID (the main listener, not forked children)
	var result []int
	minPID := 0
	for pid := range pidSet {
		if minPID == 0 || pid < minPID {
			minPID = pid
		}
	}
	if minPID > 0 {
		result = append(result, minPID)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("socket found but owning process not detected")
	}

	return result, nil
}

func resolvePortNetstat(port int) ([]int, error) {
	// Fallback using netstat
	// On macOS: netstat -anv -p tcp | grep LISTEN | grep .<port>
	out, err := proc.Run("netstat", "-anv", "-p", "tcp")
	if err != nil {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	portStr := fmt.Sprintf(".%d", port)
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		if !strings.Contains(line, portStr) {
			continue
		}

		// netstat -anv format includes PID in the last column
		fields := strings.Fields(line)
		if len(fields) >= 9 {
			// The PID is typically in the 9th field
			pid, err := strconv.Atoi(fields[8])
			if err == nil && pid > 0 {
				return []int{pid}, nil
			}
		}
	}

	return nil, fmt.Errorf("no process listening on port %d", port)
}
