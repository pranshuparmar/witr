package completion

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// CompleteType represents what kind of completion is requested
type CompleteType string

const (
	CompletePIDs      CompleteType = "pids"
	CompletePorts     CompleteType = "ports"
	CompleteProcesses CompleteType = "processes"
)

// Complete outputs completion candidates for the given type
func Complete(completeType CompleteType) {
	var candidates []string

	switch completeType {
	case CompletePIDs:
		candidates = getRunningPIDs()
	case CompletePorts:
		candidates = getListeningPorts()
	case CompleteProcesses:
		candidates = getProcessNames()
	default:
		fmt.Fprintf(os.Stderr, "unknown completion type: %s\n", completeType)
		os.Exit(1)
	}

	for _, c := range candidates {
		fmt.Println(c)
	}
}

// shellMetaChars contains characters that are unsafe in shell completion contexts.
// Process names containing these characters are filtered out to prevent command injection.
const shellMetaChars = " \t\n$`\\\"';&|<>(){}[]!*?~"

// isShellSafe returns true if the string contains no shell metacharacters
func isShellSafe(s string) bool {
	return !strings.ContainsAny(s, shellMetaChars)
}

// uniqueSorted returns a sorted slice with duplicates removed
func uniqueSorted(items []string) []string {
	seen := make(map[string]bool)
	var result []string
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" && !seen[item] && isShellSafe(item) {
			seen[item] = true
			result = append(result, item)
		}
	}
	sort.Strings(result)
	return result
}

// uniqueSortedInts returns a sorted slice of ints as strings with duplicates removed
func uniqueSortedInts(items []int) []string {
	seen := make(map[int]bool)
	var nums []int
	for _, item := range items {
		if item > 0 && !seen[item] {
			seen[item] = true
			nums = append(nums, item)
		}
	}
	sort.Ints(nums)
	result := make([]string, len(nums))
	for i, n := range nums {
		result[i] = strconv.Itoa(n)
	}
	return result
}
