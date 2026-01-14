//go:build darwin

package target

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func ResolvePort(port int) ([]int, error) {
	// Use lsof to find the process listening on this port
	// -i TCP:<port> = specific TCP port
	// -s TCP:LISTEN = only LISTEN state
	// -n = no hostname resolution
	// -P = no port name resolution
	// -t = terse output (PIDs only)
	out, err := exec.Command("lsof", "-i", fmt.Sprintf("TCP:%d", port), "-s", "TCP:LISTEN", "-n", "-P", "-t").Output()
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

	// collect all owning pids so callers can handle multi-owner sockets
	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		result = append(result, pid)
	}
	sort.Ints(result)

	if len(result) == 0 {
		return nil, fmt.Errorf("socket found but owning process not detected")
	}

	return result, nil
}

func resolvePortNetstat(port int) ([]int, error) {
	// Fallback using netstat
	// On macOS: netstat -anv -p tcp | grep LISTEN | grep .<port>
	out, err := exec.Command("netstat", "-anv", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	portStr := fmt.Sprintf(".%d", port)

	pidSet := make(map[int]bool) // collect matches so we can return all owners
	for line := range strings.Lines(string(out)) {
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
				pidSet[pid] = true
			}
		}
	}

	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		result = append(result, pid)
	}
	sort.Ints(result)
	if len(result) > 0 {
		return result, nil
	}

	return nil, fmt.Errorf("no process listening on port %d", port)
}
