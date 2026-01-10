//go:build windows

package proc

import (
	"os/exec"
	"strings"
	"time"
)

func bootTime() time.Time {
	// powershell Get-CimInstance Win32_OperatingSystem
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "Get-CimInstance -ClassName Win32_OperatingSystem | Select-Object -ExpandProperty LastBootUpTime | Get-Date -Format 'yyyyMMddHHmmss'").Output()
	if err != nil {
		return time.Now()
	}
	// Output format:
	// 20231025123456
	val := strings.TrimSpace(string(out))
	if len(val) < 14 {
		return time.Now()
	}
	// Parse 20231025123456
	t, err := time.Parse("20060102150405", val[:14])
	if err != nil {
		return time.Now()
	}
	return t
}
