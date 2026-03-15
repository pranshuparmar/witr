package source

import (
	"path/filepath"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

var knownSupervisors = map[string]string{
	"pm2":          "pm2",
	"supervisord":  "supervisord",
	"supervisor":   "supervisord",
	"gunicorn":     "gunicorn",
	"uwsgi":        "uwsgi",
	"s6-supervise": "s6",
	"s6":           "s6",
	"s6-svscan":    "s6",
	"runsv":        "runit",
	"runit":        "runit",
	"runit-init":   "runit",
	"openrc":       "openrc",
	"openrc-init":  "openrc",
	"monit":        "monit",
	"circusd":      "circus",
	"circus":       "circus",
	"systemd":      "systemd service",
	"systemctl":    "systemd service",
	"daemontools":  "daemontools",
	"initctl":      "upstart",
	"tini":         "tini",
	"docker-init":  "docker-init",
	"podman-init":  "podman-init",
	"smf":          "smf",
	"launchd":      "launchd",
	"god":          "god",
	"forever":      "forever",
	"nssm":         "nssm",
}

func detectSupervisor(ancestry []model.Process) *model.Source {
	// Check if there's a shell in the ancestry
	hasShell := false
	for _, p := range ancestry {
		if shells[filepath.Base(p.Command)] {
			hasShell = true
			break
		}
	}

	for _, p := range ancestry {
		base := filepath.Base(p.Command)
		if base == "init" {
			if !hasShell {
				return &model.Source{
					Type: model.SourceSupervisor,
					Name: "init",
				}
			}
		}

		if label, ok := knownSupervisors[strings.ToLower(base)]; ok {
			if label == "init" && hasShell {
				continue
			}
			return &model.Source{
				Type: model.SourceSupervisor,
				Name: label,
			}
		}
		// Match individual tokens from the cmdline against supervisor keys
		if label := matchCmdlineTokens(p.Cmdline, hasShell); label != "" {
			return &model.Source{
				Type: model.SourceSupervisor,
				Name: label,
			}
		}
	}
	return nil
}

// matchCmdlineTokens extracts the executable basename and each argument token
// from a command line, then looks up each against knownSupervisors by exact match.
func matchCmdlineTokens(cmdline string, hasShell bool) string {
	for _, token := range strings.Fields(strings.ToLower(cmdline)) {
		// Skip flags and env assignments
		if strings.HasPrefix(token, "-") || strings.Contains(token, "=") {
			continue
		}
		base := filepath.Base(token)
		if label, ok := knownSupervisors[base]; ok {
			if label == "init" && hasShell {
				continue
			}
			return label
		}
	}
	return ""
}
