package target

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// matchesExactToken checks whether name matches any token in the cmdline,
// including path components. For example, "core24" matches the argument
// "/snap/core24/1349" because "core24" is a complete path segment.
// Handles both forward slashes and backslashes as path separators.
func matchesExactToken(cmdline, name string) bool {
	for _, part := range strings.Fields(cmdline) {
		if part == name {
			return true
		}
		// Normalize backslashes to forward slashes for uniform splitting
		normalized := strings.ReplaceAll(part, "\\", "/")
		for _, seg := range strings.Split(normalized, "/") {
			if seg == name {
				return true
			}
		}
	}
	return false
}

func Resolve(t model.Target, exact bool) ([]int, error) {
	val := strings.TrimSpace(t.Value)

	switch t.Type {
	case model.TargetPID:
		pid, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid pid")
		}
		if pid <= 0 {
			return nil, fmt.Errorf("invalid pid: must be a positive integer")
		}
		return []int{pid}, nil

	case model.TargetPort:
		port, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid port")
		}
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("invalid port: must be between 1 and 65535")
		}
		return ResolvePort(port)

	case model.TargetName:
		return ResolveName(val, exact)

	case model.TargetFile:
		return ResolveFile(val)

	default:
		return nil, fmt.Errorf("unknown target")
	}
}
