package output

import (
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

// RenderEnvOnly prints only the command and environment variables for a process
func RenderEnvOnly(w io.Writer, proc model.Process, colorEnabled bool) {
	p := NewPrinter(w)

	colorResetEnv := ansiString("")
	colorBlueEnv := ansiString("")
	colorRedEnv := ansiString("")
	colorGreenEnv := ansiString("")
	if colorEnabled {
		colorResetEnv = ansiString("\033[0m")
		colorBlueEnv = ansiString("\033[34m")
		colorRedEnv = ansiString("\033[31m")
		colorGreenEnv = ansiString("\033[32m")
	}

	p.Printf("%sCommand%s     : %s\n", colorGreenEnv, colorResetEnv, proc.Cmdline)
	if len(proc.Env) > 0 {
		p.Printf("%sEnvironment%s :\n", colorBlueEnv, colorResetEnv)
		for _, env := range proc.Env {
			p.Printf("  %s\n", env)
		}
	} else {
		p.Printf("%sNo environment variables found.%s\n", colorRedEnv, colorResetEnv)
	}
}
