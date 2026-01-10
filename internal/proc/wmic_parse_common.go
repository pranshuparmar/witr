package proc

import (
	"strings"
)

// parseWmicServiceNameInternal parses the output of wmic service get Name /format:list
func parseWmicServiceNameInternal(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Name=") {
			return strings.TrimPrefix(line, "Name=")
		}
	}
	return ""
}
