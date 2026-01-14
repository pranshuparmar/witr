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
	seen := make(map[string]bool)

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
		// specialized handling for [::] or [::1] on windows to avoid double bracket
		if len(ip) > 2 && strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
			ip = ip[1 : len(ip)-1]
		}

		port, err := strconv.Atoi(portStr)
		if err == nil {
			key := ip + ":" + portStr
			if !seen[key] {
				ports = append(ports, port)
				addrs = append(addrs, ip)
				seen[key] = true
			}
		}
	}
	return ports, addrs
}
