//go:build linux

package target

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func findSocketInodes(port int) (map[string]bool, error) {
	inodes := make(map[string]bool)

	type procNetFile struct {
		path  string
		isTCP bool
	}
	files := []procNetFile{
		{"/proc/net/tcp", true},
		{"/proc/net/tcp6", true},
		{"/proc/net/udp", false},
		{"/proc/net/udp6", false},
	}
	targetHex := fmt.Sprintf("%04X", port)

	for _, f := range files {
		data, err := os.ReadFile(f.path)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 10 {
				continue
			}

			localAddr := fields[1]
			parts := strings.Split(localAddr, ":")
			if len(parts) != 2 {
				continue
			}

			state := fields[3]
			if f.isTCP {
				// 0A = TCP_LISTEN — only report actual listeners
				if state != "0A" {
					continue
				}
			} else {
				// UDP is connectionless; state 07 (CLOSE) means the socket is
				// bound and ready to receive. Also accept 01 (ESTABLISHED) for
				// connected UDP sockets.
				if state != "07" && state != "01" {
					continue
				}
			}

			if parts[1] == targetHex {
				inodes[fields[9]] = true
			}
		}
	}

	if len(inodes) == 0 {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	return inodes, nil
}

func ResolvePort(port int) ([]int, error) {
	inodes, err := findSocketInodes(port)
	if err != nil {
		return nil, err
	}

	// collect all owning pids so callers can handle multi-owner sockets.
	pidSet := make(map[int]bool)
	procEntries, _ := os.ReadDir("/proc")
	for _, entry := range procEntries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		fdDir := filepath.Join("/proc", entry.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}

			if rest, ok := strings.CutPrefix(link, "socket:["); ok {
				inode, ok := strings.CutSuffix(rest, "]")
				if ok && inodes[inode] {
					pidSet[pid] = true
					break
				}
			}
		}
	}

	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		if len(pidSet) > 1 && pid == 1 {
			continue
		}
		result = append(result, pid)
	}
	sort.Ints(result)

	if len(result) == 0 {
		return nil, fmt.Errorf("socket found but owning process not detected")
	}

	return result, nil
}
