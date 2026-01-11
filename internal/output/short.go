package output

import (
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetShort   = ansiString("\033[0m")
	colorMagentaShort = ansiString("\033[35m")
	colorGreenShort   = ansiString("\033[32m")
	colorBoldShort    = ansiString("\033[2m")
)

func RenderShort(w io.Writer, r model.Result, colorEnabled bool) {
	p := NewPrinter(w)

	for i, proc := range r.Ancestry {
		if i > 0 {
			if colorEnabled {
				p.Printf("%s → %s", colorMagentaShort, colorResetShort)
			} else {
				p.Print(" → ")
			}
		}

		if colorEnabled {
			nameColor := ansiString("")
			if i == len(r.Ancestry)-1 {
				nameColor = colorGreenShort
			}
			p.Printf("%s%s%s (%spid %d%s)", nameColor, proc.Command, colorResetShort, colorBoldShort, proc.PID, colorResetShort)
		} else {
			p.Printf("%s (pid %d)", proc.Command, proc.PID)
		}
	}
	p.Println()
}
