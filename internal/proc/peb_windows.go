//go:build windows

package proc

import (
	"fmt"
	"syscall"
	"time"
	"unsafe"
)

// Win32 API constants and structures
const (
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010
)

var (
	modntdll            = syscall.NewLazyDLL("ntdll.dll")
	procNtQueryInfo     = modntdll.NewProc("NtQueryInformationProcess")
	modkernel32         = syscall.NewLazyDLL("kernel32.dll")
	procReadProcessMem  = modkernel32.NewProc("ReadProcessMemory")
	procGetProcessTimes = modkernel32.NewProc("GetProcessTimes")
)

type processBasicInformation struct {
	ExitStatus                   uintptr
	PebBaseAddress               uintptr
	AffinityMask                 uintptr
	BasePriority                 uintptr
	UniqueProcessId              uintptr
	InheritedFromUniqueProcessId uintptr
}

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        uintptr
}

// Partial RTL_USER_PROCESS_PARAMETERS
type rtlUserProcessParameters struct {
	Reserved1              [16]byte
	Reserved2              [5]uintptr
	CurrentDirectoryPath   unicodeString
	CurrentDirectoryHandle uintptr
	DllPath                unicodeString
	ImagePathName          unicodeString
	CommandLine            unicodeString
	Environment            uintptr
}

type Win32ProcessInfo struct {
	PPID        int
	CommandLine string
	Exe         string
	Cwd         string
	Env         []string
	StartedAt   time.Time
}

func GetProcessDetailedInfo(pid int) (Win32ProcessInfo, error) {
	var info Win32ProcessInfo
	handle, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return info, err
	}
	defer syscall.CloseHandle(handle)

	// Get Start Time
	info.StartedAt = getProcessStartTime(handle)

	var pbi processBasicInformation
	var returnLength uint32
	status, _, _ := procNtQueryInfo.Call(
		uintptr(handle),
		0, // ProcessBasicInformation
		uintptr(unsafe.Pointer(&pbi)),
		uintptr(unsafe.Sizeof(pbi)),
		uintptr(unsafe.Pointer(&returnLength)),
	)

	if status != 0 {
		return info, fmt.Errorf("NtQueryInformationProcess failed with status %x", status)
	}

	info.PPID = int(pbi.InheritedFromUniqueProcessId)

	if pbi.PebBaseAddress == 0 {
		return info, fmt.Errorf("PEB Base Address is 0")
	}

	// Read PEB
	var pebPtr uintptr
	// PebBaseAddress + offset to ProcessParameters (0x20 on x64, 0x10 on x86)
	paramsOffset := uintptr(0x20)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		paramsOffset = 0x10
	}

	if !readProcessMemory(handle, pbi.PebBaseAddress+paramsOffset, unsafe.Pointer(&pebPtr), unsafe.Sizeof(pebPtr)) {
		return info, fmt.Errorf("failed to read PEB ProcessParameters address")
	}

	var params rtlUserProcessParameters
	if !readProcessMemory(handle, pebPtr, unsafe.Pointer(&params), unsafe.Sizeof(params)) {
		return info, fmt.Errorf("failed to read ProcessParameters struct")
	}

	info.Cwd = readUnicodeString(handle, params.CurrentDirectoryPath)
	info.CommandLine = readUnicodeString(handle, params.CommandLine)
	info.Exe = readUnicodeString(handle, params.ImagePathName)

	// Environment is complex to read remotely (block of null-terminated strings)
	// Leaving empty for now as requested, matching previous behavior/capability level.
	info.Env = []string{}

	return info, nil
}

func readProcessMemory(handle syscall.Handle, addr uintptr, dest unsafe.Pointer, size uintptr) bool {
	var read uint32
	ret, _, _ := procReadProcessMem.Call(
		uintptr(handle),
		addr,
		uintptr(dest),
		size,
		uintptr(unsafe.Pointer(&read)),
	)
	return ret != 0
}

func readUnicodeString(handle syscall.Handle, us unicodeString) string {
	if us.Length == 0 {
		return ""
	}
	buf := make([]uint16, us.Length/2)
	if !readProcessMemory(handle, us.Buffer, unsafe.Pointer(&buf[0]), uintptr(us.Length)) {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func getProcessStartTime(handle syscall.Handle) time.Time {
	var creation, exit, kernel, user syscall.Filetime
	ret, _, _ := procGetProcessTimes.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&creation)),
		uintptr(unsafe.Pointer(&exit)),
		uintptr(unsafe.Pointer(&kernel)),
		uintptr(unsafe.Pointer(&user)),
	)
	if ret == 0 {
		return time.Time{}
	}
	return time.Unix(0, creation.Nanoseconds())
}
