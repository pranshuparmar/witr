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
	addressToPIDs, err := sockstatPortLookup(port, true)
	if err == nil && len(addressToPIDs) == 0 {
		err = fmt.Errorf("empty")
	}
	if err != nil {
		if fallback, fbErr := sockstatPortLookup(port, false); fbErr == nil && len(fallback) > 0 {
			addressToPIDs = fallback
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
		return nil, ErrSocketOwnerUnknown
	}

	return result, nil
}

// sockstatPortLookup queries sockstat for the given port. When listenersOnly
// is true, only listening sockets are returned (the historical behavior). When
// false, all sockets bound to or connected on the local port are returned so
// processes with established connections become discoverable.
func sockstatPortLookup(port int, listenersOnly bool) (map[string][]int, error) {
	addressToPIDs := make(map[string][]int)
	var queryErr error

	for _, proto := range []string{"tcp", "udp"} {
		for _, flag := range []string{"-4", "-6"} {
			args := []string{flag, "-P", proto, "-p", strconv.Itoa(port)}
			if listenersOnly {
				args = append([]string{"-l"}, args...)
			}
			out, err := exec.Command("sockstat", args...).Output()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
					continue
				}
				if queryErr == nil {
					queryErr = err
				}
				continue
			}

			// Parse sockstat output
			// USER     COMMAND    PID   FD PROTO  LOCAL ADDRESS         FOREIGN ADDRESS
			// root     nginx      1234  6  tcp4   *:80                  *:*
			for line := range strings.Lines(string(out)) {
				fields := strings.Fields(line)
				if len(fields) < 6 {
					continue
				}
				if fields[0] == "USER" {
					continue
				}

				pid, err := strconv.Atoi(fields[2])
				if err != nil || pid <= 0 {
					continue
				}

				localAddr := fields[5]
				addressToPIDs[localAddr] = append(addressToPIDs[localAddr], pid)
			}
		}
	}

	if queryErr != nil {
		return addressToPIDs, fmt.Errorf("query sockstat: %w", queryErr)
	}
	return addressToPIDs, nil
}

func resolvePortNetstat(port int) ([]int, error) {
	portStr := fmt.Sprintf(".%d", port)
	portColonStr := fmt.Sprintf(":%d", port)

	// Check both TCP and UDP via netstat. Any match — listener or connected —
	// is enough to forward to fstat for PID resolution.
	for _, proto := range []string{"tcp", "udp"} {
		out, err := exec.Command("netstat", "-an", "-p", proto).Output()
		if err != nil {
			continue
		}

		for line := range strings.Lines(string(out)) {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			if strings.HasSuffix(fields[3], portStr) || strings.HasSuffix(fields[3], portColonStr) {
				return resolvePortFstat(port)
			}
		}
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

	portSuffix := fmt.Sprintf(":%d", port)
	pidSet := make(map[int]bool)

	for line := range strings.Lines(string(out)) {
		if !strings.Contains(line, "tcp") && !strings.Contains(line, "udp") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		lastField := fields[len(fields)-1]
		if !strings.HasSuffix(lastField, portSuffix) {
			continue
		}
		pid, err := strconv.Atoi(fields[2])
		if err == nil && pid > 0 {
			pidSet[pid] = true
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
