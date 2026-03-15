package target

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func ResolveFile(path string) ([]int, error) {
	// fstat <file>
	cmd := exec.Command("fstat", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("fstat failed: %w", err)
	}

	var pids []int
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pidStr := fields[2]
			pid, err := strconv.Atoi(pidStr)
			if err == nil {
				pids = append(pids, pid)
			}
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process found holding file: %s", path)
	}
	return pids, nil
}
