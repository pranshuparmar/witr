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
			src := &model.Source{
				Type: model.SourceBsdRc,
				Name: p.Service,
				Details: map[string]string{
					"service": p.Service,
				},
			}

			if path := resolveRcScript(p.Service); path != "" {
				src.UnitFile = path
				src.Description = readRcDescription(path)
			}

			return src
		}
	}

	// Priority 2: Check if target process is a direct child of init
	// without any shell in the ancestry (likely an rc.d service)
	if len(ancestry) >= 2 {
		target := ancestry[len(ancestry)-1]
		shells := getShells()

		hasShell := false
		for i := 0; i < len(ancestry)-1; i++ {
			if shells[filepath.Base(ancestry[i].Command)] {
				hasShell = true
				break
			}
		}

		if target.PPID == 1 && !hasShell {
			// Try to guess service name from command if not explicitly set
			name := target.Command
			path := resolveRcScript(name)

			src := &model.Source{
				Type: model.SourceBsdRc,
				Name: name,
			}

			if path != "" {
				src.UnitFile = path
				src.Description = readRcDescription(path)
			}

			return src
		}
	}

	return nil
}

func readRcDescription(path string) string {
	if path == "" {
		return ""
	}
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	// Simple heuristic: look for "# description:" or similar
	buf := make([]byte, 2048)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return ""
	}
	content := string(buf[:n])
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		lower := strings.ToLower(line)
		if strings.HasPrefix(lower, "# description:") {
			return strings.TrimSpace(line[14:]) // len("# description:") is 14
		}
		if strings.HasPrefix(lower, "# desc:") {
			return strings.TrimSpace(line[7:])
		}
	}
	return ""
}

func resolveRcScript(serviceName string) string {
	paths := []string{
		"/etc/rc.d/" + serviceName,
		"/usr/local/etc/rc.d/" + serviceName,
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
