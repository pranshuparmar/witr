package source

import (
	"strings"

	"github.com/SanCognition/witr/pkg/model"
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
	"init":         "init",
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
	for _, p := range ancestry {
		// Normalize: remove spaces, lowercase
		pname := strings.ReplaceAll(strings.ToLower(p.Command), " ", "")
		pcmd := strings.ReplaceAll(strings.ToLower(p.Cmdline), " ", "")
		if strings.Contains(pname, "pm2") || strings.Contains(pcmd, "pm2") {
			return &model.Source{
				Type:       model.SourceSupervisor,
				Name:       "pm2",
				Confidence: 0.9,
			}
		}
		if label, ok := knownSupervisors[strings.ToLower(p.Command)]; ok {
			return &model.Source{
				Type:       model.SourceSupervisor,
				Name:       label,
				Confidence: 0.7,
			}
		}
		// Also match on command line for supervisor keywords
		for sup, label := range knownSupervisors {
			if strings.Contains(strings.ToLower(p.Cmdline), sup) {
				return &model.Source{
					Type:       model.SourceSupervisor,
					Name:       label,
					Confidence: 0.7,
				}
			}
		}
	}
	return nil
}
