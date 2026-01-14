//go:build darwin

package proc

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// readListeningSockets returns a map of pseudo-inodes to sockets
// On macOS, we use lsof to get listening sockets
// We use a combination of PID:port as the "inode" since macOS doesn't expose inodes like Linux
func readListeningSockets() (map[string]model.Socket, error) {
	sockets := make(map[string]model.Socket)

	// Use lsof to get listening TCP sockets
	// -i TCP = only TCP sockets
	// -s TCP:LISTEN = only in LISTEN state
	// -n = don't resolve hostnames
	// -P = don't resolve port names
	out, err := exec.Command("lsof", "-i", "TCP", "-s", "TCP:LISTEN", "-n", "-P", "-F", "pn").Output()
	if err != nil {
		// lsof might fail without root, try netstat as fallback
		return readListeningSocketsNetstat()
	}

	// Parse lsof -F output format
	// p<pid>
	// n<address>
	var currentPID string
	for line := range strings.Lines(string(out)) {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case 'p':
			currentPID = line[1:]
		case 'n':
			// Format: n*:8080 or n127.0.0.1:8080 or n[::1]:8080
			addr := line[1:]
			address, port := parseNetstatAddr(addr)
			if port > 0 {
				// Use PID:port as pseudo-inode
				inode := currentPID + ":" + strconv.Itoa(port)
				sockets[inode] = model.Socket{
					Inode:   inode,
					Port:    port,
					Address: address,
				}
			}
		}
	}

	return sockets, nil
}

func readListeningSocketsNetstat() (map[string]model.Socket, error) {
	sockets := make(map[string]model.Socket)

	// Use netstat as fallback
	out, err := exec.Command("netstat", "-an", "-p", "tcp").Output()
	if err != nil {
		return sockets, nil
	}

	for line := range strings.Lines(string(out)) {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Local address is typically field 3 (0-indexed)
		localAddr := fields[3]
		address, port := parseNetstatAddr(localAddr)
		if port > 0 {
			// Generate a unique key
			inode := "netstat:" + localAddr
			sockets[inode] = model.Socket{
				Inode:   inode,
				Port:    port,
				Address: address,
			}
		}
	}

	return sockets, nil
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
