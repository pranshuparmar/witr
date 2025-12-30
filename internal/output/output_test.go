package output

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestToJSON(t *testing.T) {
	tests := []struct {
		name   string
		result model.Result
	}{
		{"basic", model.Result{Process: model.Process{PID: 123, Command: "nginx"}}},
		{"with ancestry", model.Result{Ancestry: []model.Process{{PID: 1}, {PID: 100}}}},
		{"with warnings", model.Result{Warnings: []string{"warning1", "warning2"}}},
		{"with source", model.Result{Source: model.Source{Type: model.SourceShell, Name: "bash"}}},
		{"full", model.Result{
			Process:  model.Process{PID: 123, Command: "app", User: "root", StartedAt: time.Now()},
			Ancestry: []model.Process{{PID: 1, Command: "init"}},
			Source:   model.Source{Type: model.SourceCron, Name: "cron"},
			Warnings: []string{"warning"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ToJSON(tt.result)
			if err != nil {
				t.Errorf("ToJSON() error = %v", err)
				return
			}
			var parsed map[string]interface{}
			if err := json.Unmarshal([]byte(got), &parsed); err != nil {
				t.Errorf("ToJSON() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestFormatDetailLabel(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"type", "              Type"},
		{"plist", "              Plist"},
		{"triggers", "              Trigger"},
		{"keepalive", "              KeepAlive"},
		{"custom", "              custom"},
	}
	for _, tt := range tests {
		got := formatDetailLabel(tt.key)
		if got != tt.want {
			t.Errorf("formatDetailLabel(%q) = %q, want %q", tt.key, got, tt.want)
		}
	}
}

func TestRenderWarnings(t *testing.T) {
	RenderWarnings(nil, false)
	RenderWarnings([]string{}, true)
	RenderWarnings([]string{"warning1"}, false)
	RenderWarnings([]string{"warning1", "warning2"}, true)
}

func TestRenderEnvOnly(t *testing.T) {
	proc := model.Process{Cmdline: "test --flag", Env: []string{"VAR=val", "PATH=/bin"}}
	RenderEnvOnly(proc, false)
	RenderEnvOnly(proc, true)
	RenderEnvOnly(model.Process{Cmdline: "test"}, true)
}

func TestRenderShort(t *testing.T) {
	result := model.Result{Ancestry: []model.Process{
		{PID: 1, Command: "init"},
		{PID: 100, Command: "bash"},
		{PID: 200, Command: "test"},
	}}
	RenderShort(result, false)
	RenderShort(result, true)
}

func TestPrintTree(t *testing.T) {
	chain := []model.Process{
		{PID: 1, Command: "init"},
		{PID: 100, Command: "bash"},
		{PID: 200, Command: "test"},
	}
	PrintTree(chain, false)
	PrintTree(chain, true)
	PrintTree([]model.Process{{PID: 1}}, true)
}

func TestRenderStandard(t *testing.T) {
	result := model.Result{
		Ancestry: []model.Process{
			{PID: 1, Command: "init"},
			{PID: 100, Command: "bash", User: "root", WorkingDir: "/tmp"},
			{PID: 200, Command: "test", ListeningPorts: []int{8080}, BindAddresses: []string{"0.0.0.0"}},
		},
		Source: model.Source{
			Type:    model.SourceShell,
			Name:    "bash",
			Details: map[string]string{"type": "shell", "plist": "/path"},
		},
		Warnings: []string{"warning1", "warning2"},
	}
	RenderStandard(result, false)
	RenderStandard(result, true)

	launchdResult := model.Result{
		Ancestry: []model.Process{{PID: 1, Command: "launchd"}, {PID: 100}},
		Source: model.Source{
			Type: model.SourceLaunchd,
			Name: "com.test.service",
			Details: map[string]string{
				"type":      "Launch Daemon",
				"plist":     "/Library/LaunchDaemons/com.test.plist",
				"triggers":  "RunAtLoad",
				"keepalive": "Yes",
			},
		},
	}
	RenderStandard(launchdResult, false)
}

func TestRenderStandardEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RenderStandard panics with empty Result: %v", r)
		}
	}()
	RenderStandard(model.Result{}, false)
}
