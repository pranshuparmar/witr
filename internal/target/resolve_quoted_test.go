package target

import (
	"reflect"
	"testing"
)

// Windows command lines quote arguments with double quotes. An apostrophe in
// an earlier argument is literal data and must not open a quote.
func TestSplitWindowsCmdline(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		want    []string
	}{
		{
			name:    "quoted path with spaces",
			cmdline: `python.exe "C:\Program Files\My App\main.py" --flag`,
			want:    []string{"python.exe", `C:\Program Files\My App\main.py`, "--flag"},
		},
		{
			name:    "apostrophe in earlier argument is literal",
			cmdline: `cmd.exe /c "C:\Users\O'Brien\app.exe" --arg`,
			want:    []string{"cmd.exe", "/c", `C:\Users\O'Brien\app.exe`, "--arg"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := splitWindowsCmdline(tt.cmdline); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("splitWindowsCmdline(%q) = %q, want %q", tt.cmdline, got, tt.want)
			}
		})
	}
}

// On Unix the command line is argv joined with spaces. An apostrophe in an
// earlier argument is just another character and must not hide later tokens.
func TestMatchesExactTokenUnixApostrophe(t *testing.T) {
	cmdline := `sh -c 'echo it works' /usr/bin/python3`
	if !matchesExactToken(cmdline, "python3") {
		t.Errorf("matchesExactToken(%q, %q) = false, want true", cmdline, "python3")
	}
	if matchesExactToken(cmdline, "works") {
		t.Errorf("matchesExactToken(%q, %q) = true, want false (quoted prose is not a token)", cmdline, "works")
	}
}
