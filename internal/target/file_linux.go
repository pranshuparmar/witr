package target

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

// ResolveFile finds processes holding a lock on the given file path
func ResolveFile(path string) ([]int, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	realPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		realPath = absPath
	}

	var pids []int

	procDirs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to read /proc: %w", err)
	}

	for _, d := range procDirs {
		if !d.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(d.Name())
		if err != nil {
			continue
		}

		fdDir := filepath.Join("/proc", d.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			linkPath, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}

			if linkPath == realPath || linkPath == absPath {
				pids = append(pids, pid)
				break
			}
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process found holding file: %s", absPath)
	}

	return pids, nil
}
