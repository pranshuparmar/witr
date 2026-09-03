package proc

import (
	"fmt"

	"github.com/pranshuparmar/witr/pkg/model"
)

// processReader is the seam tests use to feed deterministic process records
// into ResolveAncestry. Production wires it to ReadProcess; tests substitute a
// stub to drive the recycled-PPID edge cases without touching /proc.
var processReader = ReadProcess

func ResolveAncestry(pid int) ([]model.Process, error) {
	var chain []model.Process
	seen := make(map[int]bool)

	current := pid

	for current > 0 {
		if seen[current] {
			break // loop protection
		}
		seen[current] = true

		p, err := processReader(current)
		if err != nil {
			break
		}

		chain = append(chain, p)

		if p.PPID == 0 || p.PID == 1 {
			break
		}

		// Validate that the PPID we are about to follow is a real ancestor.
		// PIDs are recycled: a fresh process can occupy the same PID slot as
		// the original parent and start strictly *after* the current child.
		// Without this check, the walk would happily stitch that unrelated
		// process into the ancestry chain.
		//
		// Skip the check when either timestamp is unavailable (e.g. macOS
		// and Windows paths that leave StartedAt zero), and accept equal
		// timestamps for platforms with coarse resolution.
		if !p.StartedAt.IsZero() {
			if parent, err := processReader(p.PPID); err == nil && !parent.StartedAt.IsZero() {
				if parent.StartedAt.After(p.StartedAt) {
					break
				}
			}
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
