package output

import (
	"fmt"
	"github.com/pranshuparmar/witr/pkg/model"
	"runtime"
)

// RenderEnvOnly prints only the command and environment variables for a process
func RenderEnvOnly(proc model.Process, colorEnabled bool) {
	colorResetEnv := ""
	colorBlueEnv := ""
	colorRedEnv := ""
	colorGreenEnv := ""
	colorYellowEnv := ""
	if colorEnabled {
		colorResetEnv = "\033[0m"
		colorBlueEnv = "\033[34m"
		colorRedEnv = "\033[31m"
		colorGreenEnv = "\033[32m"
		colorYellowEnv = "\033[33m"
	}
	fmt.Printf("%sCommand%s     : %s\n", colorGreenEnv, colorResetEnv, proc.Cmdline)
	if len(proc.Env) > 0 {
		fmt.Printf("%sEnvironment%s :\n", colorBlueEnv, colorResetEnv)
		for _, env := range proc.Env {
			fmt.Printf("  %s\n", env)
		}
	} else if runtime.GOOS == "darwin" {
		fmt.Printf("%sEnvironment variable extraction is not available on macOS.%s\n", colorYellowEnv, colorResetEnv)
	} else {
		fmt.Printf("%sNo environment variables found.%s\n", colorRedEnv, colorResetEnv)
	}
}
