//go:build linux

package proc

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func socketsForPID(pid int) []string {
	var inodes []string
	seen := make(map[string]bool)
	fdPath := "/proc/" + strconv.Itoa(pid) + "/fd"

	entries, err := os.ReadDir(fdPath)
	if err != nil {
		return inodes
	}

	for _, e := range entries {
		link, err := os.Readlink(filepath.Join(fdPath, e.Name()))
		if err != nil {
			continue
		}

		if strings.HasPrefix(link, "socket:[") {
			inode := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
			if !seen[inode] {
				seen[inode] = true
				inodes = append(inodes, inode)
			}
		}
	}

	return inodes
}
