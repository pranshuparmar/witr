//go:build linux

package proc

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

func readUser(pid int) string {
	path := "/proc/" + strconv.Itoa(pid)

	info, err := os.Stat(path)
	if err != nil {
		return "unknown"
	}

	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return "unknown"
	}

	uid := int(stat.Uid)
	if uid == 0 {
		return "root"
	}
	// Try to resolve username from /etc/passwd
	uidStr := strconv.Itoa(uid)
	passwd, err := os.ReadFile("/etc/passwd")
	if err == nil {
		for line := range strings.Lines(string(passwd)) {
			fields := strings.Split(line, ":")
			if len(fields) > 2 && fields[2] == uidStr {
				return fields[0]
			}
		}
	}
	return uidStr
}
