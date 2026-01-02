package output

import (
	"fmt"

	"github.com/SanCognition/witr/pkg/model"
)

var (
	colorResetTree   = "\033[0m"
	colorMagentaTree = "\033[35m"
	colorBoldTree    = "\033[2m"
)

func PrintTree(chain []model.Process, colorEnabled bool) {
	colorReset := ""
	colorMagenta := ""
	colorBold := ""
	if colorEnabled {
		colorReset = colorResetTree
		colorMagenta = colorMagentaTree
		colorBold = colorBoldTree
	}
	for i, p := range chain {
		prefix := ""
		for j := 0; j < i; j++ {
			prefix += "  "
		}
		if i > 0 {
			if colorEnabled {
				prefix += colorMagenta + "└─ " + colorReset
			} else {
				prefix += "└─ "
			}
		}
		if colorEnabled {
			fmt.Printf("%s%s (%spid %d%s)\n", prefix, p.Command, colorBold, p.PID, colorReset)
		} else {
			fmt.Printf("%s%s (pid %d)\n", prefix, p.Command, p.PID)
		}
	}
}
