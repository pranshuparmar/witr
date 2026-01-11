package source

import (
	"path/filepath"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// detectInit checks if the process is a direct descendant of the init process (PID 1)
// effectively acting as a catch-all for SysVinit, OpenRC, or other init systems.
func detectInit(ancestry []model.Process) *model.Source {
	if len(ancestry) == 0 {
		return nil
	}

	root := ancestry[0]
	if root.PID != 1 {
		return nil
	}

	// Check if there's any shell in the chain between root and target.
	// If there is a shell, it's likely a manual command run by a user or script, not a pure service.

	hasShell := false
	for i := 1; i < len(ancestry)-1; i++ {
		name := strings.ToLower(filepath.Base(ancestry[i].Command))
		if isShell(name) {
			hasShell = true
			break
		}
	}

	if !hasShell {
		return &model.Source{
			Type: model.SourceInit,
			Name: "init",
			Details: map[string]string{
				"pid":  "1",
				"comm": root.Command,
			},
		}
	}

	return nil
}

func isShell(name string) bool {
	switch name {
	case "sh", "bash", "zsh", "dash", "ash", "csh", "tcsh", "fish", "powershell.exe", "pwsh.exe", "cmd.exe":
		return true
	}
	return false
}
