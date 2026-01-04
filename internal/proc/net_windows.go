//go:build windows

package proc

import (
	"os/exec"
	"strconv"
	"strings"
)

func GetListeningPortsForPID(pid int) ([]int, []string) {
	// netstat -ano | findstr LISTENING | findstr <pid>
	// But findstr is not perfect.
	// Better: netstat -ano
	// Parse output.

	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, nil
	}

	lines := strings.Split(string(out), "\n")
	var ports []int
	var addrs []string
	seen := make(map[int]bool)

	pidStr := strconv.Itoa(pid)

	for _, line := range lines {
		fields := strings.Fields(line)
		// Proto Local Address Foreign Address State PID
		// TCP 0.0.0.0:135 0.0.0.0:0 LISTENING 888
		if len(fields) < 5 {
			continue
		}
		if fields[3] != "LISTENING" {
			continue
		}
		if fields[4] != pidStr {
			continue
		}

		localAddr := fields[1]
		// Parse IP:Port
		lastColon := strings.LastIndex(localAddr, ":")
		if lastColon == -1 {
			continue
		}
		portStr := localAddr[lastColon+1:]
		ip := localAddr[:lastColon]

		port, err := strconv.Atoi(portStr)
		if err == nil {
			if !seen[port] {
				ports = append(ports, port)
				seen[port] = true
			}
			addrs = append(addrs, ip)
		}
	}
	return ports, addrs
}
