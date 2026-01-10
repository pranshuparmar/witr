//go:build windows

package proc

import (
	"syscall"
	"unsafe"
)

// Win32 API constants and structures
const (
	PROCESS_QUERY_INFORMATION = 0x0400
	PROCESS_VM_READ           = 0x0010
)

var (
	modntdll           = syscall.NewLazyDLL("ntdll.dll")
	procNtQueryInfo    = modntdll.NewProc("NtQueryInformationProcess")
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	procReadProcessMem = modkernel32.NewProc("ReadProcessMemory")
)

type processBasicInformation struct {
	Reserved1       uintptr
	PebBaseAddress  uintptr
	Reserved2       [2]uintptr
	UniqueProcessId uintptr
	Reserved3       uintptr
}

type unicodeString struct {
	Length        uint16
	MaximumLength uint16
	Buffer        uintptr
}

// Partial RTL_USER_PROCESS_PARAMETERS
type rtlUserProcessParameters struct {
	Reserved1              [16]byte
	Reserved2              [10]uintptr
	CurrentDirectoryPath   unicodeString
	CurrentDirectoryHandle uintptr
	DllPath                unicodeString
	ImagePathName          unicodeString
	CommandLine            unicodeString
	Environment            uintptr
}

func readPEBData(pid int) (string, []string) {
	handle, err := syscall.OpenProcess(PROCESS_QUERY_INFORMATION|PROCESS_VM_READ, false, uint32(pid))
	if err != nil {
		return "unknown", []string{}
	}
	defer syscall.CloseHandle(handle)

	var pbi processBasicInformation
	var returnLength uint32
	status, _, _ := procNtQueryInfo.Call(
		uintptr(handle),
		0, // ProcessBasicInformation
		uintptr(unsafe.Pointer(&pbi)),
		uintptr(unsafe.Sizeof(pbi)),
		uintptr(unsafe.Pointer(&returnLength)),
	)

	if status != 0 || pbi.PebBaseAddress == 0 {
		return "unknown", []string{}
	}

	// Read PEB
	var pebPtr uintptr
	// PebBaseAddress + offset to ProcessParameters (0x20 on x64, 0x10 on x86)
	// For simplicity and 64-bit focus:
	paramsOffset := uintptr(0x20)
	if unsafe.Sizeof(uintptr(0)) == 4 {
		paramsOffset = 0x10
	}

	if !readProcessMemory(handle, pbi.PebBaseAddress+paramsOffset, unsafe.Pointer(&pebPtr), unsafe.Sizeof(pebPtr)) {
		return "unknown", []string{}
	}

	var params rtlUserProcessParameters
	if !readProcessMemory(handle, pebPtr, unsafe.Pointer(&params), unsafe.Sizeof(params)) {
		return "unknown", []string{}
	}

	wd := readUnicodeString(handle, params.CurrentDirectoryPath)
	// Environment is more complex to read as it's a block of null-terminated strings
	// For now we'll return the WD. Env reading involves scanning until double null.

	return wd, []string{}
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
