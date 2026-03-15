//go:build linux

package source

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// IsSystemdRunning checks whether systemd is actually the running init system.
// This is the canonical check used by sd_booted() in libsystemd.
func IsSystemdRunning() bool {
	_, err := os.Stat("/run/systemd/system")
	return err == nil
}

func detectSystemd(ancestry []model.Process) *model.Source {
	// 1. Verify systemd is actually the init system, not just that PID 1
	// happens to be named "init" (which could be SysVinit, OpenRC, runit, etc.)
	if !IsSystemdRunning() {
		return nil
	}

	// 2. Check if PID 1 is in ancestry
	hasPID1 := false
	for _, p := range ancestry {
		if p.PID == 1 {
			hasPID1 = true
			break
		}
	}

	if !hasPID1 {
		return nil
	}

	// 2. Resolve the unit file for the target process (last in user's request chain)
	targetProc := ancestry[len(ancestry)-1]
	unitFile := resolveUnitFile(targetProc.PID)
	description := resolveUnitDescription(targetProc.PID)

	return &model.Source{
		Type:        model.SourceSystemd,
		Name:        "systemd",
		Description: description,
		UnitFile:    unitFile,
	}
}

func resolveUnitDescription(pid int) string {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return ""
	}

	unitName := getUnitNameFromCgroup(pid)
	if unitName != "" {
		if desc := querySystemdProperty("Description", unitName); desc != "" {
			return desc
		}
	}
	if desc := querySystemdProperty("Description", fmt.Sprintf("%d", pid)); desc != "" {
		return desc
	}
	return ""
}

func resolveUnitFile(pid int) string {
	if _, err := exec.LookPath("systemctl"); err != nil {
		return ""
	}

	unitName := getUnitNameFromCgroup(pid)

	if unitName != "" {
		if path := querySystemdProperty("FragmentPath", unitName); path != "" {
			return path
		}
		if path := querySystemdProperty("SourcePath", unitName); path != "" {
			return path
		}
	}
	if path := querySystemdProperty("FragmentPath", fmt.Sprintf("%d", pid)); path != "" {
		return path
	}
	return querySystemdProperty("SourcePath", fmt.Sprintf("%d", pid))
}

func querySystemdProperty(prop, target string) string {
	cmd := exec.Command("systemctl", "show", "-p", prop, "--value", target)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	path := strings.TrimSpace(string(out))
	if path == "" || strings.Contains(path, "not set") {
		return ""
	}
	return path
}

func getUnitNameFromCgroup(pid int) string {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cgroup", pid))
	if err != nil {
		return ""
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 3 {
			continue
		}
		controllers := parts[1]
		path := parts[2]

		if controllers == "" || strings.Contains(controllers, "systemd") {
			path = strings.TrimSpace(path)
			pathParts := strings.Split(path, "/")

			for i := len(pathParts) - 1; i >= 0; i-- {
				part := pathParts[i]
				if strings.HasSuffix(part, ".service") || strings.HasSuffix(part, ".scope") {
					return part
				}
			}
		}
	}
	return ""
}
