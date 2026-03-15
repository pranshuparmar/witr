//go:build windows

package target

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func ResolvePort(port int) ([]int, error) {
	// netstat -ano
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	portStr := fmt.Sprintf(":%d", port)
	var pids []int
	seen := make(map[int]bool)

	for _, line := range lines {
		if !strings.Contains(line, portStr) {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToUpper(fields[0])
		localAddr := fields[1]
		if !strings.HasSuffix(localAddr, portStr) {
			continue
		}

		var pid int
		if strings.HasPrefix(proto, "TCP") {
			// TCP: Proto LocalAddr ForeignAddr State PID
			if len(fields) < 5 || fields[3] != "LISTENING" {
				continue
			}
			pid, _ = strconv.Atoi(fields[4])
		} else if strings.HasPrefix(proto, "UDP") {
			// UDP: Proto LocalAddr *:* PID (no state column)
			if len(fields) < 4 {
				continue
			}
			pid, _ = strconv.Atoi(fields[3])
		} else {
			continue
		}

		if pid != 0 && !seen[pid] {
			pids = append(pids, pid)
			seen[pid] = true
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process found listening on port %d", port)
	}
	return pids, nil
}
