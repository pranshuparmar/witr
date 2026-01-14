//go:build linux

package proc

import (
	"bufio"
	"encoding/hex"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func readListeningSockets() (map[string]model.Socket, error) {
	sockets := make(map[string]model.Socket)

	parse := func(path string, ipv6 bool) {
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
			state := fields[3]
			inode := fields[9]

			// 0A = LISTEN
			if state != "0A" {
				continue
			}

			addr, port := parseAddr(local, ipv6)
			sockets[inode] = model.Socket{
				Inode:   inode,
				Port:    port,
				Address: addr,
			}
		}
	}

	parse("/proc/net/tcp", false)
	parse("/proc/net/tcp6", true)

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
