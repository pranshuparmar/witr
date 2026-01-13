package output

import (
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

func RenderShort(w io.Writer, r model.Result, colorEnabled bool) {
	p := NewPrinter(w)

	for i, proc := range r.Ancestry {
		if i > 0 {
			if colorEnabled {
				p.Printf("%s → %s", ColorMagenta, ColorReset)
			} else {
				p.Print(" → ")
			}
		}

		if colorEnabled {
			nameColor := ansiString("")
			if i == len(r.Ancestry)-1 {
				nameColor = ColorGreen
			}
			p.Printf("%s%s%s (%spid %d%s)", nameColor, proc.Command, ColorReset, ColorBold, proc.PID, ColorReset)
		} else {
			p.Printf("%s (pid %d)", proc.Command, proc.PID)
		}
	}
	p.Println()
}
