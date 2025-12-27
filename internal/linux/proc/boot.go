//go:build linux

package proc

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

func bootTime() time.Time {
	f, _ := os.Open("/proc/stat")
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime") {
			parts := strings.Fields(line)
			sec, _ := strconv.ParseInt(parts[1], 10, 64)
			return time.Unix(sec, 0)
		}
	}
	return time.Now()
}

func ticksPerSecond() time.Duration {
	return 100 // Linux default; portable enough for now
}
