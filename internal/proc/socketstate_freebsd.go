//go:build freebsd

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetSocketStates returns all socket states for a given port
func GetSocketStates(port int) ([]model.SocketInfo, error) {
	var sockets []model.SocketInfo

	// Use netstat to get all socket states (not just LISTEN)
	// netstat -an -p tcp shows all TCP connections with states
	out, err := exec.Command("netstat", "-an", "-p", "tcp").Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get socket states: %w", err)
	}

	portSuffix := fmt.Sprintf(".%d", port)
	portColonSuffix := fmt.Sprintf(":%d", port)

	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}

		// Check if this line mentions our port
		localAddr := fields[3]
		if !strings.HasSuffix(localAddr, portSuffix) && !strings.HasSuffix(localAddr, portColonSuffix) {
			continue
		}

		// Parse the state (field 5)
		state := fields[5]
		remoteAddr := fields[4]

		// Parse local address
		proto := fields[0] // tcp4 or tcp6
		address, _ := parseSockstatAddr(localAddr, proto)

		info := model.SocketInfo{
			Port:       port,
			State:      state,
			LocalAddr:  address,
			RemoteAddr: remoteAddr,
		}

		// Add explanation and workaround based on state
		addStateExplanation(&info)

		sockets = append(sockets, info)
	}

	return sockets, nil
}

// GetSocketStateForPort returns the most relevant socket state for a port
// Prioritizes non-LISTEN states that explain why a port might be unavailable
func GetSocketStateForPort(port int) *model.SocketInfo {
	states, err := GetSocketStates(port)
	if err != nil || len(states) == 0 {
		return nil
	}

	// Prioritize problematic states
	for _, s := range states {
		if s.State == "TIME_WAIT" || s.State == "CLOSE_WAIT" || s.State == "FIN_WAIT_1" || s.State == "FIN_WAIT_2" {
			return &s
		}
	}

	// Return LISTEN if that's all we have
	for _, s := range states {
		if s.State == "LISTEN" {
			return &s
		}
	}

	// Return first state found
	if len(states) > 0 {
		return &states[0]
	}

	return nil
}

// addStateExplanation adds human-readable explanation for socket states
func addStateExplanation(info *model.SocketInfo) {
	switch info.State {
	case "LISTEN":
		info.Explanation = "Actively listening for connections"

	case "TIME_WAIT":
		info.Explanation = "Connection closed, waiting for delayed packets (default 60s on FreeBSD)"
		info.Workaround = "Wait for timeout to expire, or use SO_REUSEADDR in your server"

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

// GetMSLDuration returns the Maximum Segment Lifetime setting
// This determines TIME_WAIT duration (2 * MSL)
func GetMSLDuration() int {
	// Try to read from sysctl
	out, err := exec.Command("sysctl", "-n", "net.inet.tcp.msl").Output()
	if err != nil {
		return 30000 // Default 30 seconds in milliseconds
	}

	msl, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 30000
	}

	return msl
}
