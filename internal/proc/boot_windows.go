//go:build windows

package proc

import "time"

var procGetTickCount64 = modkernel32.NewProc("GetTickCount64")

// bootTime derives the system boot time from milliseconds since startup.
func bootTime() time.Time {
	ret, _, _ := procGetTickCount64.Call()
	uptime := time.Duration(ret) * time.Millisecond
	return time.Now().Add(-uptime)
}
