package output

import (
	"fmt"
	"io"
)

type ansiString string

// Printer writes terminal-safe output to an io.Writer
// sanitizing any string-like arguments (string, []byte, error, fmt.Stringer)
type Printer struct {
	w io.Writer
}

func NewPrinter(w io.Writer) Printer {
	return Printer{w: w}
}

func (p Printer) Printf(format string, args ...any) {
	fmt.Fprintf(p.w, format, sanitizePrintArgs(args)...)
}

func (p Printer) Print(args ...any) {
	fmt.Fprint(p.w, sanitizePrintArgs(args)...)
}

func (p Printer) Println(args ...any) {
	fmt.Fprintln(p.w, sanitizePrintArgs(args)...)
}

func sanitizePrintArgs(args []any) []any {
	if len(args) == 0 {
		return nil
	}
	out := make([]any, len(args))
	for i, a := range args {
		switch v := a.(type) {
		case ansiString: // our own ansiString type is allowed to render as-is
			out[i] = string(v)
		case string:
			out[i] = SanitizeTerminal(v)
		case []byte:
			out[i] = SanitizeTerminal(string(v))
		case error:
			out[i] = SanitizeTerminal(v.Error())
		case fmt.Stringer:
			out[i] = SanitizeTerminal(v.String())
		default:
			out[i] = a
		}
	}
	return out
}
