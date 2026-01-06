//go:build linux

package proc

import (
	"fmt"
	"os"
	"os/exec"
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
	var locked []string
	output, err := exec.Command("lslocks", "-o", "PATH", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return locked
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

	return locked
}
