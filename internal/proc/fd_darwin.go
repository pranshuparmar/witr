//go:build darwin

package proc

import (
	"strconv"
	"strings"
)

// socketsForPID returns socket inodes/identifiers for a given PID
// On macOS, we use lsof to get this information
func socketsForPID(pid int) []string {
	var inodes []string

	// Use lsof to find sockets for this PID
	// -a = AND logic
	// -p <pid> = specific process
	// -i TCP = TCP sockets
	// -n = don't resolve hostnames
	// -P = don't resolve port names
	out, err := executor.Run("lsof", "-a", "-p", strconv.Itoa(pid), "-i", "TCP", "-n", "-P", "-F", "n")
	if err != nil {
		return inodes
	}

	// Parse lsof output
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if line[0] == 'n' {
			// n<address>
			addr := line[1:]
			_, port := parseNetstatAddr(addr)
			if port > 0 {
				// Create pseudo-inode matching the format in readListeningSockets
				inode := strconv.Itoa(pid) + ":" + strconv.Itoa(port)
				inodes = append(inodes, inode)
			}
		}
	}

	return inodes
}
