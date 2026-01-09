//go:build freebsd

package target

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
)

// isValidServiceLabel validates that a service name contains only
// safe characters to prevent command injection. Valid names contain only
// alphanumeric characters, dots, hyphens, and underscores.
var validServiceLabelRegex = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

func isValidServiceLabel(label string) bool {
	if len(label) == 0 || len(label) > 256 {
		return false
	}
	return validServiceLabelRegex.MatchString(label)
}

func ResolveName(name string) ([]int, error) {
	var procPIDs []int

	lowerName := strings.ToLower(name)
	selfPid := os.Getpid()
	parentPid := os.Getppid()

	// Use ps to list all processes on FreeBSD
	// FreeBSD syntax: ps -axww -o pid -o comm -o args
	out, err := exec.Command("ps", "-axww", "-o", "pid", "-o", "comm", "-o", "args").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	// Skip header line (first line)
	for i, line := range lines {
		if i == 0 {
			continue // Skip header
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		// Prevent matching the PID itself as a name
		if lowerName == strconv.Itoa(pid) {
			continue
		}

		// Exclude self and parent (witr, go run, etc.)
		if pid == selfPid || pid == parentPid {
			continue
		}

		comm := strings.ToLower(fields[1])
		args := ""
		if len(fields) > 2 {
			args = strings.ToLower(strings.Join(fields[2:], " "))
		}

		// Match against command name (exact or substring)
		if strings.Contains(comm, lowerName) || comm == lowerName {
			// Exclude grep-like processes
			if !strings.Contains(comm, "grep") {
				procPIDs = append(procPIDs, pid)
				continue
			}
		}

		// Match against full command line
		if strings.Contains(args, lowerName) &&
			!strings.Contains(args, "grep") {

			procPIDs = append(procPIDs, pid)
		}
	}

	// If all matches are filtered out, treat as no result
	if len(procPIDs) == 0 {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}

	// Service detection (rc.d)
	servicePID, _ := resolveRcServicePID(name)

	// Ambiguity: both process and service, but only if there are at least two unique PIDs
	uniquePIDs := map[int]bool{}
	if servicePID > 0 {
		uniquePIDs[servicePID] = true
	}
	for _, pid := range procPIDs {
		uniquePIDs[pid] = true
	}
	if len(uniquePIDs) > 1 {
		safeName := output.SanitizeTerminal(name)

		fmt.Printf("Ambiguous target: \"%s\"\n\n", safeName)
		fmt.Println("The name matches multiple entities:")
		fmt.Println()
		// Service entry first
		if servicePID > 0 {
			fmt.Printf("[1] PID %d   %s: rc.d service   (service)\n", servicePID, safeName)
		}
		// Process entries (skip if PID matches servicePID)
		idx := 2
		if servicePID == 0 {
			idx = 1
		}
		for _, pid := range procPIDs {
			if pid == servicePID {
				continue
			}
			fmt.Printf("[%d] PID %d   %s: process   (manual)\n", idx, pid, safeName)
			idx++
		}
		fmt.Println()
		fmt.Println("witr cannot determine intent safely.")
		fmt.Println("Please re-run with an explicit PID:")
		fmt.Println("  witr --pid <pid>")
		os.Exit(1)
	}

	// Service only
	if servicePID > 0 {
		return []int{servicePID}, nil
	}

	// Process only
	if len(procPIDs) > 0 {
		return procPIDs, nil
	}

	return nil, fmt.Errorf("no running process or service named %q", name)
}

// resolveRcServicePID tries to resolve a FreeBSD rc.d service and returns its PID if running.
func resolveRcServicePID(name string) (int, error) {
	// Validate input before using in command
	if !isValidServiceLabel(name) {
		return 0, fmt.Errorf("invalid service name %q", name)
	}

	// Check /var/run/<name>.pid
	pidFile := "/var/run/" + name + ".pid"
	content, err := os.ReadFile(pidFile)
	if err == nil {
		pid, err := strconv.Atoi(strings.TrimSpace(string(content)))
		if err == nil && pid > 0 {
			// Verify process exists using ps command
			out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid=").Output()
			if err == nil && strings.TrimSpace(string(out)) != "" {
				return pid, nil
			}
		}
	}

	// Try service <name> status
	out, err := exec.Command("service", name, "status").Output()
	if err == nil {
		outStr := string(out)
		// Look for "is running as pid <number>" or "PID: <number>"
		if strings.Contains(outStr, "is running") {
			// Extract PID from output
			if idx := strings.Index(outStr, "pid "); idx != -1 {
				start := idx + 4
				end := start
				for end < len(outStr) && outStr[end] >= '0' && outStr[end] <= '9' {
					end++
				}
				if end > start {
					pid, err := strconv.Atoi(outStr[start:end])
					if err == nil && pid > 0 {
						return pid, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("service %q not found", name)
}
