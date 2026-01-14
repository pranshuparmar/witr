//go:build linux || darwin || freebsd

package proc

import (
	"fmt"
	"sort"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ResolveChildren returns the direct child processes for the provided PID.
func ResolveChildren(pid int) ([]model.Process, error) {
	if pid <= 0 {
		return nil, fmt.Errorf("invalid pid")
	}

	processes, err := listProcessSnapshot()
	if err != nil {
		return nil, err
	}

	children := make([]model.Process, 0)
	for _, proc := range processes {
		if proc.PPID == pid {
			children = append(children, proc)
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
