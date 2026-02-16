//go:build freebsd

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetResourceContext returns resource usage context for a process
// FreeBSD implementation - basic support
func GetResourceContext(pid int) *model.ResourceContext {
	// FreeBSD doesn't have macOS-style power assertions or thermal monitoring
	// Could potentially check CPU temperature via sysctl dev.cpu.*.temperature
	// but this is not process-specific

	ctx := &model.ResourceContext{}

	out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "%cpu,rss").Output()
	if err == nil {
		lines := strings.Split(strings.TrimSpace(string(out)), "\n")
		if len(lines) > 0 {
			fields := strings.Fields(lines[len(lines)-1])
			if len(fields) >= 2 {
				if cpu, err := strconv.ParseFloat(fields[0], 64); err == nil {
					ctx.CPUUsage = cpu
				}
				if rssKB, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
					ctx.MemoryUsage = rssKB * 1024
				}
			}
		}
	}

	if ctx.CPUUsage > 0 || ctx.MemoryUsage > 0 {
		return ctx
	}

	return nil
}
