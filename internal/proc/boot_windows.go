//go:build windows

package proc

import (
	"os/exec"
	"strings"
	"time"
)

func bootTime() time.Time {
	// wmic os get lastbootuptime
	out, err := exec.Command("wmic", "os", "get", "lastbootuptime").Output()
	if err != nil {
		return time.Now()
	}
	// Output format:
	// LastBootUpTime
	// 20231025123456.123456+120
	lines := strings.Split(string(out), "\n")
	if len(lines) < 2 {
		return time.Now()
	}
	val := strings.TrimSpace(lines[1])
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
