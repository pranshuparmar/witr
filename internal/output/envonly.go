package output

import (
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

// RenderEnvOnly prints only the command and environment variables for a process
func RenderEnvOnly(w io.Writer, r model.Result, colorEnabled bool) {
	p := NewPrinter(w)

	colorResetEnv := ansiString("")
	colorBlueEnv := ansiString("")
	colorRedEnv := ansiString("")
	colorGreenEnv := ansiString("")
	colorDimEnv := ansiString("")
	if colorEnabled {
		colorResetEnv = ColorReset
		colorBlueEnv = ColorBlue
		colorRedEnv = ColorRed
		colorGreenEnv = ColorGreen
		colorDimEnv = ColorDim
	}

	procName := r.Process.Command
	if len(r.Ancestry) > 0 {
		procName = r.Ancestry[len(r.Ancestry)-1].Command
	}

	if colorEnabled {
		p.Printf("%sProcess%s     : %s%s%s (%spid %d%s)\n", colorBlueEnv, colorResetEnv, colorGreenEnv, procName, colorResetEnv, colorDimEnv, r.Process.PID, colorResetEnv)
	} else {
		p.Printf("Process     : %s (pid %d)\n", procName, r.Process.PID)
	}

	p.Printf("%sCommand%s     : %s\n", colorGreenEnv, colorResetEnv, r.Process.Cmdline)
	if len(r.Process.Env) > 0 {
		p.Printf("%sEnvironment%s :\n", colorBlueEnv, colorResetEnv)
		for _, env := range r.Process.Env {
			p.Printf("  %s\n", env)
		}
	} else {
		p.Printf("%sEnvironment%s : %sNo environment variables found.%s\n", colorBlueEnv, colorResetEnv, colorRedEnv, colorResetEnv)
	}
}
