//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strings"
)

func readUser(pid int) string {
	// powershell Get-CimInstance Win32_Process GetOwner
	psScript := fmt.Sprintf("Get-CimInstance -ClassName Win32_Process -Filter \"ProcessId=%d\" | Invoke-CimMethod -MethodName GetOwner | ForEach-Object { 'User=' + $_.User; 'Domain=' + $_.Domain }", pid)
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", psScript).Output()
	if err != nil {
		return "unknown"
	}

	output := string(out)
	var user, domain string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "User=") {
			val := strings.TrimPrefix(line, "User=")
			user = strings.TrimSpace(val)
		}
		if strings.HasPrefix(line, "Domain=") {
			val := strings.TrimPrefix(line, "Domain=")
			domain = strings.TrimSpace(val)
		}
	}

	if user != "" {
		if domain != "" {
			return domain + "\\" + user
		}
		return user
	}
	return "unknown"
}
