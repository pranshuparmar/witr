//go:build freebsd

package target

import (
	"fmt"
	"os/exec"
	"sort"
)

func ResolveBoundPort(port int) ([]int, error) {
	if _, err := exec.LookPath("sockstat"); err != nil {
		return nil, fmt.Errorf("find sockstat: %w", err)
	}
	owners, err := sockstatPortLookup(port, true)
	if err != nil {
		return nil, err
	}
	pidSet := make(map[int]struct{})
	protectedOwner := false
	for _, pids := range owners {
		for _, pid := range pids {
			if pid > 1 {
				pidSet[pid] = struct{}{}
			} else if pid == 1 {
				protectedOwner = true
			}
		}
	}
	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		result = append(result, pid)
	}
	sort.Ints(result)
	if len(result) == 0 {
		if protectedOwner {
			return nil, ErrSocketOwnerUnknown
		}
		return nil, fmt.Errorf("%w %d", ErrPortNotBound, port)
	}
	return result, nil
}
