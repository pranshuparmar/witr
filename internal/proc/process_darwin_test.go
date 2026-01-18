//go:build darwin

package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDeriveDisplayCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		comm    string
		cmdline string
		want    string
	}{
		{
			name:    "falls back to executable when ps truncates name",
			comm:    "AccessibilityVis",
			cmdline: "/System/Library/PrivateFrameworks/AccessibilitySupport.framework/Versions/A/Resources/AccessibilityVisualsAgent.app/Contents/MacOS/AccessibilityVisualsAgent",
			want:    "AccessibilityVisualsAgent",
		},
		{
			name:    "keeps comm when executable does not share prefix",
			comm:    "python3",
			cmdline: "python3 /tmp/script.py",
			want:    "python3",
		},
		{
			name:    "uses executable when comm empty",
			comm:    "",
			cmdline: "\"/Applications/App Name/MyBinary\" --flag",
			want:    "MyBinary",
		},
		{
			name:    "ignores env assignments before executable",
			comm:    "AccessibilityUIServer",
			cmdline: "PATH=/usr/bin /System/Library/CoreServices/AccessibilityUIServer.app/Contents/MacOS/AccessibilityUIServer",
			want:    "AccessibilityUIServer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := deriveDisplayCommand(tt.comm, tt.cmdline); got != tt.want {
				t.Fatalf("deriveDisplayCommand(%q, %q) = %q, want %q", tt.comm, tt.cmdline, got, tt.want)
			}
		})
	}
}

func TestExtractExecutableName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmdline string
		want    string
	}{
		{
			name:    "handles quoted path with spaces",
			cmdline: "\"/Applications/Visual Tool.app/Contents/MacOS/Visual Tool\" --flag",
			want:    "Visual Tool",
		},
		{
			name:    "skips env assignment tokens",
			cmdline: "FOO=bar BAR=baz /usr/local/bin/server --mode production",
			want:    "server",
		},
		{
			name:    "returns empty when no executable found",
			cmdline: "",
			want:    "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := extractExecutableName(tt.cmdline); got != tt.want {
				t.Fatalf("extractExecutableName(%q) = %q, want %q", tt.cmdline, got, tt.want)
			}
		})
	}
}

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
