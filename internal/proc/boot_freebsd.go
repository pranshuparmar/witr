//go:build freebsd

package proc

import (
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func bootTime() time.Time {
	// Use sysctl kern.boottime on FreeBSD (same format as macOS)
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err != nil {
		return time.Now()
	}

	// Output format: { sec = 1703123456, usec = 123456 } ...
	outStr := string(out)
	if idx := strings.Index(outStr, "sec = "); idx != -1 {
		start := idx + 6
		end := strings.Index(outStr[start:], ",")
		if end != -1 {
			secStr := outStr[start : start+end]
			sec, err := strconv.ParseInt(strings.TrimSpace(secStr), 10, 64)
			if err == nil {
				return time.Unix(sec, 0)
			}
		}
	}

	return time.Now()
}

func ticksPerSecond() time.Duration {
	// FreeBSD default (same as Linux/macOS)
	return 100
}
