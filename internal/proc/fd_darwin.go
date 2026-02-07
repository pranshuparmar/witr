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
	// -i TCP = TCP sockets
	// -s TCP:LISTEN = only in LISTEN state
	// -n = don't resolve hostnames
	// -P = don't resolve port names
	out, err := exec.Command("lsof", "-i", "TCP", "-s", "TCP:LISTEN", "-n", "-P", "-F", "pn").Output()
	if err != nil {
		return inodes
	}

	// Parse lsof output
	seen := make(map[string]bool)
	var blocks = strings.Split(string(out), "p")
	for i := range blocks {
		if strings.HasPrefix(blocks[i], strconv.Itoa(pid)+"\n") {
			for line := range strings.Lines(blocks[i]) {
				if len(line) == 0 {
					continue
				}
				if line[0] == 'n' {
					// n<address>
					addr, port := parseNetstatAddr(strings.TrimSpace(line[1:]))
					if port > 0 {
						// Create pseudo-inode matching the format in readListeningSockets
						inode := addr + ":" + strconv.Itoa(port)
						if !seen[inode] {
							seen[inode] = true
							inodes = append(inodes, inode)
						}
					}
				}
			}
			break
		}
	}
	return inodes
}
