package proc

import (
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ResolveChildren returns the direct child processes for the provided PID.
// Windows implementation using direct powershell query instead of global snapshot.
func ResolveChildren(pid int) ([]model.Process, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid")
	}

	children := make([]model.Process, 0)

	// powershell Get-CimInstance Win32_Process
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", fmt.Sprintf("Get-CimInstance -ClassName Win32_Process -Filter \"ParentProcessId=%d\" | Select-Object Name,ProcessId | ConvertTo-Csv -NoTypeInformation", pid))
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell child query: %w", err)
	}

	// Parse CSV output cleanly
	// "Name","ProcessId"
	// "chrome.exe","1234"
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "\"Name\"") {
			continue
		}

		parts := strings.Split(line, ",")
		// "chrome.exe","1234"
		if len(parts) >= 2 {
			pidStr := strings.Trim(parts[len(parts)-1], "\"")
			name := strings.Trim(parts[len(parts)-2], "\"")

			cpid, err := strconv.Atoi(pidStr)
			if err != nil {
				continue
			}

			children = append(children, model.Process{
				PID:     cpid,
				PPID:    pid,
				Command: name,
			})
		}
	}

	sortProcesses(children)
	return children, nil
}

func sortProcesses(processes []model.Process) {
	sort.Slice(processes, func(i, j int) bool {
		return processes[i].PID < processes[j].PID
	})
}
