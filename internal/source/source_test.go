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
		{"systemd", []model.Process{{Command: "systemd"}, {Command: "nginx"}}, "systemd"},
		{"dockerd", []model.Process{{Command: "dockerd"}, {Command: "myapp"}}, "docker"},
		{"containerd", []model.Process{{Command: "containerd"}, {Command: "myapp"}}, "docker"},
		{"pm2", []model.Process{{Command: "pm2"}, {Command: "node"}}, "pm2"},
		{"cron", []model.Process{{Command: "cron"}, {Command: "job"}}, "cron"},
		{"manual", []model.Process{{Command: "bash"}, {Command: "myapp"}}, "manual"},
		{"empty", []model.Process{}, "manual"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectPrimary(tt.chain); got != tt.want {
				t.Errorf("DetectPrimary() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestItoa(t *testing.T) {
	if got := itoa(123); got != "123" {
		t.Errorf("itoa(123) = %v, want 123", got)
	}
}
