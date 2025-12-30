package source

import (
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetect(t *testing.T) {
	tests := []struct {
		name     string
		ancestry []model.Process
		wantType model.SourceType
		wantName string
	}{
		{"shell - bash", []model.Process{{PID: 0, Command: "bash"}, {PID: 0, Command: "myapp"}}, model.SourceShell, "bash"},
		{"supervisor - pm2", []model.Process{{PID: 0, Command: "pm2"}, {PID: 0, Command: "node"}}, model.SourceSupervisor, "pm2"},
		{"cron", []model.Process{{PID: 0, Command: "cron"}, {PID: 0, Command: "backup.sh"}}, model.SourceCron, "cron"},
		{"unknown - empty", []model.Process{}, model.SourceUnknown, ""},
		{"unknown - no match", []model.Process{{PID: 0, Command: "unknown_process"}}, model.SourceUnknown, ""},
		{"supervisor priority", []model.Process{{PID: 0, Command: "bash"}, {PID: 0, Command: "supervisord"}, {PID: 0, Command: "myapp"}}, model.SourceSupervisor, "supervisord"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Detect(tt.ancestry)
			if got.Type != tt.wantType {
				t.Errorf("Detect() type = %v, want %v", got.Type, tt.wantType)
			}
			if got.Name != tt.wantName {
				t.Errorf("Detect() name = %v, want %v", got.Name, tt.wantName)
			}
		})
	}
}

func TestIsPublicBind(t *testing.T) {
	tests := []struct {
		addrs []string
		want  bool
	}{
		{[]string{"0.0.0.0"}, true},
		{[]string{"::"}, true},
		{[]string{"127.0.0.1"}, false},
		{[]string{"127.0.0.1", "0.0.0.0"}, true},
		{[]string{}, false},
		{nil, false},
	}
	for _, tt := range tests {
		if got := IsPublicBind(tt.addrs); got != tt.want {
			t.Errorf("IsPublicBind(%v) = %v, want %v", tt.addrs, got, tt.want)
		}
	}
}

func TestWarnings(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name      string
		processes []model.Process
		contains  string
	}{
		{"zombie", []model.Process{{Command: "bash"}, {Command: "myapp", Health: "zombie"}}, "zombie"},
		{"root", []model.Process{{Command: "bash"}, {Command: "myapp", User: "root"}}, "root"},
		{"public", []model.Process{{Command: "bash"}, {Command: "myapp", BindAddresses: []string{"0.0.0.0"}}}, "public"},
		{"tmp", []model.Process{{Command: "bash"}, {Command: "myapp", WorkingDir: "/tmp"}}, "suspicious"},
		{"stopped", []model.Process{{Command: "bash"}, {Command: "myapp", Health: "stopped"}}, "stopped"},
		{"high-cpu", []model.Process{{Command: "bash"}, {Command: "myapp", Health: "high-cpu"}}, "CPU"},
		{"high-mem", []model.Process{{Command: "bash"}, {Command: "myapp", Health: "high-mem"}}, "memory"},
		{"container", []model.Process{{Command: "bash"}, {Command: "myapp", Container: "abc123"}}, "healthcheck"},
		{"mismatch", []model.Process{{Command: "bash"}, {Command: "myapp", Service: "other"}}, "match"},
		{"old", []model.Process{{Command: "bash"}, {Command: "myapp", StartedAt: now.Add(-100 * 24 * time.Hour)}}, "90"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := Warnings(tt.processes)
			found := false
			for _, w := range warnings {
				if strings.Contains(strings.ToLower(w), strings.ToLower(tt.contains)) {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Warnings() missing %q, got %v", tt.contains, warnings)
			}
		})
	}
}

func TestWarningsRestartCount(t *testing.T) {
	procs := []model.Process{
		{Command: "app"}, {Command: "app"}, {Command: "app"},
		{Command: "app"}, {Command: "app"}, {Command: "app"}, {Command: "app"},
	}
	warnings := Warnings(procs)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "restart") {
			found = true
		}
	}
	if !found {
		t.Error("Expected restart warning for repeated commands")
	}
}
