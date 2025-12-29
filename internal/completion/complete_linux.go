//go:build linux

package completion

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// getRunningPIDs returns a list of all running PIDs
func getRunningPIDs() []string {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var pids []int
	selfPid := os.Getpid()
	parentPid := os.Getppid()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
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
	// Try ss first (faster)
	out, err := exec.Command("ss", "-tlnH").Output()
	if err == nil {
		return parseSSOutput(string(out))
	}

	// Fallback to reading /proc/net/tcp
	return getListeningPortsProc()
}

// parseSSOutput parses ss -tlnH output
func parseSSOutput(output string) []string {
	var ports []int
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Local address is in the 4th field (index 3)
		addr := fields[3]
		if idx := strings.LastIndex(addr, ":"); idx != -1 {
			portStr := addr[idx+1:]
			port, err := strconv.Atoi(portStr)
			if err == nil && port > 0 {
				ports = append(ports, port)
			}
		}
	}
	return uniqueSortedInts(ports)
}

// getListeningPortsProc reads from /proc/net/tcp
func getListeningPortsProc() []string {
	data, err := os.ReadFile("/proc/net/tcp")
	if err != nil {
		return nil
	}

	var ports []int
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		// Skip header
		if fields[0] == "sl" {
			continue
		}
		// Check if state is LISTEN (0A)
		if fields[3] != "0A" {
			continue
		}
		// Local address is in field 1, format: IP:PORT (hex)
		addr := fields[1]
		if idx := strings.Index(addr, ":"); idx != -1 {
			portHex := addr[idx+1:]
			port64, err := strconv.ParseInt(portHex, 16, 32)
			if err == nil && port64 > 0 {
				ports = append(ports, int(port64))
			}
		}
	}

	return uniqueSortedInts(ports)
}

// getProcessNames returns a list of unique running process names
func getProcessNames() []string {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil
	}

	var names []string
	selfPid := os.Getpid()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 || pid == selfPid {
			continue
		}

		// Read comm file for process name
		commPath := filepath.Join("/proc", entry.Name(), "comm")
		data, err := os.ReadFile(commPath)
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(data))
		if name != "" && name != "witr" {
			names = append(names, name)
		}
	}

	return uniqueSorted(names)
}
