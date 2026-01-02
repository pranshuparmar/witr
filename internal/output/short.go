package output

import (
	"fmt"

	"github.com/SanCognition/witr/pkg/model"
)

var (
	colorResetShort   = "\033[0m"
	colorMagentaShort = "\033[35m"
	colorBoldShort    = "\033[2m"
)

func RenderShort(r model.Result, colorEnabled bool) {
	for i, p := range r.Ancestry {
		if i > 0 {
			if colorEnabled {
				fmt.Print(colorMagentaShort + " → " + colorResetShort)
			} else {
				fmt.Print(" → ")
			}
		}
		if colorEnabled {
			fmt.Printf("%s (%spid %d%s)", p.Command, colorBoldShort, p.PID, colorResetShort)
		} else {
			fmt.Printf("%s (pid %d)", p.Command, p.PID)
		}
	}
	fmt.Println()
}
