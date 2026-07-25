//go:build windows

package proc

import (
	"os"
	"path/filepath"
)

// dockerBinaryCandidates returns a full path to docker.exe when it is installed
// in a common Docker Desktop location but missing from PATH.
func dockerBinaryCandidates() string {
	var roots []string
	if pf := os.Getenv("ProgramFiles"); pf != "" {
		roots = append(roots, pf)
	}
	if pf86 := os.Getenv("ProgramFiles(x86)"); pf86 != "" {
		roots = append(roots, pf86)
	}
	roots = append(roots,
		`C:\Program Files`,
		`C:\Program Files (x86)`,
	)
	// LocalAppData\Docker\wsl and user installs are less common for the CLI.
	if la := os.Getenv("LOCALAPPDATA"); la != "" {
		roots = append(roots, la)
	}

	rel := []string{
		filepath.Join("Docker", "Docker", "resources", "bin", "docker.exe"),
		filepath.Join("Docker", "Docker", "resources", "docker.exe"),
	}
	seen := make(map[string]struct{})
	for _, root := range roots {
		for _, r := range rel {
			p := filepath.Join(root, r)
			if _, dup := seen[p]; dup {
				continue
			}
			seen[p] = struct{}{}
			if st, err := os.Stat(p); err == nil && !st.IsDir() {
				return p
			}
		}
	}
	return ""
}
