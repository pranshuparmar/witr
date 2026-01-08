package output

import (
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

func PrintChildren(w io.Writer, root model.Process, children []model.Process, colorEnabled bool) {
	p := NewPrinter(w)

	rootName := root.Command
	if rootName == "" && root.Cmdline != "" {
		rootName = root.Cmdline
	}
	if rootName == "" {
		rootName = "unknown"
	}

	if colorEnabled {
		p.Printf("%sChildren%s of %s (%spid %d%s):\n", colorMagentaTree, colorResetTree, rootName, colorBoldTree, root.PID, colorResetTree)
	} else {
		p.Printf("Children of %s (pid %d):\n", rootName, root.PID)
	}

	if len(children) == 0 {
		if colorEnabled {
			p.Printf("%sNo child processes found.%s\n", colorGreen, colorReset)
		} else {
			p.Println("No child processes found.")
		}
		return
	}

	limit := 10
	count := len(children)
	for i, child := range children {
		if i >= limit {
			remaining := count - limit
			if colorEnabled {
				p.Printf("  %s└─ %s... and %d more\n", colorMagentaTree, colorResetTree, remaining)
			} else {
				p.Printf("  └─ ... and %d more\n", remaining)
			}
			break
		}

		connector := "├─ "
		isLast := (i == count-1) || (i == limit-1 && count <= limit)
		if isLast {
			connector = "└─ "
		}

		childName := child.Command
		if childName == "" && child.Cmdline != "" {
			childName = child.Cmdline
		}
		if childName == "" {
			childName = "unknown"
		}

		if colorEnabled {
			p.Printf("  %s%s%s%s (%spid %d%s)\n", colorMagentaTree, connector, colorResetTree, childName, colorBoldTree, child.PID, colorResetTree)
		} else {
			p.Printf("  %s%s (pid %d)\n", connector, childName, child.PID)
		}
	}
}
