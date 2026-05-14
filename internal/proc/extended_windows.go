//go:build windows

package proc

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	modpsapi                  = syscall.NewLazyDLL("psapi.dll")
	procGetProcessMemoryInfo  = modpsapi.NewProc("GetProcessMemoryInfo")
	procGetProcessIoCounters  = modkernel32.NewProc("GetProcessIoCounters")
	procGetProcessHandleCount = modkernel32.NewProc("GetProcessHandleCount")
)

// processMemoryCountersEx mirrors PROCESS_MEMORY_COUNTERS_EX. The CB field
// must be set to sizeof(struct) before GetProcessMemoryInfo so Windows can
// distinguish it from the non-EX variant.
type processMemoryCountersEx struct {
	CB                         uint32
	PageFaultCount             uint32
	PeakWorkingSetSize         uintptr
	WorkingSetSize             uintptr
	QuotaPeakPagedPoolUsage    uintptr
	QuotaPagedPoolUsage        uintptr
	QuotaPeakNonPagedPoolUsage uintptr
	QuotaNonPagedPoolUsage     uintptr
	PagefileUsage              uintptr
	PeakPagefileUsage          uintptr
	PrivateUsage               uintptr
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

// ReadExtendedInfo returns memory, I/O, thread, and handle counters for a
// PID. File descriptors and FD limit are zero-valued on Windows.
func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, int, error) {
	var memInfo model.MemoryInfo
	var ioStats model.IOStats
	var threadCount int
	var fdCount int

	// Full access first, falling back to limited access for protected processes.
	handle, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		handle, err = syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
		if err != nil {
			return memInfo, ioStats, nil, 0, 0, 0, fmt.Errorf("OpenProcess(%d): %w", pid, err)
		}
	}
	defer syscall.CloseHandle(handle)

	var pmc processMemoryCountersEx
	pmc.CB = uint32(unsafe.Sizeof(pmc))
	if ret, _, _ := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&pmc)),
		uintptr(pmc.CB),
	); ret != 0 {
		memInfo.RSS = uint64(pmc.WorkingSetSize)
		memInfo.RSSMB = float64(memInfo.RSS) / (1024 * 1024)
		memInfo.VMS = uint64(pmc.PrivateUsage)
		memInfo.VMSMB = float64(memInfo.VMS) / (1024 * 1024)
	}

	var io ioCounters
	if ret, _, _ := procGetProcessIoCounters.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&io)),
	); ret != 0 {
		ioStats.ReadOps = io.ReadOperationCount
		ioStats.ReadBytes = io.ReadTransferCount
		ioStats.WriteOps = io.WriteOperationCount
		ioStats.WriteBytes = io.WriteTransferCount
	}

	var hCount uint32
	if ret, _, _ := procGetProcessHandleCount.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&hCount)),
	); ret != 0 {
		fdCount = int(hCount)
	}

	if procs, err := enumerateProcesses(); err == nil {
		for _, p := range procs {
			if p.PID == pid {
				threadCount = p.Threads
				break
			}
		}
	}

	return memInfo, ioStats, nil, fdCount, 0, threadCount, nil
}
