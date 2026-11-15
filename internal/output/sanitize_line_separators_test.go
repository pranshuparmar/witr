package output

import "testing"

// TestSanitizeTerminalLineEscapesUnicodeLineSeparators pins the single-line
// contract: U+2028 (LINE SEPARATOR) and U+2029 (PARAGRAPH SEPARATOR) can break
// a one-line field just like an embedded newline, so they must be rendered as
// visible escape sequences rather than emitted as real separators.
func TestSanitizeTerminalLineEscapesUnicodeLineSeparators(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"line separator", "a\u2028b", `a\u2028b`},
		{"paragraph separator", "a\u2029b", `a\u2029b`},
		{"leading separator", "\u2028header", `\u2028header`},
		{"mixed with newline", "a\nb\u2028c", `a\nb\u2028c`},
		{"unchanged plain text", "plain", "plain"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SanitizeTerminalLine(tc.in); got != tc.want {
				t.Fatalf("SanitizeTerminalLine(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
