package source

import (
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestWarningsDetectsLDPreload(t *testing.T) {
	p := []model.Process{
		{PID: 999999, Command: "pm2", Cmdline: "pm2"},
		{
			PID:        123,
			Command:    "bash",
			StartedAt:  time.Now(),
			User:       "bob",
			WorkingDir: "/home/bob",
			Env:        []string{"LD_PRELOAD=/tmp/libhack.so"},
		},
	}

	warnings := Warnings(p)
	if !slices.Contains(warnings, "Process sets LD_PRELOAD (potential library injection)") {
		t.Fatalf("expected LD_PRELOAD warning, got: %v", warnings)
	}
}

func TestWarningsDetectsDYLDVars(t *testing.T) {
	p := []model.Process{
		{PID: 999999, Command: "pm2", Cmdline: "pm2"},
		{
			PID:        123,
			Command:    "zsh",
			StartedAt:  time.Now(),
			User:       "bob",
			WorkingDir: "/home/bob",
			Env: []string{
				"DYLD_LIBRARY_PATH=/tmp",
				"DYLD_INSERT_LIBRARIES=/tmp/inject.dylib",
			},
		},
	}

	warnings := Warnings(p)
	want := "Process sets DYLD_* variables (potential library injection): DYLD_INSERT_LIBRARIES, DYLD_LIBRARY_PATH"
	if !slices.Contains(warnings, want) {
		t.Fatalf("expected DYLD warning %q, got: %v", want, warnings)
	}
}

func TestWarningsIgnoresEmptyPreloadVars(t *testing.T) {
	p := []model.Process{
		{PID: 999999, Command: "pm2", Cmdline: "pm2"},
		{
			PID:        123,
			Command:    "zsh",
			StartedAt:  time.Now(),
			User:       "bob",
			WorkingDir: "/home/bob",
			Env: []string{
				"LD_PRELOAD=",
				"DYLD_INSERT_LIBRARIES=",
			},
		},
	}

	warnings := Warnings(p)
	if slices.Contains(warnings, "Process sets LD_PRELOAD (potential library injection)") {
		t.Fatalf("did not expect LD_PRELOAD warning, got: %v", warnings)
	}
	if slices.Contains(warnings, "Process sets DYLD_* variables (potential library injection): DYLD_INSERT_LIBRARIES") {
		t.Fatalf("did not expect DYLD warning, got: %v", warnings)
	}
}

// checks if the order of env vars warnings are deterministic
func FuzzEnvSuspiciousWarningsDeterministic(f *testing.F) {
	f.Add("LD_PRELOAD=/tmp/lib.so")
	f.Add("DYLD_LIBRARY_PATH=/tmp\nDYLD_INSERT_LIBRARIES=/tmp/inject.dylib")
	f.Add("DYLD_LIBRARY_PATH=\nLD_PRELOAD=")
	f.Add("")

	f.Fuzz(func(t *testing.T, input string) {
		parts := strings.Split(input, "\n")
		if len(parts) > 50 {
			parts = parts[:50]
		}
		for i := range parts {
			if len(parts[i]) > 200 {
				parts[i] = parts[i][:200]
			}
		}

		w1 := envSuspiciousWarnings(parts)
		w2 := envSuspiciousWarnings(parts)
		if !slices.Equal(w1, w2) {
			t.Fatalf("expected deterministic output, got %v vs %v", w1, w2)
		}
	})
}

func TestEnvSuspiciousWarnings(t *testing.T) {
	tests := []struct {
		name string
		env  []string
		want []string
	}{
		{
			name: "LD_PRELOAD",
			env:  []string{"LD_PRELOAD=/tmp/libhack.so"},
			want: []string{"Process sets LD_PRELOAD (potential library injection)"},
		},
		{
			name: "DYLD keys sorted and deduped",
			env: []string{
				"DYLD_LIBRARY_PATH=/tmp",
				"DYLD_INSERT_LIBRARIES=/tmp/inject.dylib",
				"DYLD_LIBRARY_PATH=/tmp", // dup
			},
			want: []string{
				"Process sets DYLD_* variables (potential library injection): DYLD_INSERT_LIBRARIES, DYLD_LIBRARY_PATH",
			},
		},
		{
			name: "ignores empty values (current behavior)",
			env:  []string{"LD_PRELOAD=", "DYLD_INSERT_LIBRARIES="},
			want: nil,
		},
		{
			name: "value with '=' still counts",
			env:  []string{"LD_PRELOAD=a=b"},
			want: []string{"Process sets LD_PRELOAD (potential library injection)"},
		},
		{
			name: "no '=' ignored",
			env:  []string{"LD_PRELOAD"},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envSuspiciousWarnings(tt.env)
			if !slices.Equal(got, tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

// attempts to cause a panic when checking for env vars
func FuzzWarningsNoPanic(f *testing.F) {
	f.Add("LD_PRELOAD=/tmp/lib.so")
	f.Add("DYLD_INSERT_LIBRARIES=/tmp/inject.dylib")
	f.Add("NOT_AN_ENV")

	f.Fuzz(func(t *testing.T, entry string) {
		if len(entry) > 2000 {
			entry = entry[:2000]
		}
		p := []model.Process{
			{
				PID:        123,
				Command:    "test",
				Cmdline:    "test",
				StartedAt:  time.Now(),
				User:       "bob",
				WorkingDir: "/home/bob",
				Env:        []string{entry},
			},
		}

		_ = Warnings(p)
	})
}

func TestWarningsDetectsDeletedExecutable(t *testing.T) {
	p := []model.Process{
		{
			PID:        123,
			Command:    "nginx",
			StartedAt:  time.Now(),
			ExeDeleted: true,
		},
	}

	warnings := Warnings(p)
	want := "Process is running from a deleted binary (potential library injection or pending update)"
	if !slices.Contains(warnings, want) {
		t.Fatalf("expected deleted binary warning, got: %v", warnings)
	}
}

func TestEnrichSocketInfo(t *testing.T) {
	tests := []struct {
		state           string
		wantExplanation string
	}{
		{"TIME_WAIT", "The local OS is holding the port in a protocol-wait state to ensure all packets are received."},
		{"CLOSE_WAIT", "The remote end has closed the connection, but the local application hasn't responded."},
		{"LISTEN", "The process is actively waiting for incoming connections."},
	}

	for _, tt := range tests {
		si := &model.SocketInfo{State: tt.state}
		EnrichSocketInfo(si)
		if si.Explanation != tt.wantExplanation {
			t.Errorf("state %s: got explanation %q, want %q", tt.state, si.Explanation, tt.wantExplanation)
		}
	}
}
