package process

import (
	"fmt"

	"github.com/pranshuparmar/witr/internal/platform"
	"github.com/pranshuparmar/witr/pkg/model"
)

func BuildAncestry(pid int) ([]model.Process, error) {
	var chain []model.Process
	seen := make(map[int]bool)

	current := pid

	for current > 0 {
		if seen[current] {
			break // loop protection
		}
		seen[current] = true

		p, err := platform.Current.ReadProcess(current)
		if err != nil {
			break
		}

		chain = append([]model.Process{p}, chain...)

		if p.PPID == 0 || p.PID == 1 {
			break
		}
		current = p.PPID
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("no process ancestry found")
	}

	return chain, nil
}
