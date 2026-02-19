package tui

import "regexp"

func stripAnsi(str string) string {
	ansi := regexp.MustCompile(`[\x1b\x9b][[\\]()#;?]*(?:(?:(?:[a-zA-Z\d]*(?:;[a-zA-Z\d]*)*)?[\x07])|(?:(?:\d{1,4}(?:;\d{0,4})*)?[\dA-PRZcf-ntqry=><~]))`)
	return ansi.ReplaceAllString(str, "")
}
