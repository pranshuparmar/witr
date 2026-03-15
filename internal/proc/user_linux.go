//go:build linux

package proc

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

var (
	userCache     map[int]string
	userCacheOnce sync.Once
)

func loadUserCache() map[int]string {
	cache := make(map[int]string)
	cache[0] = "root"

	data, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return cache
	}

	for line := range strings.Lines(string(data)) {
		fields := strings.Split(line, ":")
		if len(fields) > 2 {
			if uid, err := strconv.Atoi(fields[2]); err == nil {
				cache[uid] = fields[0]
			}
		}
	}
	return cache
}

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

	userCacheOnce.Do(func() {
		userCache = loadUserCache()
	})

	if name, ok := userCache[uid]; ok {
		return name
	}
	return strconv.Itoa(uid)
}
