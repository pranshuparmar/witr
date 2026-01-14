package source

import (
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

var knownSupervisors = map[string]string{
	"pm2":          "pm2",
	"pm2 god":      "pm2",
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
		if shells[p.Command] {
			hasShell = true
			break
		}
	}

	for _, p := range ancestry {
		// Normalize: remove spaces, lowercase
		pname := strings.ReplaceAll(strings.ToLower(p.Command), " ", "")
		pcmd := strings.ReplaceAll(strings.ToLower(p.Cmdline), " ", "")
		if strings.Contains(pname, "pm2") || strings.Contains(pcmd, "pm2") {
			return &model.Source{
				Type: model.SourceSupervisor,
				Name: "pm2",
			}
		}

		// Special handling for init to avoid false positives
		// Only match if command is exactly "init" or "/sbin/init" etc
		if p.Command == "init" || strings.HasSuffix(p.Command, "/init") {
			// Skip "init" if there's a shell in the ancestry
			if !hasShell {
				return &model.Source{
					Type: model.SourceSupervisor,
					Name: "init",
				}
			}
		}

		if label, ok := knownSupervisors[strings.ToLower(p.Command)]; ok {
			// Skip "init" if there's a shell in the ancestry
			// This allows shell-launched processes to be detected as shell rather than init
			if label == "init" && hasShell {
				continue
			}
			return &model.Source{
				Type: model.SourceSupervisor,
				Name: label,
			}
		}
		// Also match on command line for supervisor keywords
		for sup, label := range knownSupervisors {
			if strings.Contains(strings.ToLower(p.Cmdline), sup) {
				// Skip "init" if there's a shell in the ancestry
				if label == "init" && hasShell {
					continue
				}
				return &model.Source{
					Type: model.SourceSupervisor,
					Name: label,
				}
			}
		}
	}
	return nil
}
