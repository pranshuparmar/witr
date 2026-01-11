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

// listProcessSnapshot collects a lightweight view of running processes
// for child/descendant discovery.
func listProcessSnapshot() ([]model.Process, error) {
	cmd := exec.Command("wmic", "process", "get", "Name,ParentProcessId,ProcessId", "/format:csv")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("wmic process list: %w", err)
	}

	r := csv.NewReader(strings.NewReader(string(out)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse wmic output: %w", err)
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
		return nil, fmt.Errorf("invalid wmic output headers: %v", headers)
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
