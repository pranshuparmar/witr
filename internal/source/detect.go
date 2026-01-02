package source

import (
	"time"

	"github.com/SanCognition/witr/pkg/model"
)

func Detect(ancestry []model.Process) model.Source {
	// Prefer supervisor over systemd/launchd if both are present
	if src := detectContainer(ancestry); src != nil {
		return *src
	}
	if src := detectSupervisor(ancestry); src != nil {
		return *src
	}
	if src := detectSystemd(ancestry); src != nil {
		return *src
	}
	if src := detectLaunchd(ancestry); src != nil {
		return *src
	}
	if src := detectCron(ancestry); src != nil {
		return *src
	}
	if src := detectShell(ancestry); src != nil {
		return *src
	}

	return model.Source{
		Type:       model.SourceUnknown,
		Confidence: 0.2,
	}
}

func Warnings(p []model.Process) []string {
	var w []string

	last := p[len(p)-1]

	// Restart count detection (count consecutive same-command entries)
	restartCount := 0
	lastCmd := ""
	for _, proc := range p {
		if proc.Command == lastCmd {
			restartCount++
		}
		lastCmd = proc.Command
	}
	if restartCount > 5 {
		w = append(w, "Process or ancestor restarted more than 5 times")
	}

	// Health warnings
	switch last.Health {
	case "zombie":
		w = append(w, "Process is a zombie (defunct)")
	case "stopped":
		w = append(w, "Process is stopped (T state)")
	case "high-cpu":
		w = append(w, "Process is using high CPU (>2h total)")
	case "high-mem":
		w = append(w, "Process is using high memory (>1GB RSS)")
	}

	if IsPublicBind(last.BindAddresses) {
		w = append(w, "Process is listening on a public interface")
	}

	if last.User == "root" {
		w = append(w, "Process is running as root")
	}

	if Detect(p).Type == model.SourceUnknown {
		w = append(w, "No known supervisor or service manager detected")
	}

	// Warn if process is very old (>90 days)
	if time.Since(last.StartedAt).Hours() > 90*24 {
		w = append(w, "Process has been running for over 90 days")
	}

	// Warn if working dir is suspicious
	suspiciousDirs := map[string]bool{"/": true, "/tmp": true, "/var/tmp": true}
	if suspiciousDirs[last.WorkingDir] {
		w = append(w, "Process is running from a suspicious working directory: "+last.WorkingDir)
	}

	// Warn if container and no healthcheck (placeholder, as healthcheck not detected)
	if last.Container != "" {
		w = append(w, "No healthcheck detected for container (best effort)")
	}

	// Warn if service name and process name mismatch
	if last.Service != "" && last.Command != "" && last.Service != last.Command {
		w = append(w, "Service name and process name do not match")
	}

	return w
}
