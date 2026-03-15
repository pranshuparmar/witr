package source

import (
	"path/filepath"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

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

var userTools = map[string]bool{
	// Runtimes
	"python":  true,
	"python3": true,
	"node":    true,
	"ruby":    true,
	"perl":    true,
	"php":     true,
	"go":      true,
	"java":    true,
	"cargo":   true,
	"npm":     true,
	"yarn":    true,
	"make":    true,

	// Editors / IDEs
	"code":   true,
	"cursor": true,
	"vim":    true,
	"nvim":   true,
	"emacs":  true,
	"nano":   true,

	// Terminals
	"gnome-terminal-": true,
	"kitty":           true,
	"alacritty":       true,
	"wezterm":         true,
	"konsole":         true,
}

func detectShell(ancestry []model.Process) *model.Source {
	// Scan from the end (target) backwards to find the closest shell OR user tool
	// This ensures we get the direct parent rather than an ancestor
	for i := len(ancestry) - 2; i >= 0; i-- {
		cmd := ancestry[i].Command
		base := filepath.Base(cmd)

		if shells[base] {
			return &model.Source{
				Type: model.SourceShell,
				Name: base,
			}
		}

		// Normalize for Windows by stripping common executable extensions for the map lookup
		lookupName := base
		lowerBase := strings.ToLower(base)
		for _, ext := range []string{".exe", ".cmd", ".bat", ".com"} {
			if strings.HasSuffix(lowerBase, ext) {
				lookupName = strings.TrimSuffix(lowerBase, ext)
				break
			}
		}

		if userTools[lookupName] {
			return &model.Source{
				Type: model.SourceShell,
				Name: base,
			}
		}

		// Prefix matches for interpreters with versions or paths
		if strings.HasPrefix(base, "python") || strings.HasPrefix(base, "node") {
			return &model.Source{
				Type: model.SourceShell,
				Name: base,
			}
		}
	}
	return nil
}
