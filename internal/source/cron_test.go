package source

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetectCron(t *testing.T) {
	tests := []struct {
		name     string
		ancestry []model.Process
		wantNil  bool
	}{
		{"cron", []model.Process{{Command: "cron"}, {Command: "job"}}, false},
		{"crond", []model.Process{{Command: "crond"}, {Command: "job"}}, false},
		{"no cron", []model.Process{{Command: "bash"}}, true},
		{"empty", []model.Process{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectCron(tt.ancestry)
			if tt.wantNil && got != nil {
				t.Errorf("detectCron() = %v, want nil", got)
			}
			if !tt.wantNil {
				if got == nil {
					t.Fatal("detectCron() = nil, want non-nil")
				}
				if got.Type != model.SourceCron || got.Name != "cron" {
					t.Errorf("detectCron() = %v, want Type=cron Name=cron", got)
				}
			}
		})
	}
}
