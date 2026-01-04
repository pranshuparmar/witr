//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func GetSocketStateForPort(port int) *model.SocketInfo {
	// netstat -ano
	out, err := exec.Command("netstat", "-ano").Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(out), "\n")
	portStr := fmt.Sprintf(":%d", port)

	var states []model.SocketInfo

	for _, line := range lines {
		if strings.Contains(line, portStr) {
			fields := strings.Fields(line)
			if len(fields) < 4 {
				continue
			}
			// Proto Local Address Foreign Address State PID
			// TCP 0.0.0.0:135 0.0.0.0:0 LISTENING 888

			localAddr := fields[1]
			if !strings.HasSuffix(localAddr, portStr) {
				continue
			}

			state := fields[3]
			remoteAddr := fields[2]

			info := model.SocketInfo{
				Port:       port,
				State:      state,
				LocalAddr:  localAddr,
				RemoteAddr: remoteAddr,
			}
			addStateExplanation(&info)
			states = append(states, info)
		}
	}

	if len(states) == 0 {
		return nil
	}

	// Prioritize problematic states
	for _, s := range states {
		if s.State == "TIME_WAIT" || s.State == "CLOSE_WAIT" || s.State == "FIN_WAIT_1" || s.State == "FIN_WAIT_2" {
			return &s
		}
	}

	// Return LISTEN
	for _, s := range states {
		if s.State == "LISTENING" { // Windows uses LISTENING
			return &s
		}
	}

	return &states[0]
}

func addStateExplanation(info *model.SocketInfo) {
	switch info.State {
	case "LISTENING":
		info.Explanation = "Actively listening for connections"
	case "TIME_WAIT":
		info.Explanation = "Connection closed, waiting for delayed packets"
		info.Workaround = "Wait for timeout (usually 60-240s) or reuse port"
	case "CLOSE_WAIT":
		info.Explanation = "Remote side closed connection, local side still has it open"
		info.Workaround = "Check if application is leaking connections or hanging"
	case "ESTABLISHED":
		info.Explanation = "Active connection established"
	case "SYN_SENT":
		info.Explanation = "Attempting to establish connection"
		info.Workaround = "Check firewall or if remote host is up"
	case "SYN_RCVD":
		info.Explanation = "Received connection request, sending ack"
	}
}
