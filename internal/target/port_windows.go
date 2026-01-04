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
		if strings.Contains(line, portStr) {
			fields := strings.Fields(line)
			if len(fields) < 5 {
				continue
			}
			// Proto Local Address Foreign Address State PID
			localAddr := fields[1]
			if strings.HasSuffix(localAddr, portStr) {
				pidStr := fields[4]
				pid, _ := strconv.Atoi(pidStr)
				if pid != 0 && !seen[pid] {
					pids = append(pids, pid)
					seen[pid] = true
				}
			}
		}
	}

	return pids, nil
}
