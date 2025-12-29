//go:build darwin

package completion

import (
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// getRunningPIDs returns a list of all running PIDs
func getRunningPIDs() []string {
	out, err := exec.Command("ps", "-axo", "pid=").Output()
	if err != nil {
		return nil
	}

	var pids []int
	selfPid := os.Getpid()
	parentPid := os.Getppid()

	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil || pid <= 0 {
			continue
		}
		// Exclude self and parent
		if pid == selfPid || pid == parentPid {
			continue
		}
		pids = append(pids, pid)
	}

	return uniqueSortedInts(pids)
}

// getListeningPorts returns a list of all listening TCP ports
func getListeningPorts() []string {
	// Use lsof to get listening ports on macOS
	// lsof -i -P -n -s TCP:LISTEN
	out, err := exec.Command("lsof", "-i", "-P", "-n", "-s", "TCP:LISTEN").Output()
	if err != nil {
		// Fallback to netstat
		return getListeningPortsNetstat()
	}

	var ports []int
	for _, line := range strings.Split(string(out), "\n") {
		// Skip header
		if strings.HasPrefix(line, "COMMAND") {
			continue
		}
		// Only process lines that are actually TCP LISTEN
		// (the -s TCP:LISTEN filter doesn't exclude UDP lines)
		if !strings.Contains(line, "(LISTEN)") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		// The NAME field (9th) contains the address:port
		name := fields[8]
		// Extract port from patterns like *:8080 or 127.0.0.1:3000
		if idx := strings.LastIndex(name, ":"); idx != -1 {
			portStr := name[idx+1:]
			port, err := strconv.Atoi(portStr)
			if err == nil && port > 0 {
				ports = append(ports, port)
			}
		}
	}

	return uniqueSortedInts(ports)
}

// getListeningPortsNetstat is a fallback using netstat
func getListeningPortsNetstat() []string {
	out, err := exec.Command("netstat", "-anv", "-p", "tcp").Output()
	if err != nil {
		return nil
	}

	var ports []int
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Local address is in the 4th field
		addr := fields[3]
		if idx := strings.LastIndex(addr, "."); idx != -1 {
			portStr := addr[idx+1:]
			port, err := strconv.Atoi(portStr)
			if err == nil && port > 0 {
				ports = append(ports, port)
			}
		}
	}

	return uniqueSortedInts(ports)
}

// getProcessNames returns a list of unique running process names
func getProcessNames() []string {
	out, err := exec.Command("ps", "-axo", "comm=").Output()
	if err != nil {
		return nil
	}

	var names []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Remove leading dash (login shells)
		line = strings.TrimPrefix(line, "-")
		// Get just the command name without path
		if idx := strings.LastIndex(line, "/"); idx != -1 {
			line = line[idx+1:]
		}
		if line != "" && line != "witr" {
			names = append(names, line)
		}
	}

	return uniqueSorted(names)
}
