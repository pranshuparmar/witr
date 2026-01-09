//go:build freebsd

package target

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
)

func ResolvePort(port int) ([]int, error) {
	// Use sockstat to find the process listening on this port
	// sockstat -4 -l -P tcp -p <port>
	// sockstat -6 -l -P tcp -p <port>

	// Map: bind address (IP:port) -> list of PIDs
	addressToPIDs := make(map[string][]int)

	for _, flag := range []string{"-4", "-6"} {
		out, err := exec.Command("sockstat", flag, "-l", "-P", "tcp", "-p", strconv.Itoa(port)).Output()
		if err != nil {
			continue
		}

		// Parse sockstat output
		// USER     COMMAND    PID   FD PROTO  LOCAL ADDRESS         FOREIGN ADDRESS
		// root     nginx      1234  6  tcp4   *:80                  *:*
		// root     sshd       34    3  tcp4   192.168.1.2:22        *:*
		for line := range strings.Lines(string(out)) {
			fields := strings.Fields(line)
			if len(fields) < 6 {
				continue
			}

			// Skip header
			if fields[0] == "USER" {
				continue
			}

			pid, err := strconv.Atoi(fields[2])
			if err != nil || pid <= 0 {
				continue
			}

			// Extract LOCAL ADDRESS (field index 5)
			localAddr := fields[5]
			addressToPIDs[localAddr] = append(addressToPIDs[localAddr], pid)
		}
	}

	if len(addressToPIDs) == 0 {
		// Try netstat as fallback
		return resolvePortNetstat(port)
	}

	// For each unique bind address, keep only the smallest PID
	// (to handle master/worker pattern like nginx)
	uniqueAddresses := make(map[string]int) // address -> min PID
	for addr, pids := range addressToPIDs {
		minPID := pids[0]
		for _, pid := range pids {
			if pid < minPID {
				minPID = pid
			}
		}
		uniqueAddresses[addr] = minPID
	}

	// If multiple different addresses are listening, show ambiguity and exit
	// (this indicates separate services, not master/worker)
	if len(uniqueAddresses) > 1 {
		return handlePortAmbiguity(port, uniqueAddresses)
	}

	// Single address: return the PID
	var result []int
	for _, pid := range uniqueAddresses {
		result = append(result, pid)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("socket found but owning process not detected")
	}

	return result, nil
}

func resolvePortNetstat(port int) ([]int, error) {
	// Fallback using netstat
	// On FreeBSD: netstat -an -p tcp | grep LISTEN
	out, err := exec.Command("netstat", "-an", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	portStr := fmt.Sprintf(".%d", port)
	portColonStr := fmt.Sprintf(":%d", port)

	for line := range strings.Lines(string(out)) {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		if !strings.Contains(line, portStr) && !strings.Contains(line, portColonStr) {
			continue
		}

		// Unfortunately, basic netstat doesn't show PID on FreeBSD
		// We need to use sockstat or fstat for that
		// Try fstat as last resort
		return resolvePortFstat(port)
	}

	return nil, fmt.Errorf("no process listening on port %d", port)
}

func resolvePortFstat(port int) ([]int, error) {
	// Use fstat to find processes with open sockets
	// This is less efficient but works as a fallback
	out, err := exec.Command("fstat").Output()
	if err != nil {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	portStr := fmt.Sprintf(":%d", port)
	pidSet := make(map[int]bool)

	for line := range strings.Lines(string(out)) {
		if !strings.Contains(line, "tcp") {
			continue
		}
		if !strings.Contains(line, portStr) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) >= 3 {
			pid, err := strconv.Atoi(fields[2])
			if err == nil && pid > 0 {
				pidSet[pid] = true
			}
		}
	}

	if len(pidSet) == 0 {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	// Return the lowest PID
	var result []int
	minPID := 0
	for pid := range pidSet {
		if minPID == 0 || pid < minPID {
			minPID = pid
		}
	}
	if minPID > 0 {
		result = append(result, minPID)
	}

	return result, nil
}

// handlePortAmbiguity displays disambiguation information when multiple services
// are listening on different addresses for the same port
func handlePortAmbiguity(port int, addressToPID map[string]int) ([]int, error) {
	fmt.Fprintf(os.Stderr, "Ambiguous port query: %d\n\n", port)
	fmt.Fprintln(os.Stderr, "Multiple services are listening on different addresses:")
	fmt.Fprintln(os.Stderr, "")

	// Sort addresses for consistent output
	type addrPID struct {
		addr string
		pid  int
	}
	var entries []addrPID
	for addr, pid := range addressToPID {
		entries = append(entries, addrPID{addr, pid})
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].pid < entries[j].pid
	})

	// Display each service
	for i, entry := range entries {
		// Get command name
		cmdline := "(unknown)"
		psOut, err := exec.Command("ps", "-p", strconv.Itoa(entry.pid), "-o", "args").Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(psOut)), "\n")
			if len(lines) >= 2 {
				cmdline = strings.TrimSpace(lines[1])
			}
		}

		// Check if in jail
		context := ""
		jailOut, err := exec.Command("jls", "-j", strconv.Itoa(entry.pid)).Output()
		if err == nil && strings.TrimSpace(string(jailOut)) != "" {
			context = " (jail)"
		}

		safeAddr := output.SanitizeTerminal(entry.addr)
		safeCmdline := output.SanitizeTerminal(cmdline)
		safeContext := output.SanitizeTerminal(context)
		fmt.Fprintf(os.Stderr, "[%d] PID %d   %s   %s%s\n", i+1, entry.pid, safeAddr, safeCmdline, safeContext)
	}

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "witr cannot determine intent safely.")
	fmt.Fprintln(os.Stderr, "Please re-run with an explicit PID:")
	fmt.Fprintln(os.Stderr, "  witr --pid <pid>")

	return nil, fmt.Errorf("multiple services listening on port %d", port)
}
