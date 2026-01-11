package source

import (
	"slices"
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

func TestWarningsEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Warnings panicked on empty input: %v", r)
		}
	}()

	warnings := Warnings([]model.Process{})
	if len(warnings) != 0 {
		t.Fatalf("Warnings(empty) = %v, want empty", warnings)
	}
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
