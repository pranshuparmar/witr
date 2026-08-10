//go:build freebsd

package target

import (
	"fmt"
	"sort"
)

func ResolveBoundPort(port int) ([]int, error) {
	owners, _ := sockstatPortLookup(port, true)
	pidSet := make(map[int]struct{})
	for _, pids := range owners {
		for _, pid := range pids {
			if pid > 0 {
				pidSet[pid] = struct{}{}
			}
		}
	}
	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		result = append(result, pid)
	}
	sort.Ints(result)
	if len(result) == 0 {
		return nil, fmt.Errorf("%w %d", ErrPortNotBound, port)
	}
	return result, nil
}
