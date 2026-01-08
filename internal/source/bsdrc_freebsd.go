//go:build freebsd

package source

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	shellCache     map[string]bool
	shellCacheOnce sync.Once
)

// loadShellsFromEtc reads /etc/shells and returns a map of valid shells
func loadShellsFromEtc() map[string]bool {
	shells := make(map[string]bool)

	// Fallback list in case /etc/shells is not readable
	fallback := []string{"sh", "bash", "zsh", "csh", "tcsh", "ksh", "fish", "dash"}
	for _, s := range fallback {
		shells[s] = true
	}

	data, err := os.ReadFile("/etc/shells")
	if err != nil {
		return shells
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		shellName := filepath.Base(line)
		shells[shellName] = true
	}

	return shells
}

func getShells() map[string]bool {
	shellCacheOnce.Do(func() {
		shellCache = loadShellsFromEtc()
	})
	return shellCache
}

func detectBsdRc(ancestry []model.Process) *model.Source {
	// Priority 1: Check for explicit service detection via /var/run/*.pid
	for _, p := range ancestry {
		if p.Service != "" {
			return &model.Source{
				Type: model.SourceBsdRc,
				Name: p.Service,
				Details: map[string]string{
					"service": p.Service,
				},
			}
		}
	}

	// Priority 2: Check if target process is a direct child of init
	// without any shell in the ancestry (likely an rc.d service)
	if len(ancestry) >= 2 {
		target := ancestry[len(ancestry)-1]
		shells := getShells()

		hasShell := false
		for i := 0; i < len(ancestry)-1; i++ {
			if shells[ancestry[i].Command] {
				hasShell = true
				break
			}
		}

		if target.PPID == 1 && !hasShell {
			return &model.Source{
				Type: model.SourceBsdRc,
				Name: "bsdrc",
			}
		}
	}

	return nil
}
