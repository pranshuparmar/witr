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
	protectedOwner := false
	commands := [][]string{
		{"-i", fmt.Sprintf("TCP:%d", port), "-s", "TCP:LISTEN", "-n", "-P", "-t"},
	}
	for _, args := range commands {
		out, err := exec.Command("lsof", args...).Output()
		if err != nil {
			// lsof exits 1 when no matching file is found. Other failures
			// indicate that the lookup itself did not complete reliably.
			if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
				return nil, fmt.Errorf("query lsof for port %d: %w", port, err)
			}
			continue
		}
		for _, value := range strings.Fields(string(out)) {
			if pid, err := strconv.Atoi(value); err == nil && pid > 0 {
				if pid == 1 {
					protectedOwner = true
					continue
				}
				pids[pid] = struct{}{}
			}
		}
	}
	udpPIDs, udpErr := resolveUDPBoundPIDs(port)
	if udpErr != nil {
		return nil, udpErr
	}
	for pid := range udpPIDs {
		if pid == 1 {
			protectedOwner = true
			continue
		}
		pids[pid] = struct{}{}
	}

	result := make([]int, 0, len(pids))
	for pid := range pids {
		result = append(result, pid)
	}
	sort.Ints(result)
	if len(result) == 0 {
		if protectedOwner {
			return nil, ErrSocketOwnerUnknown
		}
		return nil, fmt.Errorf("%w %d", ErrPortNotBound, port)
	}
	return result, nil
}

// resolveUDPBoundPIDs parses lsof's NAME field instead of relying on its
// port-only filter. The latter also matches a connected UDP client's foreign
// endpoint, which must not be killed when freeing a local port.
func resolveUDPBoundPIDs(port int) (map[int]struct{}, error) {
	result := make(map[int]struct{})
	out, err := exec.Command("lsof", "-n", "-P", "-a", "-i", fmt.Sprintf("UDP:%d", port), "-Fpn").Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			return nil, fmt.Errorf("query lsof UDP sockets for port %d: %w", port, err)
		}
		return result, nil
	}
	return parseUDPBoundFields(port, string(out)), nil
}

func parseUDPBoundFields(port int, output string) map[int]struct{} {
	result := make(map[int]struct{})
	var pid int
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "p") {
			pid, _ = strconv.Atoi(strings.TrimPrefix(line, "p"))
			continue
		}
		if !strings.HasPrefix(line, "n") || pid <= 0 {
			continue
		}
		name := strings.TrimPrefix(line, "n")
		local := name
		if arrow := strings.Index(local, "->"); arrow >= 0 {
			local = local[:arrow]
		}
		if endpointPort(local) == port {
			result[pid] = struct{}{}
		}
	}
	return result
}

func endpointPort(endpoint string) int {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return 0
	}
	if idx := strings.LastIndex(endpoint, ":"); idx >= 0 {
		port, _ := strconv.Atoi(strings.Trim(endpoint[idx+1:], "[]"))
		return port
	}
	return 0
}
