//go:build windows

package proc

import (
	"os"
	"testing"
)

func TestReadExtendedInfoSelf(t *testing.T) {
	mem, _, _, fdCount, _, threadCount, err := ReadExtendedInfo(os.Getpid())
	if err != nil {
		t.Fatalf("ReadExtendedInfo(self): %v", err)
	}
	if mem.RSS == 0 {
		t.Errorf("RSS = 0, want > 0")
	}
	if mem.RSSMB <= 0 {
		t.Errorf("RSSMB = %v, want > 0", mem.RSSMB)
	}
	if mem.VMS == 0 {
		t.Errorf("VMS = 0, want > 0")
	}
	if threadCount < 1 {
		t.Errorf("threadCount = %d, want >= 1", threadCount)
	}
	if fdCount < 1 {
		t.Errorf("fdCount = %d, want >= 1", fdCount)
	}
}

func TestReadExtendedInfoNonexistentPID(t *testing.T) {
	if _, _, _, _, _, _, err := ReadExtendedInfo(0); err == nil {
		t.Errorf("ReadExtendedInfo(0) returned no error; want OpenProcess failure")
	}
}
