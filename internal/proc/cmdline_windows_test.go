//go:build windows

package proc

import (
	"os"
	"strings"
	"testing"
)

func TestGetCmdlineSelf(t *testing.T) {
	got := GetCmdline(os.Getpid())
	if got == "" || got == "(unknown)" {
		t.Errorf("GetCmdline(self) = %q, want a non-empty exe basename", got)
	}
	if !strings.HasSuffix(strings.ToLower(got), ".exe") {
		t.Errorf("GetCmdline(self) = %q, expected a .exe suffix", got)
	}
}

func TestGetCmdlineNonexistentPID(t *testing.T) {
	if got := GetCmdline(2147483646); got != "(unknown)" {
		t.Errorf("GetCmdline(large-pid) = %q, want %q", got, "(unknown)")
	}
}
