//go:build windows

package target

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

func ResolveBoundPort(port int) ([]int, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, fmt.Errorf("run netstat: %w", err)
	}
	return parseWindowsBoundPortOutput(port, string(out))
}

func parseWindowsBoundPortOutput(port int, output string) ([]int, error) {
	portSuffix := fmt.Sprintf(":%d", port)
	pidSet := make(map[int]struct{})
	protectedOwner := false

	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		proto := strings.ToUpper(fields[0])
		if !strings.HasSuffix(fields[1], portSuffix) {
			continue
		}

		var pidText string
		switch {
		case strings.HasPrefix(proto, "TCP"):
			if len(fields) < 5 || strings.ToUpper(fields[3]) != "LISTENING" {
				continue
			}
			pidText = fields[4]
		case strings.HasPrefix(proto, "UDP"):
			pidText = fields[3]
		default:
			continue
		}

		if pid, err := strconv.Atoi(pidText); err == nil && pid > 0 {
			if pid == 4 {
				protectedOwner = true
				continue
			}
			pidSet[pid] = struct{}{}
		}
	}

	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
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
