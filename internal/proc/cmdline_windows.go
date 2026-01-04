//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetCmdline returns the command line for a given PID
func GetCmdline(pid int) string {
	// wmic process where processid=PID get commandline
	out, err := exec.Command("wmic", "process", "where", fmt.Sprintf("processid=%d", pid), "get", "commandline").Output()
	if err != nil {
		return "(unknown)"
	}
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return "(unknown)"
	}
	// The first line is header "CommandLine", second is value
	// But wmic output can be messy with empty lines
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" && trimmed != "CommandLine" {
			return trimmed
		}
	}
	return "(unknown)"
}
