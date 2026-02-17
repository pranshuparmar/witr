//go:build linux

package proc

import (
	"bufio"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

var stateMap = map[string]string{
	"01": "ESTABLISHED",
	"02": "SYN_SENT",
	"03": "SYN_RECV",
	"04": "FIN_WAIT1",
	"05": "FIN_WAIT2",
	"06": "TIME_WAIT",
	"07": "CLOSE",
	"08": "CLOSE_WAIT",
	"09": "LAST_ACK",
	"0A": "LISTEN",
	"0B": "CLOSING",
}

func readSockets() (map[string]model.Socket, error) {
	sockets := make(map[string]model.Socket)

	parse := func(path, proto string, ipv6 bool) {
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Scan() // skip header

		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 10 {
				continue
			}

			local := fields[1]
			stateHex := fields[3]
			inode := fields[9]

			state, ok := stateMap[stateHex]
			if !ok {
				state = "UNKNOWN"
			}

			addr, port := parseAddr(local, ipv6)
			sockets[inode] = model.Socket{
				Inode:    inode,
				Port:     port,
				Address:  addr,
				State:    state,
				Protocol: proto,
			}
		}
	}

	parse("/proc/net/tcp", "TCP", false)
	parse("/proc/net/tcp6", "TCP6", true)
	parse("/proc/net/udp", "UDP", false)
	parse("/proc/net/udp6", "UDP6", true)

	return sockets, nil
}

func parseAddr(raw string, ipv6 bool) (string, int) {
	parts := strings.Split(raw, ":")
	if len(parts) < 2 {
		return "", 0
	}
	portHex := parts[1]
	port, _ := strconv.ParseInt(portHex, 16, 32)

	ipHex := parts[0]
	b, err := hex.DecodeString(ipHex)
	if err != nil {
		return "", int(port)
	}

	if ipv6 {
		if len(b) != 16 {
			return "::", int(port)
		}
		// /proc/net/tcp6 stores IPv6 as 4 little-endian 32-bit groups
		// Reverse bytes within each 4-byte group
		ip := make(net.IP, 16)
		for i := 0; i < 4; i++ {
			ip[i*4+0] = b[i*4+3]
			ip[i*4+1] = b[i*4+2]
			ip[i*4+2] = b[i*4+1]
			ip[i*4+3] = b[i*4+0]
		}
		return ip.String(), int(port)
	}

	if len(b) < 4 {
		return "", int(port)
	}
	ip := strconv.Itoa(int(b[3])) + "." +
		strconv.Itoa(int(b[2])) + "." +
		strconv.Itoa(int(b[1])) + "." +
		strconv.Itoa(int(b[0]))

	return ip, int(port)
}

func ListOpenPorts() ([]model.OpenPort, error) {
	sockets, err := readSockets()
	if err != nil {
		return nil, err
	}

	var openPorts []model.OpenPort

	// Scan proc
	procs, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	for _, p := range procs {
		if !p.IsDir() {
			continue
		}
		pid, err := strconv.Atoi(p.Name())
		if err != nil {
			continue
		}

		// Scan fds
		fdPath := fmt.Sprintf("/proc/%d/fd", pid)
		fds, err := os.ReadDir(fdPath)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(fmt.Sprintf("%s/%s", fdPath, fd.Name()))
			if err != nil {
				continue
			}
			if strings.HasPrefix(link, "socket:[") {
				inode := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
				if s, ok := sockets[inode]; ok {
					openPorts = append(openPorts, model.OpenPort{
						PID:      pid,
						Port:     s.Port,
						Address:  s.Address,
						Protocol: s.Protocol,
						State:    s.State,
					})
				}
			}
		}
	}
	return openPorts, nil
}
