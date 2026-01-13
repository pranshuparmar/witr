package source

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDetectContainer(t *testing.T) {
	// Setup temporary proc directory
	tempDir := t.TempDir()
	originalProcPath := procPath
	procPath = tempDir
	defer func() { procPath = originalProcPath }()

	tests := []struct {
		name          string
		cgroupContent string
		expectedType  model.SourceType
		expectedName  string
		expectNil     bool
	}{
		{
			name:          "Docker Container",
			cgroupContent: "11:devices:/docker/1234567890abcdef",
			expectedType:  model.SourceContainer,
			expectedName:  "docker",
		},
		{
			name:          "Podman Container",
			cgroupContent: "1:name=systemd:/user.slice/user-1000.slice/user@1000.service/user.slice/libpod-123.scope",
			expectedType:  model.SourceContainer,
			expectedName:  "podman",
		},
		{
			name:          "Kubernetes Pod",
			cgroupContent: "11:memory:/kubepods/burstable/pod123-456/123",
			expectedType:  model.SourceContainer,
			expectedName:  "kubernetes",
		},
		{
			name:          "Colima",
			cgroupContent: "1:name=systemd:/colima",
			expectedType:  model.SourceContainer,
			expectedName:  "colima",
		},
		{
			name:          "Containerd",
			cgroupContent: "1:name=systemd:/containerd/123",
			expectedType:  model.SourceContainer,
			expectedName:  "containerd",
		},
		{
			name:          "Not a Container",
			cgroupContent: "1:name=systemd:/user.slice/user-1000.slice/session-1.scope",
			expectNil:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create dummy process and cgroup file
			pid := 123
			cgroupDir := filepath.Join(tempDir, strconv.Itoa(pid))
			err := os.MkdirAll(cgroupDir, 0755)
			if err != nil {
				t.Fatalf("Failed to create cgroup dir: %v", err)
			}
			cgroupFile := filepath.Join(cgroupDir, "cgroup")
			err = os.WriteFile(cgroupFile, []byte(tt.cgroupContent), 0644)
			if err != nil {
				t.Fatalf("Failed to write cgroup file: %v", err)
			}

			ancestry := []model.Process{{PID: pid}}
			source := detectContainer(ancestry)

			if tt.expectNil {
				if source != nil {
					t.Errorf("Expected nil source, got %v", source)
				}
			} else {
				if source == nil {
					t.Fatalf("Expected source, got nil")
				}
				if source.Type != tt.expectedType {
					t.Errorf("Type = %v, want %v", source.Type, tt.expectedType)
				}
				if source.Name != tt.expectedName {
					t.Errorf("Name = %q, want %q", source.Name, tt.expectedName)
				}
			}
		})
	}
}

func TestDetectContainerMissingFile(t *testing.T) {
	// Setup temporary proc directory
	tempDir := t.TempDir()
	originalProcPath := procPath
	procPath = tempDir
	defer func() { procPath = originalProcPath }()

	// PID dir exists but cgroup file does not
	pid := 456
	cgroupDir := filepath.Join(tempDir, strconv.Itoa(pid))
	os.MkdirAll(cgroupDir, 0755)

	ancestry := []model.Process{{PID: pid}}
	source := detectContainer(ancestry)

	if source != nil {
		t.Errorf("Expected nil source for missing file, got %v", source)
	}
}
