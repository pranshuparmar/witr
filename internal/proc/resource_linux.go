//go:build linux

package proc

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetResourceContext returns resource usage context for a process
func GetResourceContext(pid int) *model.ResourceContext {
	ctx := &model.ResourceContext{}
	ctx.PreventsSleep = checkPreventsSleep(pid)
	ctx.ThermalState = getThermalState()
	ctx.AppNapped = getAppNapped(pid)
	ctx.EnergyImpact = GetEnergyImpact(pid)
	return ctx
}

// thermal zone info from /sys/class/thermal
func getThermalState() string {

	path := "/sys/class/thermal/thermal_zone0/temp"
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}
	readText, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	tempstr := strings.TrimSpace(string(readText))
	temp, err := strconv.Atoi(tempstr)
	if err != nil {
		return ""
	}
	tempC := temp / 1000
	switch {
	case tempC > 90:
		return fmt.Sprintf("Critical thermal pressure %d", tempC)
	case tempC > 70:
		return fmt.Sprintf("High thermal pressure %d", tempC)
	case tempC > 60:
		return fmt.Sprintf("Warm thermal state %d", tempC)
	default:
		return fmt.Sprintf("Normal thermal state %d", tempC)
	}
}

// checkPreventsSleep checks if a process has sleep prevention assertions
func checkPreventsSleep(pid int) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "systemd-inhibit", "--list").Output()
	if err != nil {
		return false
	}
	pidStr := strconv.Itoa(pid)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		// Check if this line references our PID and is a sleep prevention assertion
		if !strings.Contains(line, pidStr) {
			continue
		}
		if strings.Contains(line, pidStr) {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "sleep") ||
				strings.Contains(lower, "idle") ||
				strings.Contains(lower, "shutdown") {
				return true
			}
		}
	}
	return false
}

// detect if process is in a stopped/suspended state
func getAppNapped(pid int) bool {
	statFile := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statFile)
	if err != nil {
		return false
	}

	dataStr := string(data)
	lastParenIndex := strings.LastIndex(dataStr, ")")
	if lastParenIndex == -1 || lastParenIndex+2 >= len(dataStr) {
		return false
	}

	rest := dataStr[lastParenIndex+2:]
	fields := strings.Fields(rest)
	if len(fields) < 1 {
		return false
	}

	state := fields[0]
	return state == "T" || state == "t"
}

// GetEnergyImpact attempts to get energy impact for a process
func GetEnergyImpact(pid int, usePs ...bool) string {
	var cpu float64

	shouldUsePs := len(usePs) > 0 && usePs[0]

	if shouldUsePs {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		out, err := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "pcpu=").Output()
		if err != nil {
			return ""
		}

		cpuStr := strings.TrimSpace(string(out))
		if cpuStr == "" {
			return ""
		}

		cpu, err = strconv.ParseFloat(cpuStr, 64)
		if err != nil {
			return ""
		}
	} else {
		// Use top (default)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		out, err := exec.CommandContext(ctx, "top", "-b", "-n", "1", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return ""
		}

		lines := strings.Split(string(out), "\n")
		found := false

		for _, line := range lines {
			if strings.Contains(line, strconv.Itoa(pid)) {
				fields := strings.Fields(line)
				// CPU% is generally the 9th field in top output
				// Output pattern: PID USER PR NI VIRT RES SHR S %CPU %MEM TIME+ COMMAND
				if len(fields) >= 9 {
					cpuStr := strings.TrimSuffix(fields[8], "%")
					cpu, err = strconv.ParseFloat(cpuStr, 64)
					if err == nil {
						found = true
						break
					}
				}
			}
		}

		if !found {
			return ""
		}
	}

	switch {
	case cpu > 50:
		return "Very High"
	case cpu > 25:
		return "High"
	case cpu > 10:
		return "Medium"
	case cpu > 2:
		return "Low"
	case cpu > 0:
		return "Very Low"
	default:
		return ""
	}
}
