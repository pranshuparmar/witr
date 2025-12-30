//go:build darwin

package source

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetectLaunchd(t *testing.T) {
	tests := []struct {
		name     string
		ancestry []model.Process
		wantNil  bool
	}{
		{"no launchd", []model.Process{{PID: 100, Command: "bash"}}, true},
		{"has launchd", []model.Process{{PID: 1, Command: "launchd"}, {PID: 100, Command: "myapp"}}, false},
		{"empty", []model.Process{}, true},
		{"launchd only", []model.Process{{PID: 1, Command: "launchd"}}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectLaunchd(tt.ancestry)
			if tt.wantNil && got != nil {
				t.Errorf("detectLaunchd() = %v, want nil", got)
			}
			if !tt.wantNil && got == nil {
				t.Error("detectLaunchd() = nil, want non-nil")
			}
			if got != nil && got.Type != model.SourceLaunchd {
				t.Errorf("detectLaunchd() type = %v, want %v", got.Type, model.SourceLaunchd)
			}
		})
	}
}

func TestDetectSystemd(t *testing.T) {
	got := detectSystemd([]model.Process{{Command: "systemd"}})
	if got != nil {
		t.Error("detectSystemd should always return nil on Darwin")
	}
}

func TestDetectContainerSource(t *testing.T) {
	got := detectContainer([]model.Process{{PID: 1, Command: "bash"}})
	_ = got
}
