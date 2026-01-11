package source

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name     string
		ancestry []model.Process
		wantName string
		wantNil  bool
	}{
		{"bash", []model.Process{{Command: "bash"}, {Command: "myapp"}}, "bash", false},
		{"zsh", []model.Process{{Command: "zsh"}, {Command: "myapp"}}, "zsh", false},
		{"sh", []model.Process{{Command: "sh"}, {Command: "myapp"}}, "sh", false},
		{"fish", []model.Process{{Command: "fish"}, {Command: "myapp"}}, "fish", false},
		{"no shell", []model.Process{{Command: "init"}, {Command: "myapp"}}, "", true},
		{"empty", []model.Process{}, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := detectShell(tt.ancestry)
			if tt.wantNil && got != nil {
				t.Errorf("detectShell() = %v, want nil", got)
			}
			if !tt.wantNil {
				if got == nil {
					t.Fatalf("detectShell() = nil, want %v", tt.wantName)
				}
				if got.Type != model.SourceShell || got.Name != tt.wantName {
					t.Errorf("detectShell() = %v, want Type=shell Name=%v", got, tt.wantName)
				}
			}
		})
	}
}
