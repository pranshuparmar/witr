package source

import "github.com/pranshuparmar/witr/pkg/model"

var shells = map[string]bool{
	"bash":           true,
	"zsh":            true,
	"sh":             true,
	"fish":           true,
	"csh":            true,
	"tcsh":           true,
	"ksh":            true,
	"dash":           true,
	"cmd.exe":        true,
	"powershell.exe": true,
	"pwsh.exe":       true,
	"explorer.exe":   true,
}

func detectShell(ancestry []model.Process) *model.Source {
	// Scan from the end (target) backwards to find the closest shell
	// This ensures we get the direct parent shell rather than an ancestor shell
	for i := len(ancestry) - 1; i >= 0; i-- {
		if shells[ancestry[i].Command] {
			return &model.Source{
				Type: model.SourceShell,
				Name: ancestry[i].Command,
			}
		}
	}
	return nil
}
