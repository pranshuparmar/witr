//go:build freebsd

package proc

import (
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetFileContext returns file descriptor and lock info for a process
func GetFileContext(pid int) *model.FileContext {
	ctx := &model.FileContext{}

	// Get open file count
	openFiles, fileLimit := getOpenFileCount(pid)
	ctx.OpenFiles = openFiles
	ctx.FileLimit = fileLimit

	// Get locked files
	ctx.LockedFiles = getLockedFiles(pid)

	// Only return if we have meaningful data to show
	// Show if: high file usage (>50% of limit) or has locks
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

// getOpenFileCount returns the number of open files and the limit for a process
func getOpenFileCount(pid int) (int, int) {
	// Use fstat to count open files
	out, err := exec.Command("fstat", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return 0, 0
	}

	// Count lines (subtract 1 for header)
	openFiles := 0
	for line := range strings.Lines(string(out)) {
		if strings.TrimSpace(line) != "" {
			openFiles++
		}
	}
	if openFiles > 0 {
		openFiles-- // Subtract header line
	}

	// Get file limit using sysctl or limits
	fileLimit := getFileLimit(pid)

	return openFiles, fileLimit
}

// getFileLimit returns the file descriptor limit for a process
func getFileLimit(pid int) int {
	// Try procstat to get limits
	out, err := exec.Command("procstat", "-l", strconv.Itoa(pid)).Output()
	if err == nil {
		// Parse procstat -l output for openfiles limit
		for line := range strings.Lines(string(out)) {
			if strings.Contains(line, "openfiles") {
				fields := strings.Fields(line)
				if len(fields) >= 3 {
					limit, err := strconv.Atoi(fields[2])
					if err == nil {
						return limit
					}
				}
			}
		}
	}

	// Fallback: get system-wide limit
	out, err = exec.Command("sysctl", "-n", "kern.maxfilesperproc").Output()
	if err == nil {
		limit, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err == nil {
			return limit
		}
	}

	// Default FreeBSD limit
	return 1024
}

// getLockedFiles returns files with locks held by the process
func getLockedFiles(pid int) []string {
	var locked []string

	// Use fstat to find file locks
	out, err := exec.Command("fstat", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return locked
	}

	for line := range strings.Lines(string(out)) {
		// Look for lock indicators in the output
		if strings.Contains(line, ".lock") ||
			strings.Contains(line, ".pid") ||
			strings.Contains(line, "/lock") {
			fields := strings.Fields(line)
			if len(fields) >= 8 {
				fileName := fields[len(fields)-1]
				if !slices.Contains(locked, fileName) {
					locked = append(locked, fileName)
				}
			}
		}
	}

	return locked
}
