//go:build windows

package tui

import (
	"os"
	"os/exec"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

func TestSetNiceRangeGuardWindows(t *testing.T) {
	// Out-of-range values are rejected before any syscall is attempted, mirroring
	// the Unix guard.
	if err := setNice(os.Getpid(), 100); err == nil {
		t.Error("setNice(+100) should be rejected as out of range")
	}
	if err := setNice(os.Getpid(), -100); err == nil {
		t.Error("setNice(-100) should be rejected as out of range")
	}
}

// startSleeperWindows spawns a long-lived child to receive the actions.
func startSleeperWindows(t *testing.T) *exec.Cmd {
	t.Helper()
	c := exec.Command("ping", "-n", "120", "127.0.0.1")
	if err := c.Start(); err != nil {
		t.Skipf("cannot spawn child process: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	})
	return c
}

// waitExitWindows reports whether the child exits within the timeout.
func waitExitWindows(c *exec.Cmd, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() { _ = c.Wait(); close(done) }()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func TestWindowsTerminateAndKill(t *testing.T) {
	t.Run("term ends the process", func(t *testing.T) {
		c := startSleeperWindows(t)
		if err := termProcess(c.Process.Pid); err != nil {
			t.Fatalf("termProcess: %v", err)
		}
		if !waitExitWindows(c, 5*time.Second) {
			t.Fatal("process did not exit after term")
		}
	})

	t.Run("kill ends the process", func(t *testing.T) {
		c := startSleeperWindows(t)
		if err := killProcess(c.Process.Pid); err != nil {
			t.Fatalf("killProcess: %v", err)
		}
		if !waitExitWindows(c, 5*time.Second) {
			t.Fatal("process did not exit after kill")
		}
	})
}

func TestWindowsPauseResume(t *testing.T) {
	// ping emits one reply per second, so the output flow is the suspend
	// oracle: a paused process stops writing, a resumed one starts again.
	c := exec.Command("ping", "-n", "120", "127.0.0.1")
	stdout, err := c.StdoutPipe()
	if err != nil {
		t.Skipf("cannot open stdout pipe: %v", err)
	}
	if err := c.Start(); err != nil {
		t.Skipf("cannot spawn child process: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	})

	var total int64
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				atomic.AddInt64(&total, int64(n))
			}
			if err != nil {
				return
			}
		}
	}()

	deadline := time.Now().Add(10 * time.Second)
	for atomic.LoadInt64(&total) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("child never produced output")
		}
		time.Sleep(20 * time.Millisecond)
	}

	if err := pauseProcess(c.Process.Pid); err != nil {
		t.Fatalf("pauseProcess: %v", err)
	}
	time.Sleep(1500 * time.Millisecond) // drain whatever was already in the pipe
	frozen := atomic.LoadInt64(&total)
	time.Sleep(3 * time.Second)
	if got := atomic.LoadInt64(&total); got != frozen {
		t.Errorf("output grew from %d to %d while paused", frozen, got)
	}

	if err := resumeProcess(c.Process.Pid); err != nil {
		t.Fatalf("resumeProcess: %v", err)
	}
	deadline = time.Now().Add(5 * time.Second)
	for atomic.LoadInt64(&total) == frozen {
		if time.Now().After(deadline) {
			t.Fatal("output did not resume after resume")
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestWindowsDeadProcessErrors(t *testing.T) {
	c := startSleeperWindows(t)
	pid := c.Process.Pid
	_ = c.Process.Kill()
	_ = c.Wait() // close the handle, so the PID no longer refers to a live process

	if err := resumeProcess(pid); err == nil {
		t.Error("resuming a dead PID should fail")
	}
	if err := termProcess(pid); err == nil {
		t.Error("terminating a dead PID should fail")
	}
}

// priorityClassOf reads the priority class of the given process.
func priorityClassOf(t *testing.T, pid int) uint32 {
	t.Helper()
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		t.Fatalf("open PID %d: %v", pid, err)
	}
	defer func() { _ = windows.CloseHandle(h) }()
	class, err := windows.GetPriorityClass(h)
	if err != nil {
		t.Fatalf("GetPriorityClass: %v", err)
	}
	return class
}

func TestWindowsSetNiceMapsToPriorityClass(t *testing.T) {
	c := startSleeperWindows(t)

	// Each rung of the nice ladder maps to one priority class.
	tests := []struct {
		nice  int
		class uint32
	}{
		{-10, windows.HIGH_PRIORITY_CLASS},
		{-1, windows.ABOVE_NORMAL_PRIORITY_CLASS},
		{0, windows.NORMAL_PRIORITY_CLASS},
		{1, windows.BELOW_NORMAL_PRIORITY_CLASS},
		{11, windows.IDLE_PRIORITY_CLASS},
	}
	for _, tt := range tests {
		if err := setNice(c.Process.Pid, tt.nice); err != nil {
			t.Fatalf("setNice(%d): %v", tt.nice, err)
		}
		if got := priorityClassOf(t, c.Process.Pid); got != tt.class {
			t.Errorf("setNice(%d) class = %#x, want %#x", tt.nice, got, tt.class)
		}
	}
}

func TestWindowsSetNiceOnDeadProcessErrors(t *testing.T) {
	c := startSleeperWindows(t)
	pid := c.Process.Pid
	_ = c.Process.Kill()
	_ = c.Wait()

	if err := setNice(pid, 5); err == nil {
		t.Error("renice on a dead PID should fail")
	}
}
