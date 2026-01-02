//go:build linux

package proc

import "github.com/SanCognition/witr/pkg/model"

// GetFileContext returns file descriptor and lock info for a process
// Linux implementation - TODO: implement using /proc/<pid>/fd and /proc/locks
func GetFileContext(pid int) *model.FileContext {
	// Linux implementation could:
	// - Count /proc/<pid>/fd entries for open files
	// - Read /proc/<pid>/limits for file limit
	// - Parse /proc/locks for file locks
	return nil
}
