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
	return parseOpenPorts(out), nil
}

func parseOpenPorts(out []byte) []model.OpenPort {
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
			state = normalizeWindowsTCPState(fields[3], fields[2])
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
	return ports
}

// GetSocketsForPID returns every IP socket owned by a PID, including
// non-listening sockets, by parsing `netstat -ano`.
func GetSocketsForPID(pid int) []model.Socket {
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	var sockets []model.Socket
	seen := make(map[string]bool)

	pidStr := strconv.Itoa(pid)

	for _, line := range lines {
		fields := strings.Fields(line)
		// TCP:  Proto LocalAddr ForeignAddr State PID  (5 fields)
		// UDP:  Proto LocalAddr *:*         PID        (4 fields)
		if len(fields) < 4 {
			continue
		}

		proto := strings.ToUpper(fields[0])
		var matchPID, state string
		if strings.HasPrefix(proto, "TCP") {
			if len(fields) < 5 {
				continue
			}
			state = normalizeWindowsTCPState(fields[3], fields[2])
			matchPID = fields[4]
		} else if strings.HasPrefix(proto, "UDP") {
			state = "OPEN"
			matchPID = fields[3]
		} else {
			continue
		}

		if matchPID != pidStr {
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
		if err != nil {
			continue
		}
		key := proto + "|" + ip + "|" + portStr + "|" + state
		if seen[key] {
			continue
		}
		seen[key] = true
		sockets = append(sockets, model.Socket{
			Port:     port,
			Address:  ip,
			Protocol: proto,
			State:    state,
		})
	}
	return sockets
}

// normalizeWindowsTCPState maps a netstat state token to the name the rest of
// witr uses.
//
// netstat localizes the token, so a non-English console reports ABHÖREN or
// ECOUTE rather than LISTENING, in the console's OEM code page. The zero-port
// remote endpoint identifies a listener in every locale, but it is not unique
// to one: BOUND and CLOSED rows also carry 0.0.0.0:0. So the English names are
// matched first and the endpoint is only consulted for a token this build does
// not recognize.
func normalizeWindowsTCPState(state, remoteAddr string) string {
	switch state {
	case "LISTENING":
		return "LISTEN"
	case "ESTABLISHED", "TIME_WAIT", "CLOSE_WAIT", "SYN_SENT", "SYN_RECEIVED",
		"FIN_WAIT_1", "FIN_WAIT_2", "LAST_ACK", "CLOSING", "CLOSED", "BOUND", "DELETE_TCB":
		return state
	}
	if strings.HasSuffix(remoteAddr, ":0") {
		return "LISTEN"
	}
	return strings.ToValidUTF8(state, "?")
}
