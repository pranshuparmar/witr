//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strings"
)

func readUser(pid int) string {
	// wmic process where processid=PID call getowner
	out, err := exec.Command("wmic", "process", "where", fmt.Sprintf("processid=%d", pid), "call", "getowner").Output()
	if err != nil {
		return "unknown"
	}

	output := string(out)
	var user, domain string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "User =") {
			val := strings.TrimPrefix(line, "User =")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, ";")
			val = strings.Trim(val, "\"")
			user = val
		}
		if strings.HasPrefix(line, "Domain =") {
			val := strings.TrimPrefix(line, "Domain =")
			val = strings.TrimSpace(val)
			val = strings.Trim(val, ";")
			val = strings.Trim(val, "\"")
			domain = val
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
