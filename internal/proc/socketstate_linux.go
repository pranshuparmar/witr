//go:build linux

package proc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetSocketStateForPort returns the socket state for a port
// Linux implementation using /proc/net/tcp and /proc/net/tcp6
func GetSocketStateForPort(port int) *model.SocketInfo {
	// Check both IPv4 and IPv6
	files := []string{"/proc/net/tcp", "/proc/net/tcp6"}

	var states []model.SocketInfo

	for _, file := range files {
		isIPv6 := strings.HasSuffix(file, "tcp6")

		f, err := os.Open(file)
		if err != nil {
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Scan()

		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 10 {
				continue
			}

			// Field 1: local_address (IP:Port in hex)
			// Format example: 0100007F:1388
			localAddrHex := fields[1]
			// parseAddr is defined in internal/proc/net_linux.go (same package)
			localIP, localPort := parseAddr(localAddrHex, isIPv6)

			if localPort != port {
				continue
			}

			// Field 2: rem_address (IP:Port in hex)
			remoteAddrHex := fields[2]
			remoteIP, _ := parseAddr(remoteAddrHex, isIPv6)

			// Field 3: st (state in hex)
			stateHex := fields[3]
			stateVal, _ := strconv.ParseInt(stateHex, 16, 0)
			stateStr := mapTCPState(int(stateVal))

			info := model.SocketInfo{
				Port:       port,
				State:      stateStr,
				LocalAddr:  localIP,
				RemoteAddr: remoteIP,
			}

			addStateExplanation(&info)
			states = append(states, info)
		}
	}

	if len(states) == 0 {
		return nil
	}

	// Prioritize problematic states just like the Darwin implementation
	for _, s := range states {
		if isProblematicState(s.State) {
			return &s
		}
	}

	// Then prioritize LISTEN
	for _, s := range states {
		if s.State == "LISTEN" {
			return &s
		}
	}

	// Default to first found
	return &states[0]
}

// mapTCPState maps Linux kernel TCP states (from include/net/tcp_states.h) to strings
func mapTCPState(state int) string {
	switch state {
	case 1:
		return "ESTABLISHED"
	case 2:
		return "SYN_SENT"
	case 3:
		return "SYN_RECV"
	case 4:
		return "FIN_WAIT_1"
	case 5:
		return "FIN_WAIT_2"
	case 6:
		return "TIME_WAIT"
	case 7:
		return "CLOSE"
	case 8:
		return "CLOSE_WAIT"
	case 9:
		return "LAST_ACK"
	case 10:
		return "LISTEN"
	case 11:
		return "CLOSING"
	default:
		return fmt.Sprintf("UNKNOWN (%02X)", state)
	}
}

func isProblematicState(state string) bool {
	switch state {
	case "TIME_WAIT", "CLOSE_WAIT", "FIN_WAIT_1", "FIN_WAIT_2":
		return true
	}
	return false
}

func addStateExplanation(info *model.SocketInfo) {
	switch info.State {
	case "LISTEN":
		info.Explanation = "Actively listening for connections"
	case "TIME_WAIT":
		info.Explanation = "Connection closed, waiting for delayed packets"
		info.Workaround = "Wait for timeout (usually 60s) or use SO_REUSEADDR"
	case "CLOSE_WAIT":
		info.Explanation = "Remote side closed connection, local side has not closed yet"
		info.Workaround = "The application should call close() on the socket"
	case "FIN_WAIT_1":
		info.Explanation = "Local side initiated close, waiting for acknowledgment"
	case "FIN_WAIT_2":
		info.Explanation = "Local close acknowledged, waiting for remote close"
	case "ESTABLISHED":
		info.Explanation = "Active connection"
	case "SYN_SENT":
		info.Explanation = "Connection request sent, waiting for response"
	case "SYN_RECEIVED":
		info.Explanation = "Connection request received, sending acknowledgment"
	case "CLOSING":
		info.Explanation = "Both sides initiated close simultaneously"
	case "LAST_ACK":
		info.Explanation = "Waiting for final acknowledgment of close"
	default:
		info.Explanation = "Socket in " + info.State + " state"
	}
}
