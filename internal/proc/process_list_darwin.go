//go:build darwin

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListProcesses returns a list of all running processes with basic details (PID, Command, State).
// This is used by the TUI to display the process list.
func ListProcesses() ([]model.Process, error) {
	// Use ps to fetch rich information efficiently: pid, ppid, user, lstart, %cpu, rss, %mem, args.
	// comm is excluded from this row because it can contain spaces (e.g.
	// "Microsoft Teams") which breaks the strings.Fields column parse used below;
	// it is fetched separately via readPIDCommMap() so the display name is taken
	// from an unambiguous source rather than re-derived from the space-joined args.
	cmd := exec.Command("ps", "-axo", "pid,ppid,user,lstart,%cpu,rss,%mem,args")
	cmd.Env = buildEnvForPS()
	out, err := cmd.Output()
	if err != nil {
		// Fallback to fast snapshot if ps fails
		return ListProcessSnapshot()
	}

	commMap := readPIDCommMap()

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")

	// Skip header
	if len(lines) > 0 {
		lines = lines[1:]
	}

	processes := make([]model.Process, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)

		// Expected minimum fields: pid(1) + ppid(1) + user(1) + lstart(5) + cpu(1) + rss(1) + mem(1) = 11
		if len(fields) < 11 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		user := fields[2]

		// lstart format: "Mon Jan 1 12:00:00 2024" (5 fields)
		timeStr := strings.Join(fields[3:8], " ")
		started, _ := time.Parse("Mon Jan 2 15:04:05 2006", timeStr)

		cpu, _ := strconv.ParseFloat(fields[8], 64)
		rss, _ := strconv.ParseUint(fields[9], 10, 64)
		rss *= 1024

		mem, _ := strconv.ParseFloat(fields[10], 64)

		cmdline := ""
		if len(fields) > 11 {
			cmdline = strings.Join(fields[11:], " ")
		}

		// Prefer the comm value (full executable path on macOS, captured
		// separately so embedded spaces survive) over an args-based extractor
		// which would mis-split paths like "/Applications/Microsoft Teams.app/...".
		displayName := binaryBasename(commMap[pid])
		if displayName == "" {
			displayName = extractExecutableName(cmdline)
		}
		if displayName == "" && len(fields) > 11 {
			displayName = fields[11]
		}
		if displayName == "" {
			continue
		}
		if cmdline == "" {
			cmdline = displayName
		}

		processes = append(processes, model.Process{
			PID:           pid,
			PPID:          ppid,
			Command:       displayName,
			User:          user,
			StartedAt:     started,
			CPUPercent:    cpu,
			MemoryRSS:     rss,
			MemoryPercent: mem,
			Cmdline:       cmdline,
		})
	}

	return processes, nil
}

// ListProcessSnapshot collects a lightweight view of running processes
// for child/descendant discovery. We avoid full ReadProcess calls to keep
// this path fast and to reduce permission-sensitive reads.
func ListProcessSnapshot() ([]model.Process, error) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,comm=").Output()
	if err != nil {
		return nil, fmt.Errorf("ps process list: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	processes := make([]model.Process, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		ppid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		command := strings.Join(fields[2:], " ")
		processes = append(processes, model.Process{
			PID:     pid,
			PPID:    ppid,
			Command: command,
		})
	}

	return processes, nil
}
