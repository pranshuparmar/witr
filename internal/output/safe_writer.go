package output

import "io"

// SafeTerminalWriter sanitizes all bytes written to it so the output is safe to
// display in an interactive terminal, it should be used to print anything that
// we don't control (like processes' args, env vars, ...)
type SafeTerminalWriter struct {
	W io.Writer
}

func (w SafeTerminalWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	_, err := io.WriteString(w.W, SanitizeTerminal(string(p)))
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

func NewSafeTerminalWriter(w io.Writer) io.Writer {
	return SafeTerminalWriter{W: w}
}
