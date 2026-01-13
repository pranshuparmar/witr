package output

import (
	"io"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func PrintTree(w io.Writer, chain []model.Process, children []model.Process, colorEnabled bool) {
	p := NewPrinter(w)

	for i, proc := range chain {
		indent := strings.Repeat("  ", i)
		if i > 0 {
			if colorEnabled {
				p.Printf("%s%s└─ %s", indent, ColorMagenta, ColorReset)
			} else {
				p.Printf("%s└─ ", indent)
			}
		}

		if colorEnabled {
			cmdColor := ansiString("")
			if i == len(chain)-1 {
				cmdColor = ColorGreen
			}
			p.Printf("%s%s%s (%spid %d%s)\n", cmdColor, proc.Command, ColorReset, ColorBold, proc.PID, ColorReset)
		} else {
			p.Printf("%s (pid %d)\n", proc.Command, proc.PID)
		}
	}

	if len(children) == 0 {
		return
	}

	baseIndent := strings.Repeat("  ", len(chain))

	limit := 10
	count := len(children)
	for i, child := range children {
		if i >= limit {
			remaining := count - limit
			if colorEnabled {
				p.Printf("%s%s└─ %s... and %d more\n", baseIndent, ColorMagenta, ColorReset, remaining)
			} else {
				p.Printf("%s└─ ... and %d more\n", baseIndent, remaining)
			}
			break
		}

		connector := "├─ "
		isLast := (i == count-1) || (i == limit-1 && count <= limit)
		if isLast {
			connector = "└─ "
		}

		if colorEnabled {
			p.Printf("%s%s%s%s%s (%spid %d%s)\n", baseIndent, ColorMagenta, connector, ColorReset, child.Command, ColorBold, child.PID, ColorReset)
		} else {
			p.Printf("%s%s%s (pid %d)\n", baseIndent, connector, child.Command, child.PID)
		}
	}
}
