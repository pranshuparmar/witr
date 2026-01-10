//go:build linux

package proc

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetFileContext returns file descriptor and lock info for a process
// Linux implementation - TODO: implement using /proc/<pid>/fd and /proc/locks
func GetFileContext(pid int) *model.FileContext {
	// Linux implementation could:
	ctx := &model.FileContext{}
	ctx.OpenFiles, ctx.FileLimit = getOpenFileCount(pid)
	ctx.LockedFiles = getLockedFiles(pid)
	ctx.WatchedDirs = getWatchedDirs(pid)
	if len(ctx.LockedFiles) > 0 {
		return ctx
	}
	if ctx.FileLimit > 0 && ctx.OpenFiles > 0 {
		usagePercent := float64(ctx.OpenFiles) / float64(ctx.FileLimit) * 100
		if usagePercent > 50 {
			return ctx
		}
	}

	return nil
}

// count /proc/<pid>/fd entries for open files
// read /proc/<pid>/limits for file limit
// count the # of Open files and maximum limit and hence usagePercent
func getOpenFileCount(pid int) (int, int) {

	out, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid))
	if err != nil {
		return 0, 0
	}
	openFiles := len(out)
	fileLimit := 0
	limitsData, err := os.ReadFile(fmt.Sprintf("/proc/%d/limits", pid))
	if err == nil {
		seq := strings.Split(string(limitsData), "\n")
		for _, line := range seq{
			if strings.HasPrefix(line, "Max open files") {
				// format: "Max open files   1024  524288  files"
				fields := strings.Fields(line)
				if len(fields) >= 4 {
					// fields[3] is the soft limit
					if limit, err := strconv.Atoi(fields[3]); err == nil {
						fileLimit = limit
					}
				}
				break
			}
		}
	}

	return openFiles, fileLimit
}

// get list of locked files by the process
func getLockedFiles(pid int) []string {
	lockedFileData, err := os.ReadFile("/proc/locks")
	if err != nil {
		return nil
	}

	var result []string
	// Output Pattern: <ID>: <TYPE> <ADVISORY/MANDATORY> <ACCESS> <PID> <DEVICE> <START> <END>
	pidStr := strconv.Itoa(pid)

	for _, line := range strings.Split(string(lockedFileData), "\n") {
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 8 {
			continue
		}

		// lockType := fields[1]    // FLOCK, POSIX, or OFDLCK
		lockPid := fields[4]     // PID that owns the lock
		deviceInode := fields[5] // Device:Inode identifier

		// consider POSIX locks (these have valid PIDs)
		// Skip OFDLCK as PID is -1 (owned by multiple processes)
		// skip FLOCK as it may not have valid PID association
		if (lockPid != strconv.Itoa(-1)) && lockPid == pidStr {
			// Store device:inode as identifier (resolving to file path would require scanning filesystem)
			if !slices.Contains(result, deviceInode) {
				result = append(result, deviceInode)
			}
		}
	}

	return result
}

// get list of directories being accessed by the process
// directories being watched/accessed (detectable via lsof)
func getWatchedDirs(pid int) []string {
  var result []string
	return result
}
