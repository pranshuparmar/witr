//go:build linux

package target

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
)

// findSocketInodes returns the inodes of the sockets on port, each with the
// uid that owns it.
func findSocketInodes(port int, listenersOnly bool) (map[string]int, error) {
	inodes := make(map[string]int)

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

			localParts := strings.Split(fields[1], ":")
			if len(localParts) != 2 {
				continue
			}
			remoteParts := strings.Split(fields[2], ":")

			state := fields[3]
			if f.isTCP {
				if listenersOnly {
					// 0A = TCP_LISTEN — only report actual listeners
					if state != "0A" {
						continue
					}
				}
				// In fallback mode, accept any TCP state so connected sockets
				// (ESTABLISHED, TIME_WAIT, etc.) are also surfaced.
			} else {
				// UDP is connectionless; state 07 (CLOSE) means the socket is
				// bound and ready to receive. Also accept 01 (ESTABLISHED) for
				// connected UDP sockets.
				if state != "07" && state != "01" {
					continue
				}
			}

			matches := localParts[1] == targetHex
			if !listenersOnly && !matches && len(remoteParts) == 2 {
				// Match either side of the connection so a process with an
				// outbound connection to :port is also found.
				matches = remoteParts[1] == targetHex
			}
			if matches {
				uid, err := strconv.Atoi(fields[7])
				if err != nil {
					uid = -1
				}
				inodes[fields[9]] = uid
			}
		}
	}

	if len(inodes) == 0 {
		if listenersOnly {
			return nil, fmt.Errorf("no process listening on port %d", port)
		}
		return nil, fmt.Errorf("no process bound to or connected on port %d", port)
	}

	return inodes, nil
}

func ResolvePort(port int) ([]int, error) {
	inodes, err := findSocketInodes(port, true)
	if err != nil {
		fallbackInodes, fallbackErr := findSocketInodes(port, false)
		if fallbackErr != nil {
			return nil, err
		}
		inodes = fallbackInodes
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
				if !ok {
					continue
				}
				if socketInodeHasOwner(inodes, inode) {
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
		return nil, ownerUnknownError(inodes)
	}

	return result, nil
}

// ownerUnknownError is ErrSocketOwnerUnknown naming the users the port's
// sockets belong to: /proc/net records them even when the processes are
// hidden from this user.
func ownerUnknownError(inodes map[string]int) error {
	var users []string
	for _, uid := range inodes {
		if uid < 0 {
			continue
		}
		if name := procpkg.UserName(uid); !slices.Contains(users, name) {
			users = append(users, name)
		}
	}
	if len(users) == 0 {
		return ErrSocketOwnerUnknown
	}
	slices.Sort(users)
	return fmt.Errorf("%w; it belongs to %s", ErrSocketOwnerUnknown, strings.Join(users, ", "))
}
