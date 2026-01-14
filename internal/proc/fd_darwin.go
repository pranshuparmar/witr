//go:build darwin

package proc

import (
	"os/exec"
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
	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-i", "TCP", "-n", "-P", "-F", "n").Output()
	if err != nil {
		return inodes
	}

	// Parse lsof output
	seen := make(map[string]bool)
	for line := range strings.Lines(string(out)) {
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
				if !seen[inode] {
					seen[inode] = true
					inodes = append(inodes, inode)
				}
			}
		}
	}

	return inodes
}
