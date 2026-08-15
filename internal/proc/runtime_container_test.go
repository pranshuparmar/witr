package proc

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name string
		p    appleProcessConfig
		want string
	}{
		{"empty", appleProcessConfig{}, ""},
		{"executable only", appleProcessConfig{Executable: "/bin/sh"}, "/bin/sh"},
		{"executable with args", appleProcessConfig{
			Executable: "/bin/sh",
			Arguments:  []string{"-c", "echo hello"},
		}, "/bin/sh -c echo hello"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildCommand(tt.p); got != tt.want {
				t.Errorf("buildCommand() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildNetworks(t *testing.T) {
	tests := []struct {
		name string
		nets []appleNetwork
		want string
	}{
		{"empty", nil, ""},
		{"single", []appleNetwork{
			{Network: "bridge", IPv4Address: "172.17.0.2"},
		}, "bridge:172.17.0.2"},
		{"multiple", []appleNetwork{
			{Network: "bridge", IPv4Address: "172.17.0.2"},
			{Network: "host", IPv4Address: "192.168.1.10"},
		}, "bridge:172.17.0.2, host:192.168.1.10"},
		{"no ip", []appleNetwork{
			{Network: "bridge"},
		}, ""},
		{"mixed", []appleNetwork{
			{Network: "bridge", IPv4Address: "172.17.0.2"},
			{Network: "none"},
		}, "bridge:172.17.0.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildNetworks(tt.nets); got != tt.want {
				t.Errorf("buildNetworks() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildMounts(t *testing.T) {
	tests := []struct {
		name   string
		mounts []appleMount
		want   string
	}{
		{"empty", nil, ""},
		{"single", []appleMount{
			{HostPath: "/host/path", ContainerPath: "/container/path"},
		}, "/host/path:/container/path"},
		{"multiple", []appleMount{
			{HostPath: "/host/a", ContainerPath: "/container/a"},
			{HostPath: "/host/b", ContainerPath: "/container/b"},
		}, "/host/a:/container/a, /host/b:/container/b"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildMounts(tt.mounts); got != tt.want {
				t.Errorf("buildMounts() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatPort(t *testing.T) {
	tests := []struct {
		name string
		p    applePublishedPort
		want string
	}{
		{"tcp", applePublishedPort{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"}, "8080->80/tcp"},
		{"udp", applePublishedPort{HostPort: 5353, ContainerPort: 53, Protocol: "udp"}, "5353->53/udp"},
		{"default protocol", applePublishedPort{HostPort: 8080, ContainerPort: 80}, "8080->80/tcp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := formatPort(tt.p); got != tt.want {
				t.Errorf("formatPort() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildPorts(t *testing.T) {
	tests := []struct {
		name  string
		ports []applePublishedPort
		want  string
	}{
		{"empty", nil, ""},
		{"single", []applePublishedPort{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
		}, "8080->80/tcp"},
		{"multiple", []applePublishedPort{
			{HostPort: 8080, ContainerPort: 80, Protocol: "tcp"},
			{HostPort: 8443, ContainerPort: 443, Protocol: "tcp"},
		}, "8080->80/tcp, 8443->443/tcp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildPorts(tt.ports); got != tt.want {
				t.Errorf("buildPorts() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseRFC3339(t *testing.T) {
	if !parseRFC3339("").IsZero() {
		t.Error("empty input should yield the zero time")
	}
	if !parseRFC3339("garbage").IsZero() {
		t.Error("garbage input should yield the zero time")
	}
	got := parseRFC3339("2024-06-15T10:30:00Z")
	if got.IsZero() || got.Year() != 2024 || got.Month() != time.June || got.Day() != 15 {
		t.Errorf("parseRFC3339(RFC3339) = %v, want 2024-06-15T10:30:00Z", got)
	}
	// Nano precision
	gotNano := parseRFC3339("2024-06-15T10:30:00.123456789Z")
	if gotNano.IsZero() || gotNano.Nanosecond() != 123456789 {
		t.Errorf("parseRFC3339(nano) nanosecond = %d, want 123456789", gotNano.Nanosecond())
	}
}

func TestAppleContainerRuntimeName(t *testing.T) {
	r := appleContainerRuntime{}
	if r.Name() != "container" {
		t.Errorf("Name() = %q, want %q", r.Name(), "container")
	}
}

func TestAppleContainerHostPID(t *testing.T) {
	r := appleContainerRuntime{}
	if r.HostPID("any-id") != 0 {
		t.Error("HostPID should always return 0 for Apple container runtime")
	}
}

func TestAppleContainerEnrichNil(t *testing.T) {
	// Enrich with nil or empty ID should not panic.
	r := appleContainerRuntime{}
	r.Enrich(nil)
	r.Enrich(&model.ContainerMatch{ID: ""})
}

func TestParseAppleContainerInspectJSON(t *testing.T) {
	sample := `[{"status":{"startedDate":"2024-06-15T10:30:00Z"}}]`
	entries, err := parseAppleContainerEntries([]byte(sample))
	if err != nil {
		t.Fatalf("failed to unmarshal inspect JSON: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Status.StartedDate != "2024-06-15T10:30:00Z" {
		t.Errorf("StartedDate = %q, want 2024-06-15T10:30:00Z", entries[0].Status.StartedDate)
	}
}

func TestAppleContainerListJSONParsing(t *testing.T) {
	// Validate the JSON schema we expect from `container ls --format json --all`.
	sample := `[{
		"configuration": {
			"id": "abc123",
			"image": {"reference": "docker.io/library/nginx:latest"},
			"platform": {"os": "linux", "architecture": "aarch64"},
			"initProcess": {"executable": "/usr/sbin/nginx", "arguments": ["-g", "daemon off;"]},
			"resources": {"cpus": 2, "memoryInBytes": 1073741824},
			"mounts": [{"source": "/tmp/host", "destination": "/data"}],
			"publishedPorts": [{"hostPort": 8080, "containerPort": 80, "proto": "tcp"}],
			"networks": [{"network": "bridge"}],
			"creationDate": "2024-06-15T10:00:00Z"
		},
		"status": {
			"state": "running",
			"networks": [{"network": "bridge", "hostname": "myhost", "ipv4Address": "172.17.0.2", "ipv4Gateway": "172.17.0.1"}],
			"startedDate": "2024-06-15T10:30:00Z"
		}
	}]`
	var entries []appleContainerEntry
	if err := json.Unmarshal([]byte(sample), &entries); err != nil {
		t.Fatalf("failed to unmarshal sample JSON: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.Configuration.ID != "abc123" {
		t.Errorf("ID = %q, want abc123", e.Configuration.ID)
	}
	if e.Configuration.Image.Reference != "docker.io/library/nginx:latest" {
		t.Errorf("Image = %q, want docker.io/library/nginx:latest", e.Configuration.Image.Reference)
	}
	if e.Configuration.Platform.OS != "linux" || e.Configuration.Platform.Architecture != "aarch64" {
		t.Errorf("Platform = %+v", e.Configuration.Platform)
	}
	if e.Configuration.InitProcess.Executable != "/usr/sbin/nginx" {
		t.Errorf("InitProcess.Executable = %q", e.Configuration.InitProcess.Executable)
	}
	if len(e.Configuration.InitProcess.Arguments) != 2 {
		t.Errorf("InitProcess.Arguments len = %d, want 2", len(e.Configuration.InitProcess.Arguments))
	}
	if e.Configuration.Resources.CPUs != 2 {
		t.Errorf("Resources.CPUs = %d, want 2", e.Configuration.Resources.CPUs)
	}
	if e.Configuration.Resources.MemoryInBytes != 1073741824 {
		t.Errorf("Resources.MemoryInBytes = %d, want 1073741824", e.Configuration.Resources.MemoryInBytes)
	}
	if len(e.Configuration.Mounts) != 1 || e.Configuration.Mounts[0].HostPath != "/tmp/host" || e.Configuration.Mounts[0].ContainerPath != "/data" {
		t.Errorf("Mounts = %+v", e.Configuration.Mounts)
	}
	if len(e.Configuration.PublishedPorts) != 1 || e.Configuration.PublishedPorts[0].HostPort != 8080 {
		t.Errorf("PublishedPorts = %+v", e.Configuration.PublishedPorts)
	}
	if e.Status.State != "running" {
		t.Errorf("State = %q, want running", e.Status.State)
	}
	if e.Status.StartedDate != "2024-06-15T10:30:00Z" {
		t.Errorf("StartedDate = %q", e.Status.StartedDate)
	}
	if len(e.Status.Networks) != 1 || e.Status.Networks[0].IPv4Address != "172.17.0.2" {
		t.Errorf("Networks = %+v", e.Status.Networks)
	}
}
