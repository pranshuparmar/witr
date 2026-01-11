//go:build darwin

package proc

import (
	"errors"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ReadExtendedInfo assembles the additional process facts.
func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, []int, int, error) {
	memInfo, threadCount, memErr := readDarwinTaskInfo(pid)
	fdCount, fileDescs, fdErr := readDarwinFDs(pid)
	ioStats, ioErr := readDarwinIO(pid)
	fdLimit := detectDarwinFileLimit()
	children := listDarwinChildren(pid)

	if memErr != nil && fdErr != nil && ioErr != nil {
		return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, errors.Join(memErr, fdErr, ioErr)
	}

	return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, nil
}

// detectDarwinFileLimit reads launchctl's maxfiles limit (soft cap) so we can
// compute descriptor headroom, falling back to the shell's ulimit if launchctl
// is unavailable.
func detectDarwinFileLimit() uint64 {
	if data, err := exec.Command("launchctl", "limit", "maxfiles").Output(); err == nil {
		for line := range strings.Lines(string(data)) {
			if strings.Contains(line, "maxfiles") {
				if limit, ok := parseLaunchctlLimitLine(line); ok {
					return limit
				}
			}
		}
	}
	if data, err := exec.Command("sh", "-c", "ulimit -n").Output(); err == nil {
		if limit, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			return limit
		}
	}
	return 0
}

func parseLaunchctlLimitLine(line string) (uint64, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return 0, false
	}
	soft := fields[1]
	if strings.EqualFold(soft, "unlimited") {
		return 0, true
	}
	limit, err := strconv.ParseUint(soft, 10, 64)
	if err != nil {
		return 0, false
	}
	return limit, true
}

// Wrapper around pgrep(1).
func listDarwinChildren(pid int) []int {
	cmd := exec.Command("pgrep", "-P", strconv.Itoa(pid))
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil
		}
		return nil
	}
	var children []int
	for line := range strings.Lines(string(out)) {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if pidVal, err := strconv.Atoi(trimmed); err == nil {
			children = append(children, pidVal)
		}
	}
	return children
}
