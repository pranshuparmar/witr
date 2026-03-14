//go:build darwin

package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestIsBinaryDeleted(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "fakebin")
	if err := os.WriteFile(binPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Create a fake lsof command that just prints the binPath set above.
	fakeBinDir := filepath.Join(tmpDir, "bin")
	if err := os.Mkdir(fakeBinDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	lsofPath := filepath.Join(fakeBinDir, "lsof")
	script := fmt.Sprintf("#!/bin/sh\nprintf 'p123\nn%s   \\n'", binPath)
	if err := os.WriteFile(lsofPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write lsof script: %v", err)
	}

	// Add our fake lsof to the start of the PATH.
	t.Setenv("PATH", fakeBinDir+":"+os.Getenv("PATH"))

	if got := isBinaryDeleted(123); got {
		t.Fatalf("isBinaryDeleted() = true, expected false")
	}
	if err := os.Remove(binPath); err != nil {
		t.Fatalf("rm: %v", err)
	}

	if got := isBinaryDeleted(123); !got {
		t.Fatalf("isBinaryDeleted() = false, expected true for deleted binary")
	}
}
