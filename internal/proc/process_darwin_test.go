//go:build darwin

package proc

import "testing"

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
