package target

import "testing"

func TestMatchesExactTokenQuotedPath(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		needle  string
		want    bool
	}{
		{"basename inside a quoted path", `/usr/bin/python3 "/opt/My App/main.py"`, "main.py", true},
		{"directory segment inside a quoted path", `/usr/bin/python3 "/opt/My App/main.py"`, "My App", true},
		{"bare quoted token", `cmd "some name"`, "some name", true},
		{"no match inside quoted path", `/usr/bin/python3 "/opt/My App/main.py"`, "other.py", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchesExactToken(tt.cmdline, tt.needle); got != tt.want {
				t.Errorf("matchesExactToken(%q, %q) = %v, want %v", tt.cmdline, tt.needle, got, tt.want)
			}
		})
	}
}
