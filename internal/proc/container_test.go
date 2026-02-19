package proc

import (
	"testing"
)

func TestSplitCmdline(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"simple", "docker ps", []string{"docker", "ps"}},
		{"quoted", `docker inspect --format "{{.Name}}"`, []string{"docker", "inspect", "--format", "{{.Name}}"}},
		{"empty", "", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitCmdline(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("splitCmdline(%q) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("splitCmdline(%q)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestFindLongHexID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"found", "/docker/" + "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2" + "/cgroup", "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"},
		{"not found", "no hex here", ""},
		{"too short", "a1b2c3d4e5f6", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findLongHexID(tt.in)
			if got != tt.want {
				t.Fatalf("findLongHexID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractFlagValue(t *testing.T) {
	tests := []struct {
		name    string
		cmdline string
		flags   []string
		want    string
	}{
		{"found", "docker run --name myapp", []string{"--name"}, "myapp"},
		{"not found", "docker run myapp", []string{"--name"}, ""},
		{"at end", "docker run --name", []string{"--name"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFlagValue(tt.cmdline, tt.flags...)
			if got != tt.want {
				t.Fatalf("extractFlagValue() = %q, want %q", got, tt.want)
			}
		})
	}
}
