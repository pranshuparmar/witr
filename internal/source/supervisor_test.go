package source

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetectSupervisor(t *testing.T) {
	tests := []struct {
		name     string
		ancestry []model.Process
		wantName string
		wantNil  bool
	}{
		{"pm2 command", []model.Process{{Command: "pm2"}, {Command: "node"}}, "pm2", false},
		{"pm2 cmdline", []model.Process{{Cmdline: "/usr/bin/pm2 start"}}, "pm2", false},
		{"supervisord", []model.Process{{Command: "supervisord"}}, "supervisord", false},
		{"gunicorn", []model.Process{{Command: "gunicorn"}}, "gunicorn", false},
		{"uwsgi", []model.Process{{Command: "uwsgi"}}, "uwsgi", false},
		{"runsv", []model.Process{{Command: "runsv"}}, "runit", false},
		{"s6-supervise", []model.Process{{Command: "s6-supervise"}}, "s6", false},
		{"monit", []model.Process{{Command: "monit"}}, "monit", false},
		{"circusd", []model.Process{{Command: "circusd"}}, "circus", false},
		{"tini", []model.Process{{Command: "tini"}}, "tini", false},
		{"no supervisor", []model.Process{{Command: "bash"}}, "", true},
		{"empty", []model.Process{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectSupervisor(tt.ancestry)
			if tt.wantNil && got != nil {
				t.Errorf("detectSupervisor() = %v, want nil", got)
			}
			if !tt.wantNil {
				if got == nil {
					t.Fatalf("detectSupervisor() = nil, want %v", tt.wantName)
				}
				if got.Type != model.SourceSupervisor || got.Name != tt.wantName {
					t.Errorf("detectSupervisor() = %v, want Type=supervisor Name=%v", got, tt.wantName)
				}
			}
		})
	}
}
