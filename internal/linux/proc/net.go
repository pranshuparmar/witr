//go:build linux

package proc

import (
	"bufio"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
)

func readListeningSockets() (map[string]Socket, error) {
	sockets := make(map[string]Socket)

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
			sockets[inode] = Socket{
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
	portHex := parts[1]
	port, _ := strconv.ParseInt(portHex, 16, 32)

	if ipv6 {
		return "::", int(port)
	}

	ipHex := parts[0]
	b, _ := hex.DecodeString(ipHex)
	ip := strconv.Itoa(int(b[3])) + "." +
		strconv.Itoa(int(b[2])) + "." +
		strconv.Itoa(int(b[1])) + "." +
		strconv.Itoa(int(b[0]))

	return ip, int(port)
}
