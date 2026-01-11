//go:build freebsd

package proc

import (
	"os/exec"
	"strconv"
	"strings"
)

// socketsForPID returns socket inodes/identifiers for a given PID
// On FreeBSD, we use sockstat to get this information
func socketsForPID(pid int) []string {
	var inodes []string

	// Use sockstat to find sockets for this PID
	// -P tcp = TCP protocol
	// -p <pid> = specific process
	seen := make(map[string]bool)
	for _, flag := range []string{"-4", "-6"} {
		out, err := exec.Command("sockstat", flag, "-P", "tcp").Output()
		if err != nil {
			continue
		}

		pidStr := strconv.Itoa(pid)

		for line := range strings.Lines(string(out)) {
			fields := strings.Fields(line)
			if len(fields) < 6 {
				continue
			}

			// Skip header
			if fields[0] == "USER" {
				continue
			}

			// Check if this line is for our PID
			if fields[2] != pidStr {
				continue
			}

			localAddr := fields[5]
			proto := fields[4] // tcp4 or tcp6
			address, port := parseSockstatAddr(localAddr, proto)
			if port > 0 {
				// Create pseudo-inode matching the format in readListeningSockets
				inode := pidStr + ":" + strconv.Itoa(port) + ":" + address
				if !seen[inode] {
					seen[inode] = true
					inodes = append(inodes, inode)
				}
			}
		}
	}

	return inodes
}
