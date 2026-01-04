//go:build windows

package target

import (
	"os/exec"
	"strconv"
	"strings"
)

func ResolveName(name string) ([]int, error) {
	// wmic process get ProcessId,Name,CommandLine /format:list
	out, err := exec.Command("wmic", "process", "get", "ProcessId,Name,CommandLine", "/format:list").Output()
	if err != nil {
		return nil, err
	}

	var pids []int
	lowerName := strings.ToLower(name)
	lines := strings.Split(string(out), "\n")

	var currentPID int
	var currentName string
	var currentCmd string

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
				if strings.Contains(strings.ToLower(currentName), lowerName) ||
					strings.Contains(strings.ToLower(currentCmd), lowerName) {
					pids = append(pids, currentPID)
				}
			}
			// Reset
			currentPID = 0
			currentName = ""
			currentCmd = ""
		}
	}

	return pids, nil
}
