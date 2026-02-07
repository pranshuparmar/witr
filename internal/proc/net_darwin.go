//go:build darwin

package proc

import (
	"strconv"
	"strings"
)

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
