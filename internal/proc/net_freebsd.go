//go:build freebsd

package proc

import (
	"os/exec"
	"strconv"
	"strings"
)

// readListeningSockets returns a map of pseudo-inodes to sockets
// On FreeBSD, we use sockstat to get listening sockets
// We use a combination of PID:port as the "inode" since FreeBSD doesn't expose inodes like Linux
func readListeningSockets() (map[string]Socket, error) {
	sockets := make(map[string]Socket)

	// Use sockstat to get listening TCP sockets
	// -4 = IPv4, -6 = IPv6, -l = listening, -P tcp = TCP protocol
	// Run for both IPv4 and IPv6
	for _, flag := range []string{"-4", "-6"} {
		out, err := exec.Command("sockstat", flag, "-l", "-P", "tcp").Output()
		if err != nil {
			continue
		}

		parseSockstatOutput(string(out), sockets)
	}

	if len(sockets) == 0 {
		// Try netstat as fallback
		return readListeningSocketsNetstat()
	}

	return sockets, nil
}

func parseSockstatOutput(output string, sockets map[string]Socket) {
	// sockstat output format:
	// USER     COMMAND    PID   FD PROTO  LOCAL ADDRESS         FOREIGN ADDRESS
	// root     nginx      1234  6  tcp4   *:80                  *:*

	for line := range strings.Lines(output) {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Skip header
		if fields[0] == "USER" {
			continue
		}

		pid := fields[2]
		proto := fields[4] // tcp4 or tcp6
		localAddr := fields[5]

		// Parse local address with protocol information
		address, port := parseSockstatAddr(localAddr, proto)
		if port > 0 {
			// Use PID:port:address as pseudo-inode to distinguish IPv4 and IPv6
			inode := pid + ":" + strconv.Itoa(port) + ":" + address
			sockets[inode] = Socket{
				Inode:   inode,
				Port:    port,
				Address: address,
			}
		}
	}
}

func readListeningSocketsNetstat() (map[string]Socket, error) {
	sockets := make(map[string]Socket)

	// Use netstat as fallback
	// netstat -an -p tcp shows all TCP connections
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
		// First field (0) is proto like "tcp4" or "tcp6"
		proto := ""
		if len(fields) > 0 {
			proto = fields[0]
		}
		localAddr := fields[3]
		address, port := parseSockstatAddr(localAddr, proto)
		if port > 0 {
			// Generate a unique key
			inode := "netstat:" + localAddr
			sockets[inode] = Socket{
				Inode:   inode,
				Port:    port,
				Address: address,
			}
		}
	}

	return sockets, nil
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
