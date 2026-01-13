//go:build darwin

package proc

import (
	"strconv"
	"strings"
)

// GetCmdline returns the command line for a given PID
func GetCmdline(pid int) string {
	out, err := executor.Run("ps", "-p", strconv.Itoa(pid), "-o", "args=")
	if err != nil {
		return "(unknown)"
	}
	cmdline := strings.TrimSpace(string(out))
	if cmdline == "" {
		return "(unknown)"
	}
	return cmdline
}
