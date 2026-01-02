//go:build darwin

package proc

import (
	"os/exec"
	"slices"
	"strconv"
	"strings"

	"github.com/SanCognition/witr/pkg/model"
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
	// Use lsof to count open files
	// lsof -p <pid> returns all open files
	out, err := exec.Command("lsof", "-p", strconv.Itoa(pid)).Output()
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

	// Get file limit using launchctl or ulimit
	fileLimit := getFileLimit(pid)

	return openFiles, fileLimit
}

// getFileLimit returns the file descriptor limit for a process
func getFileLimit(pid int) int {
	// Try to get per-process limit
	// On macOS, we can use launchctl limit or check /proc equivalent
	// Default to common macOS limits

	// Try launchctl limit (system-wide soft limit)
	out, err := exec.Command("launchctl", "limit", "maxfiles").Output()
	if err == nil {
		// Format: maxfiles    256            unlimited
		fields := strings.Fields(string(out))
		if len(fields) >= 2 {
			limit, err := strconv.Atoi(fields[1])
			if err == nil {
				return limit
			}
		}
	}

	// Default macOS limit
	return 256
}

// getLockedFiles returns files with locks held by the process
func getLockedFiles(pid int) []string {
	var locked []string

	// Use lsof to find locked files
	// -p <pid> for specific process
	// Look for lock indicators in the output
	out, err := exec.Command("lsof", "-p", strconv.Itoa(pid), "-F", "fn").Output()
	if err != nil {
		return locked
	}

	// Parse lsof -F output
	// f = file descriptor info
	// n = file name
	var currentFD string
	for line := range strings.Lines(string(out)) {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case 'f':
			currentFD = line[1:]
		case 'n':
			fileName := line[1:]
			// Check if this FD indicates a lock
			// Common lock indicators: .lock files, fcntl locks shown with 'l' type
			if strings.HasSuffix(fileName, ".lock") ||
				strings.HasSuffix(fileName, ".pid") ||
				strings.Contains(fileName, "/lock") {
				if !slices.Contains(locked, fileName) {
					locked = append(locked, fileName)
				}
			}
			_ = currentFD // Used for future lock type detection
		}
	}

	// Also check for actual fcntl/flock locks using lsof -F with lock info
	out2, err := exec.Command("lsof", "-p", strconv.Itoa(pid)).Output()
	if err == nil {
		for line := range strings.Lines(string(out2)) {
			fields := strings.Fields(line)
			// Look for lock type indicators (varies by lsof version)
			// Typically shows "r" for read lock, "w" for write lock, "R" for read lock on entire file
			if len(fields) >= 5 {
				lockType := fields[4]
				if lockType == "r" || lockType == "w" || lockType == "R" || lockType == "W" {
					// This file has a lock
					if len(fields) >= 9 {
						fileName := fields[8]
						if !slices.Contains(locked, fileName) {
							locked = append(locked, fileName)
						}
					}
				}
			}
		}
	}

	return locked
}
