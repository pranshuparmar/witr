package source

import (
	"sort"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

type envSuspiciousRule struct {
	pattern     string
	match       func(key, pattern string) bool
	warning     string
	includeKeys bool
}

var (
	envVarRules = []envSuspiciousRule{
		{
			pattern: "LD_PRELOAD",
			match:   func(key, pattern string) bool { return key == pattern },
			warning: "Process sets LD_PRELOAD (potential library injection)",
		},

		{
			pattern:     "DYLD_",
			match:       strings.HasPrefix,
			warning:     "Process sets DYLD_* variables (potential library injection)",
			includeKeys: true,
		},
	}
)

func Detect(ancestry []model.Process) model.Source {
	// Detection order prioritizes platform-specific init systems
	// over generic supervisor detection to avoid false positives
	if src := detectContainer(ancestry); src != nil {
		return *src
	}
	if src := detectSystemd(ancestry); src != nil {
		return *src
	}
	if src := detectLaunchd(ancestry); src != nil {
		return *src
	}
	if src := detectBsdRc(ancestry); src != nil {
		return *src
	}
	if src := detectSupervisor(ancestry); src != nil {
		return *src
	}
	if src := detectCron(ancestry); src != nil {
		return *src
	}
	if src := detectWindowsService(ancestry); src != nil {
		return *src
	}
	if src := detectInit(ancestry); src != nil {
		return *src
	}
	if src := detectShell(ancestry); src != nil {
		return *src
	}

	return model.Source{
		Type: model.SourceUnknown,
	}
}

// env suspicious warnings returns warnings for known env based library injection patterns
func envSuspiciousWarnings(env []string) []string {
	matched := make([]bool, len(envVarRules))
	matchedKeys := make([]map[string]struct{}, len(envVarRules))

	// init per rule key capture only for rules that include keys
	for i, rule := range envVarRules {
		if rule.includeKeys {
			matchedKeys[i] = map[string]struct{}{}
		}
	}

	// scan env entries and record which rules match
	for _, entry := range env {
		key, value, ok := strings.Cut(entry, "=")
		if !ok || value == "" {
			continue
		}

		// check this key against each configured rule
		for i, rule := range envVarRules {
			if !rule.match(key, rule.pattern) {
				continue
			}
			matched[i] = true
			if rule.includeKeys {
				matchedKeys[i][key] = struct{}{}
			}
		}
	}

	var warnings []string

	// emit warnings in the same order as envVarRules
	for i, rule := range envVarRules {
		if !matched[i] {
			continue
		}
		if !rule.includeKeys {
			warnings = append(warnings, rule.warning)
			continue
		}

		keys := make([]string, 0, len(matchedKeys[i]))
		// collect all matched keys for this rule
		for key := range matchedKeys[i] {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		warnings = append(warnings, rule.warning+": "+strings.Join(keys, ", "))
	}

	return warnings
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

	// Include warnings based on suspicious env variables
	w = append(w, envSuspiciousWarnings(last.Env)...)

	return w
}
