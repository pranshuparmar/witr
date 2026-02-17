//go:build darwin

package proc

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ListOpenPorts() ([]model.OpenPort, error) {
	cmd := exec.Command("lsof", "-i", "-P", "-n")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var ports []model.OpenPort
	lines := strings.Split(string(out), "\n")

	startIdx := 0
	if len(lines) > 0 && strings.HasPrefix(lines[0], "COMMAND") {
		startIdx = 1
	}

	for _, line := range lines[startIdx:] {
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}

		pidStr := fields[1]
		pid, err := strconv.Atoi(pidStr)
		if err != nil {
			continue
		}

		protocol := fields[7]
		if protocol != "TCP" && protocol != "UDP" {
			if strings.Contains(line, "TCP") {
				protocol = "TCP"
			} else if strings.Contains(line, "UDP") {
				protocol = "UDP"
			} else {
				protocol = "UNKNOWN"
			}
		}

		nameField := fields[8] // Address:Port
		state := "UNKNOWN"
		if len(fields) > 9 {
			state = strings.Trim(fields[9], "()")
		} else if protocol == "UDP" {
			state = "OPEN"
		}

		addr, port := parseNetstatAddr(nameField)
		if port == 0 {
			lastColon := strings.LastIndex(nameField, ":")
			if lastColon != -1 {
				portStr := nameField[lastColon+1:]
				if p, err := strconv.Atoi(portStr); err == nil {
					port = p
					addr = nameField[:lastColon]
					if addr == "*" {
						addr = "0.0.0.0"
					}
				}
			}
		}

		if port > 0 {
			ports = append(ports, model.OpenPort{
				PID:      pid,
				Port:     port,
				Address:  addr,
				Protocol: protocol,
				State:    state,
			})
		}
	}

	return ports, nil
}

// parseNetstatAddr parses addresses like "*.8080", "127.0.0.1.8080", "[::1].8080"
func parseNetstatAddr(addr string) (string, int) {
	// Handle IPv6 format [::]:port or [::1]:port
	if strings.HasPrefix(addr, "[") {
		// IPv6 format
		bracketEnd := strings.LastIndex(addr, "]")
		if bracketEnd == -1 {
			return "", 0
		}
		ip := addr[1:bracketEnd]
		rest := addr[bracketEnd+1:]
		// rest should be ":port" or ".port"
		if len(rest) > 1 && (rest[0] == ':' || rest[0] == '.') {
			port, err := strconv.Atoi(rest[1:])
			if err == nil {
				if ip == "::" || ip == "" {
					return "::", port
				}
				return ip, port
			}
		}
		return "", 0
	}

	// Handle formats like "*:8080" or "*.8080"
	if strings.HasPrefix(addr, "*") {
		if len(addr) > 1 && (addr[1] == ':' || addr[1] == '.') {
			port, err := strconv.Atoi(addr[2:])
			if err == nil {
				return "0.0.0.0", port
			}
		}
		return "", 0
	}

	// Handle IPv4 format: "127.0.0.1:8080" or "127.0.0.1.8080"
	// Try colon-separated first (standard format)
	if idx := strings.LastIndex(addr, ":"); idx != -1 {
		ip := addr[:idx]
		portStr := addr[idx+1:]
		port, err := strconv.Atoi(portStr)
		if err == nil {
			return ip, port
		}
	}

	// macOS netstat uses dot-separated: "127.0.0.1.8080"
	// Find the last dot and check if what follows is a port
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
