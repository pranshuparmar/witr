package source

import (
	"fmt"
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
			src := &model.Source{
				Type: model.SourceShell,
				Name: base,
			}
			enrichMultiplexer(src, ancestry)
			return src
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
			src := &model.Source{
				Type: model.SourceShell,
				Name: base,
			}
			enrichMultiplexer(src, ancestry)
			return src
		}

		// Prefix matches for interpreters with versions or paths
		if strings.HasPrefix(base, "python") || strings.HasPrefix(base, "node") {
			src := &model.Source{
				Type: model.SourceShell,
				Name: base,
			}
			enrichMultiplexer(src, ancestry)
			return src
		}
	}
	return nil
}

// enrichMultiplexer checks if tmux or screen is in the ancestry and adds
// session details to the source description.
func enrichMultiplexer(src *model.Source, ancestry []model.Process) {
	for i := 0; i < len(ancestry)-1; i++ {
		base := filepath.Base(ancestry[i].Command)

		if base == "tmux" || strings.HasPrefix(base, "tmux:") {
			session := findEnvVar(ancestry, "TMUX")
			desc := "tmux session"
			if session != "" {
				// TMUX env var format: /tmp/tmux-1000/default,12345,0
				// The session name is between the last "/" and the first ","
				if parts := strings.Split(session, ","); len(parts) >= 1 {
					path := parts[0]
					if idx := strings.LastIndex(path, "/"); idx >= 0 {
						desc = fmt.Sprintf("tmux session '%s'", path[idx+1:])
					}
				}
			}
			src.Description = desc
			return
		}

		if base == "screen" || strings.HasPrefix(base, "SCREEN") {
			session := findEnvVar(ancestry, "STY")
			desc := "screen session"
			if session != "" {
				desc = fmt.Sprintf("screen session '%s'", session)
			}
			src.Description = desc
			return
		}
	}
}

// findEnvVar searches the ancestry chain (target first) for an environment variable.
func findEnvVar(ancestry []model.Process, key string) string {
	for i := len(ancestry) - 1; i >= 0; i-- {
		for _, entry := range ancestry[i].Env {
			k, v, ok := strings.Cut(entry, "=")
			if ok && k == key {
				return v
			}
		}
	}
	return ""
}
