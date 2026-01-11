//go:build freebsd

package proc

import "github.com/pranshuparmar/witr/pkg/model"

// GetResourceContext returns resource usage context for a process
// FreeBSD implementation - basic support
func GetResourceContext(pid int) *model.ResourceContext {
	// FreeBSD doesn't have macOS-style power assertions or thermal monitoring
	// Could potentially check CPU temperature via sysctl dev.cpu.*.temperature
	// but this is not process-specific
	return nil
}
