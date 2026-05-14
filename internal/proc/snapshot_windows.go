//go:build windows

package proc

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type processSnapshot struct {
	PID     int
	PPID    int
	Exe     string
	Threads int
}

var (
	snapshotCache     []processSnapshot
	snapshotCacheTime time.Time
	snapshotCacheMu   sync.Mutex
	snapshotCacheTTL  = 1 * time.Second
)

// enumerateProcesses returns every running process via ToolHelp32. Cached so
// repeated calls within a render pass reuse a single snapshot.
func enumerateProcesses() ([]processSnapshot, error) {
	snapshotCacheMu.Lock()
	defer snapshotCacheMu.Unlock()

	if snapshotCache != nil && time.Since(snapshotCacheTime) < snapshotCacheTTL {
		return snapshotCache, nil
	}

	snap, _, _ := procCreateToolhelp32Snapshot.Call(uintptr(TH32CS_SNAPPROCESS), 0)
	if syscall.Handle(snap) == syscall.InvalidHandle {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot failed")
	}
	defer syscall.CloseHandle(syscall.Handle(snap))

	var pe32 PROCESSENTRY32
	pe32.Size = uint32(unsafe.Sizeof(pe32))

	ret, _, _ := procProcess32First.Call(snap, uintptr(unsafe.Pointer(&pe32)))
	if ret == 0 {
		return nil, fmt.Errorf("Process32First failed")
	}

	var out []processSnapshot
	for {
		out = append(out, processSnapshot{
			PID:     int(pe32.ProcessID),
			PPID:    int(pe32.ParentProcessID),
			Exe:     syscall.UTF16ToString(pe32.ExeFile[:]),
			Threads: int(pe32.CntThreads),
		})
		ret, _, _ = procProcess32Next.Call(snap, uintptr(unsafe.Pointer(&pe32)))
		if ret == 0 {
			break
		}
	}

	snapshotCache = out
	snapshotCacheTime = time.Now()
	return out, nil
}
