//go:build darwin

package target

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// ResolveBoundPort returns every process that owns a TCP listener or UDP
// socket bound to port. Unlike ResolvePort, it never falls back to processes
// that merely have an outbound connection to the port.
func ResolveBoundPort(port int) ([]int, error) {
	if _, err := exec.LookPath("lsof"); err != nil {
		return nil, fmt.Errorf("find lsof: %w", err)
	}
	pids := make(map[int]struct{})
	commands := [][]string{
		{"-i", fmt.Sprintf("TCP:%d", port), "-s", "TCP:LISTEN", "-n", "-P", "-t"},
		{"-i", fmt.Sprintf("UDP:%d", port), "-n", "-P", "-t"},
	}
	for _, args := range commands {
		out, err := exec.Command("lsof", args...).Output()
		if err != nil {
			continue
		}
		for _, value := range strings.Fields(string(out)) {
			if pid, err := strconv.Atoi(value); err == nil && pid > 0 {
				pids[pid] = struct{}{}
			}
		}
	}

	result := make([]int, 0, len(pids))
	for pid := range pids {
		result = append(result, pid)
	}
	sort.Ints(result)
	if len(result) == 0 {
		return nil, fmt.Errorf("%w %d", ErrPortNotBound, port)
	}
	return result, nil
}
