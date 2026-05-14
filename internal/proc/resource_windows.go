//go:build windows

package proc

import (
	"syscall"
	"time"
	"unsafe"

	"github.com/pranshuparmar/witr/pkg/model"
)

// GetResourceContext returns CPU and memory usage for a process.
//
// CPU usage is the lifetime average — total kernel + user CPU time divided
// by wall-clock time since the process started — not an instantaneous %.
// Memory is the private commit (PrivateUsage).
func GetResourceContext(pid int) *model.ResourceContext {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return nil
	}
	defer syscall.CloseHandle(handle)

	var (
		cpu float64
		mem uint64
	)

	var pmc processMemoryCountersEx
	pmc.CB = uint32(unsafe.Sizeof(pmc))
	if ret, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&pmc)),
		uintptr(pmc.CB),
	); ret != 0 {
		mem = uint64(pmc.PrivateUsage)
	}

	var creation, exit, kernel, user syscall.Filetime
	if ret, _, _ := procGetProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	); ret != 0 {
		startTime := time.Unix(0, creation.Nanoseconds())
		wall := time.Since(startTime)
		cpuTime := filetimeTicksToDuration(kernel) + filetimeTicksToDuration(user)
		if wall > 0 {
			cpu = float64(cpuTime) / float64(wall) * 100.0
		}
	}

	return &model.ResourceContext{
		CPUUsage:    cpu,
		MemoryUsage: mem,
	}
}

// filetimeTicksToDuration treats a Filetime as a count of 100-ns ticks (the
// shape kernel/user time take in GetProcessTimes), not as an absolute
// timestamp.
func filetimeTicksToDuration(ft syscall.Filetime) time.Duration {
	ticks := uint64(ft.HighDateTime)<<32 | uint64(ft.LowDateTime)
	return time.Duration(ticks) * 100 * time.Nanosecond
}
