//go:build linux

package proc

import "github.com/SanCognition/witr/pkg/model"

// GetResourceContext returns resource usage context for a process
// Linux implementation - TODO: implement using /proc and cgroup info
func GetResourceContext(pid int) *model.ResourceContext {
	// Linux implementation could check:
	// - /proc/<pid>/oom_score for memory pressure
	// - cgroup CPU throttling
	// - thermal zone info from /sys/class/thermal
	return nil
}
