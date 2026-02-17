//go:build freebsd

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListOpenPorts returns all open ports
func ListOpenPorts() ([]model.OpenPort, error) {
	var openPorts []model.OpenPort
	sockets := make(map[string]model.Socket)

	for _, flag := range []string{"-4", "-6"} {
		out, err := exec.Command("sockstat", flag).Output()
		if err != nil {
			continue
		}

		parseSockstatOutput(string(out), sockets)
	}

	for _, s := range sockets {
		openPorts = append(openPorts, model.OpenPort{
			PID:      extractPID(s.Inode),
			Port:     s.Port,
			Address:  s.Address,
			Protocol: s.Protocol,
			State:    s.State,
		})
	}

	return openPorts, nil
}

func extractPID(inode string) int {
	parts := strings.Split(inode, ":")
	if len(parts) > 0 {
		pid, _ := strconv.Atoi(parts[0])
		return pid
	}
	return 0
}

func readListeningSockets() (map[string]model.Socket, error) {
	ports, err := ListOpenPorts()
	if err != nil {
		return nil, err
	}
	sockets := make(map[string]model.Socket)
	for _, p := range ports {
		if p.State == "LISTEN" {
			inode := fmt.Sprintf("%d:%d:%s", p.PID, p.Port, p.Address)
			sockets[inode] = model.Socket{
				Inode:    inode,
				Port:     p.Port,
				Address:  p.Address,
				State:    p.State,
				Protocol: p.Protocol,
			}
		}
	}
	return sockets, nil
}

func parseSockstatOutput(output string, sockets map[string]model.Socket) {
	for line := range strings.Lines(output) {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		if fields[0] == "USER" {
			continue
		}

		pid := fields[2]
		proto := fields[4] // tcp4, tcp6, udp4, udp6
		localAddr := fields[5]
		foreignAddr := fields[6]

		state := "UNKNOWN"
		protocol := "UNKNOWN"
		if strings.Contains(proto, "tcp") {
			protocol = "TCP"
			if strings.Contains(proto, "6") {
				protocol = "TCP6"
			}

			if foreignAddr == "*:*" || foreignAddr == "0.0.0.0:0" || foreignAddr == "[::]:0" {
				state = "LISTEN"
			} else {
				state = "ESTABLISHED"
			}
		} else if strings.Contains(proto, "udp") {
			protocol = "UDP"
			if strings.Contains(proto, "6") {
				protocol = "UDP6"
			}
			state = "OPEN"
		}

		address, port := parseSockstatAddr(localAddr, proto)
		if port > 0 {
			inode := pid + ":" + strconv.Itoa(port) + ":" + address
			sockets[inode] = model.Socket{
				Inode:    inode,
				Port:     port,
				Address:  address,
				Protocol: protocol,
				State:    state,
			}
		}
	}
}

// parseSockstatAddr parses addresses like "*:80", "127.0.0.1:8080", "[::1]:8080"
// proto is the protocol field from sockstat (tcp4 or tcp6) to distinguish IPv4 vs IPv6
func parseSockstatAddr(addr string, proto string) (string, int) {
	// Handle IPv6 format [::]:port or [::1]:port
	if strings.HasPrefix(addr, "[") {
		bracketEnd := strings.LastIndex(addr, "]")
		if bracketEnd == -1 {
			return "", 0
		}
		ip := addr[1:bracketEnd]
		rest := addr[bracketEnd+1:]
		// rest should be ":port"
		if len(rest) > 1 && rest[0] == ':' {
			port, err := strconv.Atoi(rest[1:])
			if err == nil {
				// Return IPv6 address without brackets for proper formatting with net.JoinHostPort
				return ip, port
			}
		}
		return "", 0
	}

	// Handle wildcard format "*:port"
	// Distinguish between IPv4 and IPv6 based on protocol
	if strings.HasPrefix(addr, "*:") {
		port, err := strconv.Atoi(addr[2:])
		if err == nil {
			// If proto is tcp6, return IPv6 any address with brackets
			if strings.Contains(proto, "6") {
				return "::", port
			}
			// Default to IPv4 any address
			return "0.0.0.0", port
		}
		return "", 0
	}

	// Handle IPv4 format "127.0.0.1:8080"
	// FreeBSD sockstat uses colon as separator
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		ip := addr[:idx]
		portStr := addr[idx+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil {
			if ip == "*" {
				// Check protocol for IPv6 vs IPv4
				if strings.Contains(proto, "6") {
					return "[::]", port
				}
				return "0.0.0.0", port
			}
			// If IP contains colons (IPv6), wrap with brackets
			if strings.Contains(ip, ":") {
				return ip, port
			}
			return ip, port
		}
	}

	// Handle dot-separated format (some FreeBSD versions)
	// "127.0.0.1.8080"
	if idx := strings.LastIndex(addr, "."); idx != -1 {
		portStr := addr[idx+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil {
			ip := addr[:idx]
			return ip, port
		}
	}

	return "", 0
}
