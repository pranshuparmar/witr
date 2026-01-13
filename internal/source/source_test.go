package source

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetectPrimary(t *testing.T) {
	tests := []struct {
		name  string
		chain []model.Process
		want  string
	}{
		{"systemd", []model.Process{{Command: "systemd"}, {Command: "nginx"}}, "supervisor"},
		{"dockerd", []model.Process{{Command: "dockerd"}, {Command: "myapp"}}, "unknown"},
		{"containerd", []model.Process{{Command: "containerd"}, {Command: "myapp"}}, "unknown"},
		{"pm2", []model.Process{{Command: "pm2"}, {Command: "node"}}, "supervisor"},
		{"cron", []model.Process{{Command: "cron"}, {Command: "job"}}, "cron"},
		{"manual", []model.Process{{Command: "bash"}, {Command: "myapp"}}, "shell"},
		{"empty", []model.Process{}, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := string(Detect(tt.chain).Type); got != tt.want {
				t.Errorf("Detect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	if got := itoa(123); got != "123" {
		t.Errorf("itoa(123) = %v, want 123", got)
	}
}
