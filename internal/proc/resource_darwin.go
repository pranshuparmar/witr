//go:build darwin

package proc

import (
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetResourceContext returns resource usage context for a process
func GetResourceContext(pid int) *model.ResourceContext {
	ctx := &model.ResourceContext{}

	// Check if process is preventing sleep
	ctx.PreventsSleep = checkPreventsSleep(pid)

	// Get thermal state
	ctx.ThermalState = getThermalState()

	// Only return if we have meaningful data
	if ctx.PreventsSleep || ctx.ThermalState != "" {
		return ctx
	}

	return nil
}

// checkPreventsSleep checks if a process has sleep prevention assertions
func checkPreventsSleep(pid int) bool {
	// pmset -g assertions shows all power assertions
	out, err := exec.Command("pmset", "-g", "assertions").Output()
	if err != nil {
		return false
	}

	pidStr := strconv.Itoa(pid)

	// Look for lines containing our PID in assertion listings
	// Format varies but typically includes "pid <pid>" or "(<pid>)"
	for line := range strings.Lines(string(out)) {
		// Check if this line references our PID and is a sleep prevention assertion
		if strings.Contains(line, pidStr) {
			lower := strings.ToLower(line)
			if strings.Contains(lower, "preventsystemsleep") ||
				strings.Contains(lower, "preventuseridledisplaysleep") ||
				strings.Contains(lower, "preventuseridlesystemsleep") ||
				strings.Contains(lower, "nosleep") {
				return true
			}
		}
	}

	return false
}

// getThermalState returns the current thermal pressure state
func getThermalState() string {
	// pmset -g therm shows thermal conditions
	out, err := exec.Command("pmset", "-g", "therm").Output()
	if err != nil {
		return ""
	}

	output := string(out)

	// Parse thermal state from output
	// Look for "CPU_Speed_Limit" or thermal pressure indicators
	if strings.Contains(output, "CPU_Speed_Limit") {
		// Extract the speed limit percentage
		for line := range strings.Lines(output) {
			if strings.Contains(line, "CPU_Speed_Limit") {
				// Format: CPU_Speed_Limit = 100
				parts := strings.Split(line, "=")
				if len(parts) >= 2 {
					limitStr := strings.TrimSpace(parts[1])
					limit, err := strconv.Atoi(limitStr)
					if err == nil && limit < 100 {
						if limit < 50 {
							return "Heavy throttling"
						} else if limit < 80 {
							return "Moderate throttling"
						} else {
							return "Light throttling"
						}
					}
				}
			}
		}
	}

	// Check for thermal pressure level
	if strings.Contains(output, "Thermal_Level") {
		for line := range strings.Lines(output) {
			if strings.Contains(line, "Thermal_Level") {
				parts := strings.Split(line, "=")
				if len(parts) >= 2 {
					level := strings.TrimSpace(parts[1])
					switch level {
					case "0":
						return "" // Normal, don't show
					case "1":
						return "Moderate thermal pressure"
					case "2":
						return "Heavy thermal pressure"
					default:
						return "Thermal pressure level " + level
					}
				}
			}
		}
	}

	return ""
}

// GetEnergyImpact attempts to get energy impact for a process
// Note: This requires elevated privileges via powermetrics
// Returns empty string if not available
func GetEnergyImpact(pid int) string {
	// powermetrics requires root, so we can't easily get per-process energy
	// Instead, we rely on the prevents-sleep check as a proxy for high energy impact
	// A future enhancement could parse Activity Monitor's energy data via private APIs

	return ""
}
