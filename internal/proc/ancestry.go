package proc

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ResolveAncestry(pid int) ([]model.Process, error) {
	return resolveAncestry(pid, ReadProcess)
}

func resolveAncestry(pid int, readProcess func(int) (model.Process, error)) ([]model.Process, error) {
	var chain []model.Process
	seen := make(map[int]bool)

	current := pid

	for current > 0 {
		if seen[current] {
			break // loop protection
		}
		seen[current] = true

		p, err := readProcess(current)
		if err != nil {
			break
		}

		// A real parent must have started no later than its child. If the
		// process currently occupying the PPID is newer, the original parent
		// exited and its PID was recycled while (or before) we walked the
		// chain. Stop here instead of stitching an unrelated process onto the
		// ancestry. Some platforms can leave start times unavailable, so only
		// enforce the invariant when both values are known.
		if len(chain) > 0 {
			child := chain[len(chain)-1]
			if !p.StartedAt.IsZero() && !child.StartedAt.IsZero() && p.StartedAt.After(child.StartedAt) {
				break
			}
		}

		chain = append(chain, p)

		if p.PPID == 0 || p.PID == 1 {
			break
		}
		current = p.PPID
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("no process ancestry found")
	}

	// Reverse the chain to get root
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain, nil
}
