//go:build darwin

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

// isValidServiceLabel validates that a launchd service label contains only
// safe characters to prevent command injection. Valid labels contain only
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

	// Use ps to list all processes on macOS
	// ps -axo pid=,comm=,args=
	out, err := exec.Command("ps", "-axo", "pid=,comm=,args=").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list processes: %w", err)
	}

	for line := range strings.Lines(string(out)) {
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

		// Match against command name
		if strings.Contains(comm, lowerName) {
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

	// Service detection (launchd)
	servicePID, _ := resolveLaunchdServicePID(name)

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
			fmt.Printf("[1] PID %d   %s: launchd service   (service)\n", servicePID, safeName)
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

// resolveLaunchdServicePID tries to resolve a launchd service and returns its PID if running.
func resolveLaunchdServicePID(name string) (int, error) {
	// Validate input before using in command
	if !isValidServiceLabel(name) {
		return 0, fmt.Errorf("invalid service name %q", name)
	}

	// Try common launchd service label patterns
	labels := []string{
		name,
		"com.apple." + name,
		"org." + name,
		"io." + name,
	}

	for _, label := range labels {
		// All labels are derived from validated name, so they're safe
		// launchctl print system/<label> or gui/<uid>/<label>
		out, err := exec.Command("launchctl", "print", "system/"+label).Output()
		if err == nil {
			// Parse output to find PID
			// Look for "pid = <number>"
			for line := range strings.Lines(string(out)) {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "pid = ") {
					pidStr := strings.TrimPrefix(line, "pid = ")
					pid, err := strconv.Atoi(strings.TrimSpace(pidStr))
					if err == nil && pid > 0 {
						return pid, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("service %q not found", name)
}
