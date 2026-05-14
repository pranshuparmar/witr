//go:build windows

package proc

import (
	"os"
	"testing"
)

func TestEnumerateProcessesIncludesSelfWithThreads(t *testing.T) {
	procs, err := enumerateProcesses()
	if err != nil {
		t.Fatalf("enumerateProcesses: %v", err)
	}

	self := os.Getpid()
	for _, p := range procs {
		if p.PID == self {
			if p.PPID == 0 {
				t.Errorf("self snapshot has PPID = 0")
			}
			if p.Threads < 1 {
				t.Errorf("self snapshot has Threads = %d, want >= 1", p.Threads)
			}
			if p.Exe == "" {
				t.Errorf("self snapshot has empty Exe field")
			}
			return
		}
	}
	t.Fatalf("enumerateProcesses did not include self PID %d (out of %d entries)", self, len(procs))
}

func TestEnumerateProcessesCacheReuse(t *testing.T) {
	first, err := enumerateProcesses()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	second, err := enumerateProcesses()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(first) != len(second) {
		t.Errorf("cached vs fresh snapshot size mismatch: %d vs %d", len(first), len(second))
	}
}
