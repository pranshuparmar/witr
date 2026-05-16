package tui

import (
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`[\x1b\x9b][[\\]()#;?]*(?:(?:(?:[a-zA-Z\d]*(?:;[a-zA-Z\d]*)*)?[\x07])|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PRZcf-ntqry=><~]))`)

func stripAnsi(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

// normalizeRow collapses runs of whitespace into single spaces and trims
// leading/trailing whitespace. Used for matching a clicked rendered line
// against a stored table row: the renderer adds cell padding and may emit
// empty cells as runs of spaces, while strings.Join puts only single spaces
// between cells. Both sides normalize identically.
func normalizeRow(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
