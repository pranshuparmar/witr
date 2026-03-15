//go:build windows

package target

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
)

func ResolveName(name string, exact bool) ([]int, error) {
	// powershell Get-CimInstance Win32_Process
	out, err := exec.Command("powershell", "-NoProfile", "-NonInteractive", "Get-CimInstance -ClassName Win32_Process | ForEach-Object { 'Name=' + $_.Name; 'CommandLine=' + $_.CommandLine; 'ProcessId=' + $_.ProcessId }").Output()
	if err != nil {
		return nil, err
	}

	var pids []int
	lowerName := strings.ToLower(name)
	lines := strings.Split(string(out), "\n")

	var currentPID int
	var currentName string
	var currentCmd string

	selfPid := os.Getpid()

	// Resolve own ancestry to exclude parents (sudo, shell, etc.) from matching
	ignoredPids := make(map[int]bool)
	ignoredPids[selfPid] = true
	if ancestry, err := procpkg.ResolveAncestry(selfPid); err == nil {
		for _, p := range ancestry {
			ignoredPids[p.PID] = true
		}
	}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "CommandLine=") {
			currentCmd = strings.TrimPrefix(line, "CommandLine=")
		} else if strings.HasPrefix(line, "Name=") {
			currentName = strings.TrimPrefix(line, "Name=")
		} else if strings.HasPrefix(line, "ProcessId=") {
			val := strings.TrimPrefix(line, "ProcessId=")
			currentPID, _ = strconv.Atoi(val)

			// Check match
			if currentPID != 0 {
				// Exclude self and ancestry
				if ignoredPids[currentPID] {
					// Reset
					currentPID = 0
					currentName = ""
					currentCmd = ""
					continue
				}

				var match bool
				if exact {
					match = strings.ToLower(currentName) == lowerName
					if !match {
						// Check if ANY argument matches exactly
						parts := strings.Fields(currentCmd)
						for _, part := range parts {
							if strings.ToLower(part) == lowerName {
								match = true
								break
							}
						}
					}
				} else {
					match = strings.Contains(strings.ToLower(currentName), lowerName) ||
						strings.Contains(strings.ToLower(currentCmd), lowerName)
				}
				if match {
					pids = append(pids, currentPID)
				}
			}
			// Reset
			currentPID = 0
			currentName = ""
			currentCmd = ""
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process found matching: %s", name)
	}
	return pids, nil
}
