//go:build linux || darwin || freebsd || windows

package target

import (
	"slices"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestResolveWithExactFlag(t *testing.T) {
	tests := []struct {
		name  string
		exact bool
	}{
		{
			name:  "fuzzy matching",
			exact: false,
		},
		{
			name:  "exact matching",
			exact: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := model.Target{
				Type:  model.TargetName,
				Value: "test",
			}

			_, err := Resolve(target, tt.exact)

			if err == nil {
				t.Log("Resolve returned successfully (may have found matches)")
			} else {
				t.Logf("Resolve returned error: %v", err)
			}
		})
	}
}

func TestResolvePIDWithExactFlag(t *testing.T) {
	tests := []struct {
		name  string
		exact bool
		pid   string
	}{
		{"exact flag true", true, "1"},
		{"exact flag false", false, "1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := model.Target{
				Type:  model.TargetPID,
				Value: tt.pid,
			}

			pids, err := Resolve(target, tt.exact)
			if err != nil {
				t.Fatalf("Resolve failed: %v", err)
			}

			if len(pids) != 1 {
				t.Fatalf("expected 1 PID, got %d", len(pids))
			}

			if pids[0] != 1 {
				t.Fatalf("expected PID 1, got %d", pids[0])
			}
		})
	}
}

func TestMatchesExactToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cmdline string
		needle  string
		want    bool
	}{
		{
			name:    "matches a path segment in the executable arg",
			cmdline: "/usr/bin/nginx -g daemon off",
			needle:  "nginx",
			want:    true,
		},
		{
			name:    "matches path component (snap pattern)",
			cmdline: "/snap/core24/1349/bin/python",
			needle:  "core24",
			want:    true,
		},
		{
			name:    "does not match substring of a token",
			cmdline: "/usr/local/bin/foo-bar",
			needle:  "foo",
			want:    false,
		},
		{
			name:    "matches a bare argument",
			cmdline: "node server.js production",
			needle:  "node",
			want:    true,
		},
		{
			name:    "matches across backslash-separated paths (Windows-style)",
			cmdline: `C:\Program Files\App\bin.exe`,
			needle:  "App",
			want:    true,
		},
		{
			name:    "does not match partial filename",
			cmdline: "/usr/bin/python3",
			needle:  "python",
			want:    false,
		},
		{
			name:    "matches file basename with extension",
			cmdline: `C:\Program Files\App\witr.exe`,
			needle:  "witr.exe",
			want:    true,
		},
		{
			name:    "empty cmdline never matches",
			cmdline: "",
			needle:  "nginx",
			want:    false,
		},
		{
			name:    "empty name never matches an absolute path",
			cmdline: "/usr/bin/foo",
			needle:  "",
			want:    false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := matchesExactToken(tt.cmdline, tt.needle); got != tt.want {
				t.Errorf("matchesExactToken(%q, %q) = %v, want %v", tt.cmdline, tt.needle, got, tt.want)
			}
		})
	}
}

func TestResolvePortWithExactFlag(t *testing.T) {
	tests := []struct {
		name  string
		exact bool
		port  string
	}{
		{"exact flag true", true, "22"},
		{"exact flag false", false, "22"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			target := model.Target{
				Type:  model.TargetPort,
				Value: tt.port,
			}

			_, err := Resolve(target, tt.exact)

			if err == nil {
				t.Log("Resolve returned successfully (may have found matches)")
			} else {
				t.Logf("Resolve returned error: %v", err)
			}
		})
	}
}

func TestAddressOwnerPIDs(t *testing.T) {
	tests := []struct {
		name string
		in   map[string][]int
		want []int
	}{
		{"one process on IPv4 and IPv6", map[string][]int{"127.0.0.1:8080": {100}, "[::1]:8080": {100}}, []int{100}},
		{"master and workers share the listener", map[string][]int{"*:80": {102, 100, 101}}, []int{100}},
		{"master on both families, workers on one", map[string][]int{"*:80": {101, 100}, "*6:80": {100}}, []int{100}},
		{"two services", map[string][]int{"127.0.0.1:53": {300}, "10.0.0.1:53": {200}}, []int{200, 300}},
		{"nothing", map[string][]int{"*:80": {}}, nil},
	}
	for _, tt := range tests {
		if got := addressOwnerPIDs(tt.in); !slices.Equal(got, tt.want) {
			t.Errorf("%s: addressOwnerPIDs = %v, want %v", tt.name, got, tt.want)
		}
	}
}
