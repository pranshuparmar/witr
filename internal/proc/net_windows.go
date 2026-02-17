package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ListOpenPorts() ([]model.OpenPort, error) {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(out), "\n")
	var ports []model.OpenPort
	seen := make(map[string]bool)

	for _, line := range lines {
		fields := strings.Fields(line)
		// TCP 0.0.0.0:135 0.0.0.0:0 LISTENING 888  (len 5)
		// UDP 0.0.0.0:123 *:*       999            (len 4)

		if len(fields) < 4 {
			continue
		}

		proto := fields[0]
		if proto != "TCP" && proto != "UDP" && proto != "TCPv6" && proto != "UDPv6" {
			continue
		}

		var pidStr, state string
		if len(fields) == 4 {
			pidStr = fields[3]
			state = "LISTEN"
		} else if len(fields) >= 5 {
			pidStr = fields[4]
			state = fields[3]
			if state == "LISTENING" {
				state = "LISTEN"
			}
		}

		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		localAddr := fields[1]
		lastColon := strings.LastIndex(localAddr, ":")
		if lastColon == -1 {
			continue
		}
		portStr := localAddr[lastColon+1:]
		ip := localAddr[:lastColon]
		if len(ip) > 2 && strings.HasPrefix(ip, "[") && strings.HasSuffix(ip, "]") {
			ip = ip[1 : len(ip)-1]
		}

		port, err := strconv.Atoi(portStr)
		if err == nil {
			key := fmt.Sprintf("%d|%d|%s", pid, port, ip)
			if !seen[key] {
				ports = append(ports, model.OpenPort{
					PID:      pid,
					Port:     port,
					Address:  ip,
					Protocol: proto,
					State:    state,
				})
				seen[key] = true
			}
		}
	}
	return ports, nil
}

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
