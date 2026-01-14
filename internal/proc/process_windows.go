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

	// Get more info via powershell
	psScript := fmt.Sprintf("Get-CimInstance -ClassName Win32_Process -Filter \"ProcessId=%d\" | ForEach-Object { \"CommandLine=$($_.CommandLine)\"; \"CreationDate=$($_.CreationDate.ToUniversalTime().ToString('yyyyMMddHHmmss'))\"; \"ExecutablePath=$($_.ExecutablePath)\"; \"ParentProcessId=$($_.ParentProcessId)\"; \"Status=$($_.Status)\" }", pid)
	psCmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", psScript)
	psOut, _ := psCmd.Output()

	var cmdline, exe string
	var ppid int
	var startedAt time.Time
	health := "healthy"

	lines := strings.Split(string(psOut), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "CommandLine=") {
			cmdline = strings.TrimPrefix(line, "CommandLine=")
		} else if strings.HasPrefix(line, "CreationDate=") {
			val := strings.TrimPrefix(line, "CreationDate=")
			// Format: YYYYMMDDHHMMSS (UTC)
			if len(val) >= 14 {
				t, err := time.Parse("20060102150405", val[:14])
				if err == nil {
					startedAt = t
				}
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
