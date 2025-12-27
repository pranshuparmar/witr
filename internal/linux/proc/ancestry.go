//go:build linux

package proc

import (
	"github.com/pranshuparmar/witr/pkg/model"
)

func ResolveAncestry(pid int) ([]model.Process, error) {
	var chain []model.Process
	seen := make(map[int]bool)

	current := pid

	for current > 0 {
		if seen[current] {
			break // loop protection
		}
		seen[current] = true

		p, err := ReadProcess(current)
		if err != nil {
			break
		}

		chain = append(chain, p)

		// pid 1 is always the root
		if p.PID == 1 || p.PPID == 0 {
			break
		}

		current = p.PPID
	}

	return reverse(chain), nil
}

func reverse(in []model.Process) []model.Process {
	for i, j := 0, len(in)-1; i < j; i, j = i+1, j-1 {
		in[i], in[j] = in[j], in[i]
	}
	return in
}
