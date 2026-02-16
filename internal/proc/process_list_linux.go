//go:build linux

package proc

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListProcesses returns a list of all running processes with basic details (PID, Command, State).
// This is used by the TUI to display the process list.
func ListProcesses() ([]model.Process, error) {
	// Use ps to fetch rich information efficiently: pid, ppid, user, lstart, %cpu, rss, %mem, comm, args
	out, err := exec.Command("ps", "-axo", "pid,ppid,user,lstart,%cpu,rss,%mem,comm,args").Output()
	if err != nil {
		// Fallback to fast snapshot if ps fails
		return listProcessSnapshot()
	}

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

		// Expected minimum fields: pid(1) + ppid(1) + user(1) + lstart(5) + cpu(1) + rss(1) + mem(1) + comm(1) = 12
		if len(fields) < 12 {
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

		comm := fields[11]

		cmdline := comm
		if len(fields) > 12 {
			cmdline = strings.Join(fields[12:], " ")
		}

		processes = append(processes, model.Process{
			PID:           pid,
			PPID:          ppid,
			Command:       comm,
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

// listProcessSnapshot collects a lightweight view of running processes
// for child/descendant discovery. We avoid full ReadProcess calls to keep
// this path fast and to reduce permission-sensitive reads.
func listProcessSnapshot() ([]model.Process, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, fmt.Errorf("read /proc: %w", err)
	}

	processes := make([]model.Process, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		statPath := fmt.Sprintf("/proc/%d/stat", pid)
		stat, err := os.ReadFile(statPath)
		if err != nil {
			continue
		}

		proc, err := parseStatSnapshot(pid, stat)
		if err != nil {
			continue
		}

		processes = append(processes, proc)
	}

	return processes, nil
}

func parseStatSnapshot(pid int, stat []byte) (model.Process, error) {
	raw := string(stat)
	open := strings.Index(raw, "(")
	close := strings.LastIndex(raw, ")")
	if open == -1 || close == -1 || close <= open {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	comm := raw[open+1 : close]
	fields := strings.Fields(raw[close+2:])
	if len(fields) < 2 {
		return model.Process{}, fmt.Errorf("invalid stat format")
	}

	ppid, err := strconv.Atoi(fields[1])
	if err != nil {
		return model.Process{}, fmt.Errorf("invalid ppid")
	}

	return model.Process{
		PID:     pid,
		PPID:    ppid,
		Command: comm,
	}, nil
}
