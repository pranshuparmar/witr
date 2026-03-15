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
	// Verify systemd is actually the init system, not just that PID 1
	// happens to be named "init" (which could be SysVinit, OpenRC, runit, etc.)
	if !IsSystemdRunning() {
		return nil
	}

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

	targetProc := ancestry[len(ancestry)-1]
	props := resolveSystemdProperties(targetProc.PID)

	// Keep only supplemental details (top-level fields already hold name/desc/unitfile)
	details := map[string]string{}
	if v := props["NRestarts"]; v != "" {
		details["NRestarts"] = v
	}

	return &model.Source{
		Type:        model.SourceSystemd,
		Name:        props["UnitName"],
		Description: props["Description"],
		UnitFile:    props["UnitFile"],
		Details:     details,
	}
}

// resolveSystemdProperties fetches Description, FragmentPath/SourcePath, and NRestarts
// in a single systemctl call to avoid spawning multiple processes.
func resolveSystemdProperties(pid int) map[string]string {
	result := map[string]string{}

	if _, err := exec.LookPath("systemctl"); err != nil {
		return result
	}

	unitName := getUnitNameFromCgroup(pid)
	if unitName != "" {
		result["UnitName"] = unitName
	}

	// Try cgroup-resolved unit name first, fall back to PID-based lookup
	targets := []string{}
	if unitName != "" {
		targets = append(targets, unitName)
	}
	targets = append(targets, fmt.Sprintf("%d", pid))

	props := []string{"Description", "FragmentPath", "SourcePath", "NRestarts"}

	for _, target := range targets {
		values := querySystemdProperties(props, target)

		if result["Description"] == "" && values["Description"] != "" {
			result["Description"] = values["Description"]
		}
		if result["UnitFile"] == "" {
			if values["FragmentPath"] != "" {
				result["UnitFile"] = values["FragmentPath"]
			} else if values["SourcePath"] != "" {
				result["UnitFile"] = values["SourcePath"]
			}
		}
		if result["NRestarts"] == "" && values["NRestarts"] != "" {
			result["NRestarts"] = values["NRestarts"]
		}

		// Stop once we have all the info we need
		if result["Description"] != "" && result["UnitFile"] != "" && result["NRestarts"] != "" {
			break
		}
	}

	return result
}

// querySystemdProperties fetches multiple properties in a single systemctl invocation.
func querySystemdProperties(props []string, target string) map[string]string {
	args := []string{"show"}
	for _, p := range props {
		args = append(args, "-p", p)
	}
	args = append(args, "--", target)

	cmd := exec.Command("systemctl", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	result := make(map[string]string, len(props))
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" || strings.Contains(v, "not set") {
			continue
		}
		result[k] = v
	}
	return result
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
