//go:build linux

package proc

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"syscall"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetFileContext returns file descriptor and lock info for a process
// Will return nil if the context could not be gathered.
func GetFileContext(pid int) *model.FileContext {
	var fileContext model.FileContext

	// Count /proc/<pid>/fd entries for open files.
	fdFiles, err := os.ReadDir(fmt.Sprintf("/proc/%v/fd", pid))
	if err != nil {
		return nil
	}

	fileContext.OpenFiles = len(fdFiles)
	fileContext.FileLimit = getFileLimit(pid)
	fileContext.LockedFiles = getLockedFiles(pid)
	fileContext.WatchedDirs = getWatchedDirs(pid)

	return &fileContext
}

func getFileLimit(pid int) int {
	var linuxDefaultMaxOpenFile = getDefaultMaxOpenFiles()

	// Read /proc/<pid>/limits for file limit
	data, err := os.ReadFile(fmt.Sprintf("/proc/%v/limits", pid))
	if err != nil {
		return linuxDefaultMaxOpenFile
	}

	dataString := string(data)
	for line := range strings.Lines(dataString) {
		if !strings.HasPrefix(line, "Max open files") {
			continue
		}

		// Data in format: "Max open files $SOFT_LOCK_NUMBER $HARD_LOCK_NUMBER files"
		fields := strings.Fields(line)
		softLimitString := fields[3]

		if softLimitString == "unlimited" {
			return 0
		}

		softLimit, err := strconv.Atoi(softLimitString)
		if err != nil {
			return linuxDefaultMaxOpenFile
		}

		return softLimit
	}

	return linuxDefaultMaxOpenFile
}

func getDefaultMaxOpenFiles() int {
	// This seems to be a common default for many systems.
	const reasonableDefault int = 1024

	// https://www.man7.org/linux/man-pages/man2/getrlimit.2.html
	var rlimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlimit)
	if err != nil {
		return reasonableDefault
	}

	return int(rlimit.Max)
}

func getLockedFiles(pid int) []string {
	files, err := getLockedFilesLslocks(pid)
	if errors.Is(err, exec.ErrNotFound) {
		return getLockedFilesProc(pid)
	}
	return files
}

func getLockedFilesLslocks(pid int) ([]string, error) {
	var locked []string
	output, err := exec.Command("lslocks", "-o", "PATH", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return nil, err
	}

	// First line of output is PATH (column name) which is not interesting.
	skippedFirst := false
	for fileName := range strings.Lines(string(output)) {
		if !skippedFirst {
			skippedFirst = true
			continue
		}

		locked = append(locked, strings.TrimSpace(fileName))
	}

	return locked, nil
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
		for _, line := range seq {
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
func getLockedFilesProc(pid int) []string {
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
