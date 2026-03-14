package proc

import (
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
		{
			name:    "recovers truncated Linux comm (15 char limit)",
			comm:    "my-very-long-pr",
			cmdline: "/usr/local/bin/my-very-long-process-name --daemon",
			want:    "my-very-long-process-name",
		},
		{
			name:    "handles nginx-style cmdline where first token has colon",
			comm:    "nginx",
			cmdline: "nginx: master process /usr/sbin/nginx",
			want:    "nginx:",
		},
		{
			name:    "keeps comm when it matches exe exactly",
			comm:    "nginx",
			cmdline: "/usr/sbin/nginx -g daemon off;",
			want:    "nginx",
		},
		{
			name:    "returns comm when both empty",
			comm:    "",
			cmdline: "",
			want:    "",
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
		{
			name:    "handles simple command",
			cmdline: "/usr/bin/my-very-long-process-name --flag",
			want:    "my-very-long-process-name",
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
