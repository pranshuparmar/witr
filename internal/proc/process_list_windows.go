//go:build windows

package proc

import (
	"encoding/csv"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListProcesses returns a list of all running processes with basic details (PID, Command, State).
// This is used by the TUI to display the process list.
func ListProcesses() ([]model.Process, error) {
	// TODO: Enrich this with more data (User, Memory, CPU) for the TUI
	return listProcessSnapshot()
}

// listProcessSnapshot collects a lightweight view of running processes
// for child/descendant discovery.
func listProcessSnapshot() ([]model.Process, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "Get-CimInstance -ClassName Win32_Process | Select-Object Name,ParentProcessId,ProcessId | ConvertTo-Csv -NoTypeInformation")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("powershell process list: %w", err)
	}

	r := csv.NewReader(strings.NewReader(string(out)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse powershell output: %w", err)
	}

	if len(records) < 2 {
		return []model.Process{}, nil
	}

	headers := records[0]
	nameIdx := -1
	ppidIdx := -1
	pidIdx := -1

	for i, h := range headers {
		switch h {
		case "Name":
			nameIdx = i
		case "ParentProcessId":
			ppidIdx = i
		case "ProcessId":
			pidIdx = i
		}
	}

	if nameIdx == -1 || ppidIdx == -1 || pidIdx == -1 {
		// Fallback to hardcoded indices if header parsing fails or is unexpected
		return nil, fmt.Errorf("invalid powershell output headers: %v", headers)
	}

	processes := make([]model.Process, 0, len(records)-1)
	for _, record := range records[1:] {
		if len(record) <= pidIdx || len(record) <= ppidIdx || len(record) <= nameIdx {
			continue
		}

		pid, err := strconv.Atoi(record[pidIdx])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(record[ppidIdx])
		if err != nil {
			continue
		}
		name := record[nameIdx]

		processes = append(processes, model.Process{
			PID:     pid,
			PPID:    ppid,
			Command: name,
		})
	}

	return processes, nil
}
