//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetCmdline returns the command line for a given PID
func GetCmdline(pid int) string {
	// powershell Get-CimInstance ...
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", fmt.Sprintf("Get-CimInstance -ClassName Win32_Process -Filter \"ProcessId=%d\" | Select-Object -ExpandProperty CommandLine", pid)).Output()
	if err != nil {
		return "(unknown)"
	}
	val := strings.TrimSpace(string(out))
	if val == "" {
		return "(unknown)"
	}
	return val
}
