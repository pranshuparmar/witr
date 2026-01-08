//go:build freebsd

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// listProcessSnapshot collects a lightweight view of running processes
// for child/descendant discovery. We use ps on FreeBSD similar to Darwin.
func listProcessSnapshot() ([]model.Process, error) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,comm=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps process list: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	processes := make([]model.Process, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		command := strings.Join(fields[2:], " ")
		processes = append(processes, model.Process{
			PID:     pid,
			PPID:    ppid,
			Command: command,
		})
	}

	return processes, nil
}
