//go:build windows

package proc

import (
	"fmt"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

const (
	scManagerEnumerateService uint32 = 0x0004
	scEnumProcessInfo         uint32 = 0
	serviceWin32              uint32 = 0x00000030
	serviceStateAll           uint32 = 0x00000003
)

var (
	modadvapi32               = syscall.NewLazyDLL("advapi32.dll")
	procOpenSCManagerW        = modadvapi32.NewProc("OpenSCManagerW")
	procCloseServiceHandle    = modadvapi32.NewProc("CloseServiceHandle")
	procEnumServicesStatusExW = modadvapi32.NewProc("EnumServicesStatusExW")
)

type serviceStatusProcess struct {
	ServiceType             uint32
	CurrentState            uint32
	ControlsAccepted        uint32
	Win32ExitCode           uint32
	ServiceSpecificExitCode uint32
	CheckPoint              uint32
	WaitHint                uint32
	ProcessId               uint32
	ServiceFlags            uint32
}

type enumServiceStatusProcessW struct {
	ServiceName          *uint16
	DisplayName          *uint16
	ServiceStatusProcess serviceStatusProcess
}

var (
	serviceMapCache     map[int]string
	serviceMapCacheTime time.Time
	serviceMapCacheMu   sync.Mutex
	serviceMapCacheTTL  = 2 * time.Second
)

// serviceMapForPIDs returns a PID → service-name map for every running
// Windows service. Cached so an ancestry walk pays one SCM scan, not N.
func serviceMapForPIDs() (map[int]string, error) {
	serviceMapCacheMu.Lock()
	defer serviceMapCacheMu.Unlock()

	if serviceMapCache != nil && time.Since(serviceMapCacheTime) < serviceMapCacheTTL {
		return serviceMapCache, nil
	}

	scm, _, callErr := procOpenSCManagerW.Call(0, 0, uintptr(scManagerEnumerateService))
	if scm == 0 {
		return nil, fmt.Errorf("OpenSCManager: %w", callErr)
	}
	defer procCloseServiceHandle.Call(scm)

	// First call with a zero buffer probes for the required size.
	var bytesNeeded, count, resume uint32
	procEnumServicesStatusExW.Call(
		scm,
		uintptr(scEnumProcessInfo),
		uintptr(serviceWin32),
		uintptr(serviceStateAll),
		0, 0,
		uintptr(unsafe.Pointer(&bytesNeeded)),
		uintptr(unsafe.Pointer(&count)),
		uintptr(unsafe.Pointer(&resume)),
		0,
	)
	if bytesNeeded == 0 {
		serviceMapCache = map[int]string{}
		serviceMapCacheTime = time.Now()
		return serviceMapCache, nil
	}

	buf := make([]byte, bytesNeeded)
	count = 0
	resume = 0
	ret, _, callErr := procEnumServicesStatusExW.Call(
		scm,
		uintptr(scEnumProcessInfo),
		uintptr(serviceWin32),
		uintptr(serviceStateAll),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(bytesNeeded),
		uintptr(unsafe.Pointer(&bytesNeeded)),
		uintptr(unsafe.Pointer(&count)),
		uintptr(unsafe.Pointer(&resume)),
		0,
	)
	if ret == 0 {
		return nil, fmt.Errorf("EnumServicesStatusEx: %w", callErr)
	}

	out := make(map[int]string, count)
	entrySize := unsafe.Sizeof(enumServiceStatusProcessW{})
	base := unsafe.Pointer(&buf[0])
	for i := uintptr(0); i < uintptr(count); i++ {
		entry := (*enumServiceStatusProcessW)(unsafe.Pointer(uintptr(base) + i*entrySize))
		pid := int(entry.ServiceStatusProcess.ProcessId)
		if pid == 0 {
			// Service registered but not currently running.
			continue
		}
		name := utf16PtrToString(entry.ServiceName)
		if name == "" {
			continue
		}
		// First writer wins so share-process hosts (svchost.exe) keep a
		// stable name across calls.
		if _, exists := out[pid]; !exists {
			out[pid] = name
		}
	}

	serviceMapCache = out
	serviceMapCacheTime = time.Now()
	return out, nil
}

// utf16PtrToString converts a null-terminated UTF-16 pointer to a Go string.
func utf16PtrToString(p *uint16) string {
	if p == nil {
		return ""
	}
	return syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(p))[:])
}
