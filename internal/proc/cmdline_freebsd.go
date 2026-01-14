//go:build freebsd

package proc

import (
	"os/exec"
	"strconv"
	"strings"
)

// GetCmdline returns the command line for a given PID
func GetCmdline(pid int) string {
	// FreeBSD syntax: ps -p <pid> -o args
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "args").Output()
	if err != nil {
		return "(unknown)"
	}

	// Skip header line
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return "(unknown)"
	}

	cmdline := strings.TrimSpace(lines[1])
	if cmdline == "" {
		return "(unknown)"
	}
	return cmdline
}
