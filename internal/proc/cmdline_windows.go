//go:build windows

package proc

// GetCmdline returns a best-effort identifier for a PID using the exe
// basename from the ToolHelp32 snapshot. Used as a fallback in multi-match
// output when ReadProcess itself failed.
func GetCmdline(pid int) string {
	procs, err := enumerateProcesses()
	if err != nil {
		return "(unknown)"
	}
	for _, p := range procs {
		if p.PID == pid && p.Exe != "" {
			return p.Exe
		}
	}
	return "(unknown)"
}
