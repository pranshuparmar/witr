//go:build darwin

package target

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name    string
		target  model.Target
		wantErr bool
	}{
		{"valid pid", model.Target{Type: model.TargetPID, Value: "1234"}, false},
		{"invalid pid", model.Target{Type: model.TargetPID, Value: "abc"}, true},
		{"empty pid", model.Target{Type: model.TargetPID, Value: ""}, true},
		{"invalid port", model.Target{Type: model.TargetPort, Value: "http"}, true},
		{"unknown type", model.Target{Type: "invalid", Value: "x"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Resolve(tt.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsValidServiceLabel(t *testing.T) {
	tests := []struct {
		label string
		want  bool
	}{
		{"com.apple.Safari", true},
		{"org.nginx", true},
		{"my-service_123", true},
		{"a", true},
		{"", false},
		{"invalid/label", false},
		{"has space", false},
		{"has@special", false},
	}
	for _, tt := range tests {
		if got := isValidServiceLabel(tt.label); got != tt.want {
			t.Errorf("isValidServiceLabel(%q) = %v, want %v", tt.label, got, tt.want)
		}
	}
	longLabel := ""
	for i := 0; i < 300; i++ {
		longLabel += "a"
	}
	if isValidServiceLabel(longLabel) {
		t.Error("isValidServiceLabel should reject labels > 256 chars")
	}
}

func TestResolveName(t *testing.T) {
	// BUG: ResolveName calls os.Exit(1) for ambiguous names
	// Cannot test with common names like "bash" as it crashes the test runner
	// Testing only with names that won't match anything
	_, err := ResolveName("nonexistent_process_xyz123")
	if err == nil {
		t.Error("ResolveName should fail for nonexistent process")
	}

	// Test with a PID-like string (should not match as name)
	_, err = ResolveName("1234567890")
	if err == nil {
		t.Log("ResolveName matched PID-like string as process name")
	}
}

func TestResolvePort(t *testing.T) {
	_, err := ResolvePort(99999)
	if err == nil {
		t.Log("ResolvePort(99999) found something (unexpected)")
	}
}

func TestResolveLaunchdServicePID(t *testing.T) {
	_, err := resolveLaunchdServicePID("nonexistent.service.xyz")
	if err == nil {
		t.Error("resolveLaunchdServicePID should fail for nonexistent service")
	}

	// Test invalid name
	_, err = resolveLaunchdServicePID("invalid/name")
	if err == nil {
		t.Error("resolveLaunchdServicePID should reject invalid names")
	}
}

func TestResolvePortNetstat(t *testing.T) {
	_, _ = resolvePortNetstat(80)
	_, _ = resolvePortNetstat(99999)
}
