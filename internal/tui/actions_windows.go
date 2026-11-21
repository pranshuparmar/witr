//go:build windows

package tui

import (
	"fmt"

	"golang.org/x/sys/windows"
)

const actionsSupported = true

// stillActive is the exit code GetExitCodeProcess reports for a running
// process (STATUS_PENDING).
const stillActive = 0x00000103

// killProcess and termProcess both end the process with TerminateProcess.
// Windows has no asynchronous graceful-termination signal for an arbitrary
// process, so the graceful variant is not attempted; the kill action maps to
// TerminateProcess as well.
func killProcess(pid int) error { return terminateProcess(pid) }
func termProcess(pid int) error { return terminateProcess(pid) }

func terminateProcess(pid int) error {
	h, err := windows.OpenProcess(windows.PROCESS_TERMINATE|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("open PID %d for terminate failed: %w", pid, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if err := checkAlive(h, pid); err != nil {
		return err
	}
	if err := windows.TerminateProcess(h, 1); err != nil {
		return fmt.Errorf("terminating PID %d failed: %w", pid, err)
	}
	return nil
}

// NtSuspendProcess and NtResumeProcess are ntdll exports; Windows exposes no
// documented API to suspend or resume all threads of another process at once.
var (
	ntdll                = windows.NewLazySystemDLL("ntdll.dll")
	procNtSuspendProcess = ntdll.NewProc("NtSuspendProcess")
	procNtResumeProcess  = ntdll.NewProc("NtResumeProcess")
)

func pauseProcess(pid int) error  { return suspendOrResume(pid, procNtSuspendProcess) }
func resumeProcess(pid int) error { return suspendOrResume(pid, procNtResumeProcess) }

func suspendOrResume(pid int, proc *windows.LazyProc) error {
	h, err := windows.OpenProcess(windows.PROCESS_SUSPEND_RESUME|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("open PID %d for %s failed: %w", pid, proc.Name, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if err := checkAlive(h, pid); err != nil {
		return err
	}
	if r1, _, ntErr := proc.Call(uintptr(h)); r1 != 0 {
		// An NTSTATUS is not an errno; report it verbatim.
		return fmt.Errorf("%s PID %d failed: %v", proc.Name, pid, ntErr)
	}
	return nil
}

// setNice maps the Unix renice action onto Windows priority classes. Windows
// has no nice scale, so a −20…19 value is mapped onto the class ladder and the
// exact value is not recoverable afterwards.
func setNice(pid, value int) error {
	if value < -20 || value > 19 {
		return fmt.Errorf("nice value %d out of range (−20…19)", value)
	}
	h, err := windows.OpenProcess(windows.PROCESS_SET_INFORMATION|windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return fmt.Errorf("open PID %d for renice failed: %w", pid, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	if err := checkAlive(h, pid); err != nil {
		return err
	}
	if err := windows.SetPriorityClass(h, priorityClassFor(value)); err != nil {
		return fmt.Errorf("renice PID %d to %d failed: %w", pid, value, err)
	}
	return nil
}

// checkAlive rejects actions on a process that has already exited: a terminated
// process object can outlive the process briefly, and acting on it must fail
// the same way signalling a reaped PID fails on Unix.
func checkAlive(h windows.Handle, pid int) error {
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil {
		return fmt.Errorf("PID %d: %w", pid, err)
	}
	if code != stillActive {
		return fmt.Errorf("PID %d has already exited", pid)
	}
	return nil
}

// priorityClassFor maps a nice value to the priority class with the closest
// scheduling pressure. More negative means more CPU priority, matching nice
// semantics. REALTIME_PRIORITY_CLASS is deliberately never used: it can starve
// the rest of the system, and no nice value grants that level.
func priorityClassFor(value int) uint32 {
	switch {
	case value <= -10:
		return windows.HIGH_PRIORITY_CLASS
	case value < 0:
		return windows.ABOVE_NORMAL_PRIORITY_CLASS
	case value == 0:
		return windows.NORMAL_PRIORITY_CLASS
	case value <= 10:
		return windows.BELOW_NORMAL_PRIORITY_CLASS
	default:
		return windows.IDLE_PRIORITY_CLASS
	}
}
