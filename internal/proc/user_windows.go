//go:build windows

package proc

import (
	"fmt"
	"syscall"
)

func readUser(pid int) string {
	// 1. Open Process
	// PROCESS_QUERY_LIMITED_INFORMATION (0x1000) is enough for Token and allows better access coverage
	// than PROCESS_QUERY_INFORMATION (0x0400).
	const PROCESS_QUERY_LIMITED_INFORMATION = 0x1000
	hProcess, err := syscall.OpenProcess(PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		// Fallback to PROCESS_QUERY_INFORMATION if LIMITED not available or failed?
		// Usually 0x400 is stricter.
		// Try standard if fails?
		hProcess, err = syscall.OpenProcess(syscall.PROCESS_QUERY_INFORMATION, false, uint32(pid))
		if err != nil {
			return "unknown"
		}
	}
	defer syscall.CloseHandle(hProcess)

	// 2. Open Process Token
	var token syscall.Token
	err = syscall.OpenProcessToken(hProcess, syscall.TOKEN_QUERY, &token)
	if err != nil {
		return "unknown"
	}
	defer token.Close()

	// 3. Get Token User (SID)
	tokenUser, err := token.GetTokenUser()
	if err != nil {
		return "unknown"
	}

	// 4. Lookup Account SID
	// tokenUser.User.Sid is *syscall.SID
	user, domain, _, err := tokenUser.User.Sid.LookupAccount("")
	if err != nil {
		return "unknown"
	}

	if domain == "" {
		return user
	}
	return fmt.Sprintf("%s\\%s", domain, user)
}

// Helpers if needed, but syscall.Token has GetTokenUser and lookup methods in standard library (windows)
