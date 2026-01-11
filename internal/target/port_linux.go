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

	files := []string{"/proc/net/tcp", "/proc/net/tcp6"}
	targetHex := fmt.Sprintf("%04X", port)

	for _, file := range files {
		data, err := os.ReadFile(file)
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
			if state != "0A" { // 0A is the linux /proc/net/tcp* code for TCP_LISTEN, so we only report actual listeners for --port
				continue
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
				}
			}
		}
	}

	result := make([]int, 0, len(pidSet))
	for pid := range pidSet {
		result = append(result, pid)
	}
	sort.Ints(result)

	if len(result) == 0 {
		return nil, fmt.Errorf("socket found but owning process not detected")
	}

	return result, nil
}
