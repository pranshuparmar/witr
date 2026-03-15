package target

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func ResolveFile(path string) ([]int, error) {
	// Use lsof -F p <file> to get PIDs
	cmd := exec.Command("lsof", "-F", "p", path)
	out, err := cmd.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			return nil, fmt.Errorf("no process found holding file: %s", path)
		}
		return nil, fmt.Errorf("lsof failed: %w", err)
	}

	var pids []int
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "p") {
			pidStr := line[1:]
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
