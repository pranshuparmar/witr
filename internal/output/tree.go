package output

import (
	"io"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorResetTree   = ansiString("\033[0m")
	colorMagentaTree = ansiString("\033[35m")
	colorGreenTree   = ansiString("\033[32m")
	colorBoldTree    = ansiString("\033[2m")
)

func PrintTree(w io.Writer, chain []model.Process, children []model.Process, colorEnabled bool) {
	p := NewPrinter(w)

	for i, proc := range chain {
		indent := strings.Repeat("  ", i)
		if i > 0 {
			if colorEnabled {
				p.Printf("%s%s└─ %s", indent, colorMagentaTree, colorResetTree)
			} else {
				p.Printf("%s└─ ", indent)
			}
		}

		if colorEnabled {
			cmdColor := ansiString("")
			if i == len(chain)-1 {
				cmdColor = colorGreenTree
			}
			p.Printf("%s%s%s (%spid %d%s)\n", cmdColor, proc.Command, colorResetTree, colorBoldTree, proc.PID, colorResetTree)
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
				p.Printf("%s%s└─ %s... and %d more\n", baseIndent, colorMagentaTree, colorResetTree, remaining)
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
			p.Printf("%s%s%s%s%s (%spid %d%s)\n", baseIndent, colorMagentaTree, connector, colorResetTree, child.Command, colorBoldTree, child.PID, colorResetTree)
		} else {
			p.Printf("%s%s%s (pid %d)\n", baseIndent, connector, child.Command, child.PID)
		}
	}
}
