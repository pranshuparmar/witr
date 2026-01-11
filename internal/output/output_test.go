package output

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()

	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe failed: %v", err)
	}
	os.Stdout = w

	outCh := make(chan string)
	go func() {
		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		outCh <- buf.String()
	}()

	fn()
	_ = w.Close()
	os.Stdout = orig

	out := <-outCh
	_ = r.Close()
	return out
}

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
	out := captureOutput(t, func() { RenderWarnings(nil, false) })
	if !strings.Contains(out, "No warnings.") {
		t.Fatalf("RenderWarnings(nil,false) = %q", out)
	}

	out = captureOutput(t, func() { RenderWarnings([]string{}, true) })
	if !strings.Contains(out, "No warnings.") {
		t.Fatalf("RenderWarnings(empty,true) = %q", out)
	}

	out = captureOutput(t, func() { RenderWarnings([]string{"warning1"}, false) })
	if !strings.Contains(out, "Warnings:") || !strings.Contains(out, "warning1") {
		t.Fatalf("RenderWarnings(single,false) = %q", out)
	}

	out = captureOutput(t, func() { RenderWarnings([]string{"warning1", "warning2"}, true) })
	if !strings.Contains(out, "Warnings") || !strings.Contains(out, "warning2") {
		t.Fatalf("RenderWarnings(multi,true) = %q", out)
	}
}

func TestRenderEnvOnly(t *testing.T) {
	proc := model.Process{Cmdline: "test --flag", Env: []string{"VAR=val", "PATH=/bin"}}
	out := captureOutput(t, func() { RenderEnvOnly(proc, false) })
	if !strings.Contains(out, "Command     : test --flag") || !strings.Contains(out, "VAR=val") {
		t.Fatalf("RenderEnvOnly(false) = %q", out)
	}

	out = captureOutput(t, func() { RenderEnvOnly(proc, true) })
	if !strings.Contains(out, "Command") || !strings.Contains(out, "PATH=/bin") {
		t.Fatalf("RenderEnvOnly(true) = %q", out)
	}

	out = captureOutput(t, func() { RenderEnvOnly(model.Process{Cmdline: "test"}, true) })
	if !strings.Contains(out, "No environment variables found.") {
		t.Fatalf("RenderEnvOnly(no env) = %q", out)
	}
}

func TestRenderShort(t *testing.T) {
	result := model.Result{Ancestry: []model.Process{
		{PID: 1, Command: "init"},
		{PID: 100, Command: "bash"},
		{PID: 200, Command: "test"},
	}}
	out := captureOutput(t, func() { RenderShort(result, false) })
	if strings.TrimSpace(out) != "init (pid 1) → bash (pid 100) → test (pid 200)" {
		t.Fatalf("RenderShort(false) = %q", out)
	}
	out = captureOutput(t, func() { RenderShort(result, true) })
	if strings.TrimSpace(out) == "" {
		t.Fatal("RenderShort(true) produced empty output")
	}
}

func TestPrintTree(t *testing.T) {
	chain := []model.Process{
		{PID: 1, Command: "init"},
		{PID: 100, Command: "bash"},
		{PID: 200, Command: "test"},
	}
	out := captureOutput(t, func() { PrintTree(chain, false) })
	if !strings.Contains(out, "init (pid 1)") || !strings.Contains(out, "└─ bash (pid 100)") {
		t.Fatalf("PrintTree(false) = %q", out)
	}
	out = captureOutput(t, func() { PrintTree(chain, true) })
	if !strings.Contains(out, "└─") {
		t.Fatalf("PrintTree(true) = %q", out)
	}
	out = captureOutput(t, func() { PrintTree([]model.Process{{PID: 1}}, true) })
	if !strings.Contains(out, "pid 1") {
		t.Fatalf("PrintTree(single) = %q", out)
	}
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
	out := captureOutput(t, func() { RenderStandard(result, false) })
	if !strings.Contains(out, "Target      : test") || !strings.Contains(out, "Process     : test (pid 200)") {
		t.Fatalf("RenderStandard(false) = %q", out)
	}
	if !strings.Contains(out, "Listening   : 0.0.0.0:8080") || !strings.Contains(out, "Warnings    :") {
		t.Fatalf("RenderStandard(false) missing sections: %q", out)
	}

	out = captureOutput(t, func() { RenderStandard(result, true) })
	if !strings.Contains(out, "Process") {
		t.Fatalf("RenderStandard(true) = %q", out)
	}

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
	out = captureOutput(t, func() { RenderStandard(launchdResult, false) })
	if !strings.Contains(out, "Source      : com.test.service (launchd)") {
		t.Fatalf("RenderStandard(launchd) = %q", out)
	}
}

func TestRenderStandardEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("RenderStandard panics with empty Result: %v", r)
		}
	}()
	RenderStandard(model.Result{}, false)
}
