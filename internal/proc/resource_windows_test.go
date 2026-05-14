//go:build windows

package proc

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestFiletimeTicksToDuration(t *testing.T) {
	tests := []struct {
		name string
		high uint32
		low  uint32
		want time.Duration
	}{
		{"zero", 0, 0, 0},
		{"one microsecond", 0, 10, time.Microsecond},
		{"one second", 0, 10_000_000, time.Second},
		{"high word only", 1, 0, time.Duration(uint64(1)<<32) * 100 * time.Nanosecond},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ft := syscall.Filetime{HighDateTime: tt.high, LowDateTime: tt.low}
			if got := filetimeTicksToDuration(ft); got != tt.want {
				t.Errorf("filetimeTicksToDuration(%d:%d) = %v, want %v",
					tt.high, tt.low, got, tt.want)
			}
		})
	}
}

func TestGetResourceContextSelf(t *testing.T) {
	rc := GetResourceContext(os.Getpid())
	if rc == nil {
		t.Fatalf("GetResourceContext(self) = nil")
	}
	if rc.MemoryUsage == 0 {
		t.Errorf("MemoryUsage = 0, want > 0")
	}
	if rc.CPUUsage < 0 {
		t.Errorf("CPUUsage = %v, want >= 0", rc.CPUUsage)
	}
	if rc.CPUUsage > 10000 {
		t.Errorf("CPUUsage = %v, suspiciously high", rc.CPUUsage)
	}
}

func TestGetResourceContextNonexistentPID(t *testing.T) {
	if got := GetResourceContext(0); got != nil {
		t.Errorf("GetResourceContext(0) = %+v, want nil", got)
	}
}
