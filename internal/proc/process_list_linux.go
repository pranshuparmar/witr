//go:build linux

package proc

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// listProcessSnapshot collects a lightweight view of running processes
// for child/descendant discovery. We avoid full ReadProcess calls to keep
// this path fast and to reduce permission-sensitive reads.
func listProcessSnapshot() ([]model.Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	processes := make([]model.Process, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		stat, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}

		proc, err := parseStatSnapshot(pid, stat)
		if err != nil {
			continue
		}

		processes = append(processes, proc)
	}

	return processes, nil
}

func parseStatSnapshot(pid int, stat []byte) (model.Process, error) {
	raw := string(stat)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open == -1 || close == -1 || close <= open {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	comm := raw[open+1 : close]
	fields := strings.Fields(raw[close+2:])
	if len(fields) < 2 {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return model.Process{}, fmt.Errorf("invalid ppid")
	}

	return model.Process{
		PID:     pid,
		PPID:    ppid,
		Command: comm,
	}, nil
}
