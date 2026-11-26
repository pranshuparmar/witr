package target

import (
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// addressOwnerPIDs picks the processes behind a port from its sockets grouped
// by local address (FreeBSD's sockstat): the lowest PID of each address, since
// a forking server's workers share its listener, and each process once, so one
// process bound to both 127.0.0.1 and ::1 is a single owner. Several distinct
// owners go to the caller's multiple-match handling, as on other platforms.
func addressOwnerPIDs(addressToPIDs map[string][]int) []int {
	var pids []int
	for _, list := range addressToPIDs {
		if len(list) == 0 {
			continue
		}
		if pid := slices.Min(list); !slices.Contains(pids, pid) {
			pids = append(pids, pid)
		}
	}
	slices.Sort(pids)
	return pids
}

// matchesExactToken checks whether name matches any token in the cmdline,
// including path components. For example, "core24" matches the argument
// "/snap/core24/1349" because "core24" is a complete path segment.
// Handles both forward slashes and backslashes as path separators.
func matchesExactToken(cmdline, name string) bool {
	for _, part := range splitCmdlineTokens(cmdline) {
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

// splitCmdlineTokens splits a command line into argv tokens. Windows command
// lines quote arguments containing spaces with double quotes; apostrophes are
// ordinary characters there. Other platforms join argv with spaces without
// shell quoting, so strings.Fields is the correct splitter.
func splitCmdlineTokens(cmdline string) []string {
	if runtime.GOOS != "windows" {
		return strings.Fields(cmdline)
	}
	return splitWindowsCmdline(cmdline)
}

// splitWindowsCmdline splits a Windows command line, honouring double-quoted
// arguments. A single quote is data, not a quote delimiter.
func splitWindowsCmdline(cmdline string) []string {
	var tokens []string
	var current strings.Builder
	inQuotes := false
	for _, r := range cmdline {
		switch {
		case r == '"':
			inQuotes = !inQuotes
		case r == ' ' && !inQuotes:
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
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
