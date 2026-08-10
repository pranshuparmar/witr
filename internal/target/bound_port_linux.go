//go:build linux

package target

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func ResolveBoundPort(port int) ([]int, error) {
	inodes, err := findSocketInodes(port, true)
	if err != nil {
		return nil, fmt.Errorf("%w %d", ErrPortNotBound, port)
	}

	pidSet := make(map[int]struct{})
	entries, _ := os.ReadDir("/proc")
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}
		fds, err := os.ReadDir(filepath.Join("/proc", entry.Name(), "fd"))
		if err != nil {
			continue
		}
		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join("/proc", entry.Name(), "fd", fd.Name()))
			if err != nil {
				continue
			}
			if value, ok := strings.CutPrefix(link, "socket:["); ok {
				inode, ok := strings.CutSuffix(value, "]")
				if ok && inodes[inode] {
					pidSet[pid] = struct{}{}
					break
				}
			}
		}
	}

	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		// PID 1 may own a systemd socket-activation fd. Never terminate the
		// init process from a convenience command; report the owner as hidden
		// instead so callers can handle it explicitly.
		if pid == 1 {
			continue
		}
		result = append(result, pid)
	}
	sort.Ints(result)
	if len(result) == 0 {
		return nil, ErrSocketOwnerUnknown
	}
	return result, nil
}
