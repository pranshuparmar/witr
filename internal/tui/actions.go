//go:build !windows

package tui

import (
	"fmt"
	"os"
	"syscall"
)

func killProcess(pid int) error   { return sendSignal(pid, syscall.SIGKILL) }
func termProcess(pid int) error   { return sendSignal(pid, syscall.SIGTERM) }
func pauseProcess(pid int) error  { return sendSignal(pid, syscall.SIGSTOP) }
func resumeProcess(pid int) error { return sendSignal(pid, syscall.SIGCONT) }

func sendSignal(pid int, sig syscall.Signal) error {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process %d not found: %w", pid, err)
	}
	if err := proc.Signal(sig); err != nil {
		return fmt.Errorf("signal %v to PID %d failed: %w", sig, pid, err)
	}
	return nil
}

// sets the scheduling priority (nice value) for the given PID.
func setNice(pid, value int) error {
	if value < -20 || value > 19 {
		return fmt.Errorf("nice value %d out of range (−20…19)", value)
	}
	if err := syscall.Setpriority(syscall.PRIO_PROCESS, pid, value); err != nil {
		return fmt.Errorf("renice PID %d to %d failed: %w", pid, value, err)
	}
	return nil
}
