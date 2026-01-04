//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func GetResourceContext(pid int) *model.ResourceContext {
	// wmic path Win32_PerfFormattedData_PerfProc_Process where IDProcess=PID get PercentProcessorTime,WorkingSetPrivate /format:list
	cmd := exec.Command("wmic", "path", "Win32_PerfFormattedData_PerfProc_Process", "where", fmt.Sprintf("IDProcess=%d", pid), "get", "PercentProcessorTime,WorkingSetPrivate", "/format:list")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var cpu float64
	var mem uint64

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "PercentProcessorTime=") {
			val := strings.TrimPrefix(line, "PercentProcessorTime=")
			c, _ := strconv.ParseFloat(val, 64)
			cpu = c
		} else if strings.HasPrefix(line, "WorkingSetPrivate=") {
			val := strings.TrimPrefix(line, "WorkingSetPrivate=")
			m, _ := strconv.ParseUint(val, 10, 64)
			mem = m
		}
	}

	return &model.ResourceContext{
		CPUUsage:    cpu,
		MemoryUsage: mem,
	}
}
