package proc

import (
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDockerProxyPublish(t *testing.T) {
	tests := []struct {
		cmdline   string
		wantPort  int
		wantProto string
		wantOK    bool
	}{
		{"/usr/bin/docker-proxy -proto tcp -host-ip 0.0.0.0 -host-port 3120 -container-ip 172.29.0.2 -container-port 9000", 3120, "tcp", true},
		{"/usr/bin/docker-proxy -proto udp -host-ip 0.0.0.0 -host-port 53 -container-ip 172.17.0.2 -container-port 53", 53, "udp", true},
		{"/usr/bin/docker-proxy -host-ip 0.0.0.0 -host-port 8080", 8080, "tcp", true},
		{"/usr/bin/docker-proxy -proto tcp -host-ip 0.0.0.0 -host-port", 0, "tcp", false},
		{"/usr/bin/docker-proxy -host-port abc", 0, "tcp", false},
		{"/usr/bin/docker-proxy -container-port 9000", 0, "tcp", false},
	}
	for _, tt := range tests {
		port, proto, ok := dockerProxyPublish(tt.cmdline)
		if port != tt.wantPort || proto != tt.wantProto || ok != tt.wantOK {
			t.Errorf("dockerProxyPublish(%q) = %d, %q, %v; want %d, %q, %v", tt.cmdline, port, proto, ok, tt.wantPort, tt.wantProto, tt.wantOK)
		}
	}
}

func TestContainerDetailsHealthcheckFromKnownContainer(t *testing.T) {
	// A known container is used as is, so none of these query a runtime.
	tests := []struct {
		name    string
		runtime string
		health  string
		want    string
	}{
		{"docker with a health state", "docker", "healthy", "present"},
		{"docker still starting", "docker", "starting", "present"},
		{"docker without one", "docker", "", "absent"},
		{"podman without one", "podman", "", "absent"},
		{"runtime without healthchecks", "nerdctl", "", ""},
	}
	for _, tt := range tests {
		known := &model.ContainerMatch{ID: "c67b85f01c07", Health: tt.health}
		c, hc := ContainerDetails("c67b85f01c07", tt.runtime, known)
		if c != known || hc != tt.want {
			t.Errorf("%s: ContainerDetails = %p, %q; want the known container and %q", tt.name, c, hc, tt.want)
		}
	}
}

func TestContainerByIDRejectsUnsupportedInput(t *testing.T) {
	// Neither case may reach a runtime CLI: an ID that could parse as an
	// option, or a runtime without a docker-compatible `ps`.
	if c := ContainerByID("--all", "docker"); c != nil {
		t.Errorf("ContainerByID with an option-like ID = %+v, want nil", c)
	}
	if c := ContainerByID("c67b85f01c07", "crictl"); c != nil {
		t.Errorf("ContainerByID for crictl = %+v, want nil", c)
	}
}

func TestMatchContainerID(t *testing.T) {
	const id = "c67b85f01c07a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	tests := []struct {
		name  string
		query string
		exact bool
		want  bool
	}{
		{"short ID", "c67b85f01c07", false, true},
		{"full ID", id, false, true},
		{"short prefix", "c67b", false, true},
		{"too short to be an ID", "c67", false, false},
		{"not hex", "rustfs", false, false},
		{"other ID", "deadbeef", false, false},
		{"exact short ID", "c67b85f01c07", true, true},
		{"exact full ID", id, true, true},
		{"exact rejects a bare prefix", "c67b", true, false},
	}
	for _, tt := range tests {
		if got := matchContainerID(id, tt.query, tt.exact); got != tt.want {
			t.Errorf("%s: matchContainerID(%q, exact=%v) = %v, want %v", tt.name, tt.query, tt.exact, got, tt.want)
		}
	}
}

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
		{"uppercase hex", "/docker/" + "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2" + "/cgroup", "A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2C3D4E5F6A1B2"},
		{"mixed case hex", "/docker/" + "a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2" + "/cgroup", "a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2c3D4e5F6a1B2"},
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

func TestShortID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"a1b2c3d4e5f6a1b2c3d4e5f6", "a1b2c3d4e5f6"},
		{"a1b2c3d4e5f6", "a1b2c3d4e5f6"},
		{"a1b2", "a1b2"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := shortID(tt.in); got != tt.want {
			t.Errorf("shortID(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestIsValidContainerID(t *testing.T) {
	tests := []struct {
		in   string
		want bool
	}{
		{"a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2", true}, // 64-hex
		{"a1b2c3d4e5f6", true},   // short id
		{"my_app.1-name", true},  // separators mid-token
		{"", false},              // empty
		{"-rf", false},           // leading dash → would parse as a flag
		{"--format", false},      // leading dash
		{".hidden", false},       // leading separator
		{"id with space", false}, // whitespace
		{"id;rm -rf", false},     // shell metacharacter
		{"id\n--format", false},  // embedded newline
		{"a/b", false},           // slash
	}
	for _, tt := range tests {
		if got := isValidContainerID(tt.in); got != tt.want {
			t.Errorf("isValidContainerID(%q) = %v, want %v", tt.in, got, tt.want)
		}
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

// Published host ports are matched from the ps listing, ranges and protocol
// included; an exposed-only port isn't published.
func TestPublishesPort(t *testing.T) {
	ports := "0.0.0.0:18200->80/tcp, [::]:18200->80/tcp, 127.0.0.1:8000-8002->80-82/tcp, 0.0.0.0:5353->53/udp, 9000/tcp"
	tests := []struct {
		port  int
		proto string
		want  bool
	}{
		{18200, "", true},
		{18200, "tcp", true},
		{8001, "tcp", true},
		{8003, "tcp", false},
		{5353, "udp", true},
		{5353, "tcp", false},
		{5353, "", true},
		{9000, "tcp", false},
		{80, "tcp", false},
	}
	for _, tt := range tests {
		if got := publishesPort(ports, tt.port, tt.proto); got != tt.want {
			t.Errorf("publishesPort(%d, %q) = %v, want %v", tt.port, tt.proto, got, tt.want)
		}
	}
}

// Every installed runtime is listed; Docker's answer wins over Podman's, and
// Podman's over nerdctl's.
func TestResolveContainerByPort(t *testing.T) {
	orig := listRuntimeContainers
	defer func() { listRuntimeContainers = orig }()
	lists := map[string][]*model.ContainerMatch{
		"docker":  {{Name: "d-web", Ports: "0.0.0.0:8080->80/tcp, [::]:8080->80/tcp"}},
		"podman":  {{Name: "p-web", Ports: "0.0.0.0:8080->80/tcp"}, {Name: "p-dns", Ports: "0.0.0.0:5353->53/udp"}},
		"nerdctl": {{Name: "n-dns", Ports: "0.0.0.0:5353->53/udp"}, {Name: "n-range", Ports: "0.0.0.0:9000-9002->9000-9002/tcp"}},
	}
	listRuntimeContainers = func(bin string) []*model.ContainerMatch { return lists[bin] }

	tests := []struct {
		port        int
		proto, want string
	}{
		{8080, "tcp", "d-web"},
		{8080, "", "d-web"},
		{5353, "udp", "p-dns"},
		{5353, "tcp", ""},
		{9001, "", "n-range"},
		{7000, "", ""},
	}
	for _, tt := range tests {
		got := ""
		if m := ResolveContainerByPort(tt.port, tt.proto); m != nil {
			got = m.Name
		}
		if got != tt.want {
			t.Errorf("ResolveContainerByPort(%d, %q) = %q, want %q", tt.port, tt.proto, got, tt.want)
		}
	}
}
