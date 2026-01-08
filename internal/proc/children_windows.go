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
// Windows implementation using direct wmic query instead of global snapshot.
func ResolveChildren(pid int) ([]model.Process, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid")
	}

	children := make([]model.Process, 0)

	// wmic process where ParentProcessId=PID get ProcessId,Name /format:csv
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ParentProcessId=%d", pid), "get", "ProcessId,Name", "/format:csv")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wmic child query: %w", err)
	}

	// Parse CSV output manually to be robust against empty lines/CRLF
	// Output format:
	// Node,Name,ProcessId
	// MYPC,chrome.exe,1234
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "Node") {
			continue
		}

		parts := strings.Split(line, ",")
		// Expecting at least 3 parts: Node, Name, ProcessId (or just Name, ProcessId if Node omitted?)
		// standard wmic /format:csv includes Node.
		// "MYPC","chrome.exe","1234"
		if len(parts) >= 2 {
			// ProcessId is typically the last column
			pidStr := parts[len(parts)-1]
			name := parts[len(parts)-2]

			cpid, err := strconv.Atoi(strings.TrimSpace(pidStr))
			if err != nil {
				continue
			}

			children = append(children, model.Process{
				PID:     cpid,
				PPID:    pid,
				Command: strings.TrimSpace(name),
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
