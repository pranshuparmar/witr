//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	// Check if process exists using tasklist
	cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH")
	out, err := cmd.Output()
	if err != nil {
		return model.Process{}, err
	}
	output := string(out)
	if strings.Contains(output, "No tasks are running") {
		return model.Process{}, fmt.Errorf("process %d not found", pid)
	}

	// Parse basic info from tasklist
	// "Image Name","PID","Session Name","Session#","Mem Usage"
	parts := strings.Split(output, "\",\"")
	name := ""
	if len(parts) >= 1 {
		name = strings.Trim(parts[0], "\"")
	}

	// Get more info via wmic
	// wmic process where processid=PID get CommandLine,CreationDate,ExecutablePath,ParentProcessId /format:list
	wmicCmd := exec.Command("wmic", "process", "where", fmt.Sprintf("processid=%d", pid), "get", "CommandLine,CreationDate,ExecutablePath,ParentProcessId,Status", "/format:list")
	wmicOut, _ := wmicCmd.Output()

	var cmdline, exe string
	var ppid int
	var startedAt time.Time
	health := "healthy"

	lines := strings.Split(string(wmicOut), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "CommandLine=") {
			cmdline = strings.TrimPrefix(line, "CommandLine=")
		} else if strings.HasPrefix(line, "CreationDate=") {
			val := strings.TrimPrefix(line, "CreationDate=")
			// Format: YYYYMMDDHHMMSS.mmmmmm+UUU
			if len(val) >= 21 {
				t, err := time.Parse("20060102150405", val[:14])
				if err == nil {
					offsetPart := val[len(val)-4:]
					sign := offsetPart[0]
					offsetMins, err := strconv.Atoi(offsetPart[1:])
					if err == nil {
						if sign == '-' {
							offsetMins = -offsetMins
						}
						loc := time.FixedZone("Local", offsetMins*60)
						startedAt = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
					} else {
						startedAt = t
					}
				}
			} else if len(val) >= 14 {
				startedAt, _ = time.Parse("20060102150405", val[:14])
			}
		} else if strings.HasPrefix(line, "ExecutablePath=") {
			exe = strings.TrimPrefix(line, "ExecutablePath=")
		} else if strings.HasPrefix(line, "ParentProcessId=") {
			val := strings.TrimPrefix(line, "ParentProcessId=")
			ppid, _ = strconv.Atoi(val)
		} else if strings.HasPrefix(line, "Status=") {
			val := strings.TrimPrefix(line, "Status=")
			if val != "" {
				health = strings.ToLower(val)
			}
		}
	}

	ports, addrs := GetListeningPortsForPID(pid)

	return model.Process{
		PID:            pid,
		PPID:           ppid,
		Command:        name,
		Cmdline:        cmdline,
		Exe:            exe,
		StartedAt:      startedAt,
		User:           readUser(pid),
		WorkingDir:     "unknown", // Hard to get on Windows without injection
		ListeningPorts: ports,
		BindAddresses:  addrs,
		Health:         health,
		Forked:         "unknown",
		Env:            []string{}, // Hard to get on Windows
	}, nil
}
