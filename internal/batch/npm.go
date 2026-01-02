//go:build linux || darwin

package batch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type packageJSON struct {
	Name    string            `json:"name"`
	Scripts map[string]string `json:"scripts"`
}

// DetectNpmScript analyzes a Node process to find which npm script it's running.
// Returns the script name (e.g., "dev", "build", "test:watch") or entry file.
func DetectNpmScript(cmdline string, workDir string) string {
	// Strategy 1: Parse cmdline for "npm run <script>" pattern
	if idx := strings.Index(cmdline, "npm run "); idx != -1 {
		rest := cmdline[idx+8:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}

	// Strategy 2: Parse cmdline for "yarn <script>" pattern (not install/add)
	if idx := strings.Index(cmdline, "yarn "); idx != -1 {
		rest := cmdline[idx+5:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			script := parts[0]
			// Skip yarn commands that aren't scripts
			if script != "install" && script != "add" && script != "remove" && script != "upgrade" {
				return "yarn:" + script
			}
		}
	}

	// Strategy 3: Parse cmdline for "pnpm run <script>" or "pnpm <script>" pattern
	if idx := strings.Index(cmdline, "pnpm run "); idx != -1 {
		rest := cmdline[idx+9:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			return parts[0]
		}
	}
	if idx := strings.Index(cmdline, "pnpm "); idx != -1 {
		rest := cmdline[idx+5:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			script := parts[0]
			if script != "install" && script != "add" && script != "remove" && script != "update" {
				return "pnpm:" + script
			}
		}
	}

	// Strategy 4: Parse cmdline for "npx <command>" pattern
	if idx := strings.Index(cmdline, "npx "); idx != -1 {
		rest := cmdline[idx+4:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			cmd := parts[0]
			// Remove version specifier if present (e.g., "tsx@latest" -> "tsx")
			if atIdx := strings.Index(cmd, "@"); atIdx > 0 {
				cmd = cmd[:atIdx]
			}
			return "npx:" + cmd
		}
	}

	// Strategy 5: Look for package.json and match command to scripts
	if workDir != "" {
		pkgPath := filepath.Join(workDir, "package.json")
		if pkg, err := readPackageJSON(pkgPath); err == nil {
			// Try to match cmdline against known scripts
			for name, script := range pkg.Scripts {
				// Check if the script command appears in the cmdline
				if strings.Contains(cmdline, script) {
					return name
				}
			}
		}
	}

	// Strategy 6: Extract entry file from cmdline
	// e.g., "node server.js" → "server.js"
	// e.g., "node dist/index.js" → "dist/index.js"
	// e.g., "/usr/local/bin/node /path/to/script.js" → "script.js"
	if idx := strings.LastIndex(cmdline, "node "); idx != -1 {
		rest := cmdline[idx+5:]
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			file := parts[0]
			// Skip flags
			if strings.HasPrefix(file, "-") {
				for _, p := range parts[1:] {
					if !strings.HasPrefix(p, "-") {
						file = p
						break
					}
				}
			}
			// Get just the filename if it's an absolute path
			if filepath.IsAbs(file) {
				file = filepath.Base(file)
			}
			// Remove workdir prefix if present
			if workDir != "" {
				file = strings.TrimPrefix(file, workDir+"/")
			}
			if file != "" && !strings.HasPrefix(file, "-") {
				return file
			}
		}
	}

	return "-"
}

func readPackageJSON(path string) (*packageJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}
