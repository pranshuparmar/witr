//go:build freebsd

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ReadExtendedInfo reads extended process information for verbose output on FreeBSD.
// It uses `ps` to fetch memory usage and thread counts, similar to macOS.
func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, []int, int, error) {
	var memInfo model.MemoryInfo
	var ioStats model.IOStats
	var fileDescs []string
	var children []int
	var threadCount int
	var fdCount int
	var fdLimit uint64

	// 1. Get Memory info using ps
	// rss = resident set size in 1024 byte blocks
	// vsz = virtual size in 1024 byte blocks
	cmd := exec.Command("ps", "-o", "rss,vsz", "-p", strconv.Itoa(pid))
	out, err := cmd.Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) >= 2 {
			fields := strings.Fields(lines[1])
			if len(fields) >= 2 {
				// RSS
				if rss, err := strconv.ParseUint(fields[0], 10, 64); err == nil {
					memInfo.RSS = rss * 1024
					memInfo.RSSMB = float64(memInfo.RSS) / (1024 * 1024)
				}
				// VSZ
				if vsz, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					memInfo.VMS = vsz * 1024
					memInfo.VMSMB = float64(memInfo.VMS) / (1024 * 1024)
				}
			}
		}
	}

	// 2. Count threads using ps -H (threads) check
	// `ps -H`
	threadCmd := exec.Command("ps", "-H", "-p", strconv.Itoa(pid))
	if threadOut, err := threadCmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(threadOut)), "\n")
		if len(lines) > 1 {
			threadCount = len(lines) - 1
		}
	}

	// 3. Get file descriptors using lsof (best effort)
	fdCmd := exec.Command("sh", "-c", fmt.Sprintf("lsof -p %d | wc -l", pid))
	if fdOut, err := fdCmd.Output(); err == nil {
		str := strings.TrimSpace(string(fdOut))
		if count, err := strconv.Atoi(str); err == nil {
			if count > 0 {
				fdCount = count - 1
			}
		}
	}

	// 4. Children resolution using pgrep
	childCmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
	if childOut, err := childCmd.Output(); err == nil {
		lines := strings.Split(strings.TrimSpace(string(childOut)), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			if cpid, err := strconv.Atoi(line); err == nil {
				children = append(children, cpid)
			}
		}
	}

	return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, nil
}
