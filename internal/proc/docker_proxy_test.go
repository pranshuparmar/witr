package proc

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

// The forwarder check matches on the executable name.
func TestImageNameOfSelf(t *testing.T) {
	self, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	// Linux keeps only the first 15 characters of the name.
	if got := imageName(os.Getpid()); got == "" || !strings.HasPrefix(filepath.Base(self), got) {
		t.Errorf("imageName(self) = %q, want a prefix of %q", got, filepath.Base(self))
	}
}

// Rootless runtimes publish through helpers that serve every port.
func TestIsPortForwarder(t *testing.T) {
	for _, name := range []string{"rootlessport", "rootlesskit", "pasta", "pasta.avx2", "wslrelay.exe", "com.docker.backend"} {
		if !isPortForwarder(name) {
			t.Errorf("isPortForwarder(%q) = false", name)
		}
	}
	for _, name := range []string{"nginx", "pastafarian", "python3"} {
		if isPortForwarder(name) {
			t.Errorf("isPortForwarder(%q) = true", name)
		}
	}
}

// An ordinary listener never makes a port a container's.
func TestPublishedContainerIgnoresOrdinaryListeners(t *testing.T) {
	if match, _ := PublishedContainer(80, []int{os.Getpid()}); match != nil {
		t.Errorf("PublishedContainer with a plain process = %+v, want nil", match)
	}
	if match, _ := PublishedContainer(80, nil); match != nil {
		t.Errorf("PublishedContainer with no listeners = %+v, want nil", match)
	}
}

// A port is a container's only when every listener just publishes it; the
// runtime is then asked over docker-proxy's protocol, or either for forwarders.
func TestPublishedContainerDecision(t *testing.T) {
	origCmdline, origName, origByPort := cmdlineOf, imageNameOf, containerByPort
	defer func() { cmdlineOf, imageNameOf, containerByPort = origCmdline, origName, origByPort }()

	cmdlines := map[int]string{
		10: "/usr/bin/docker-proxy -proto udp -host-ip 0.0.0.0 -host-port 5353 -container-ip 172.17.0.2 -container-port 53",
		11: "/usr/bin/docker-proxy -proto tcp -host-ip 0.0.0.0 -host-port 8080 -container-ip 172.17.0.3 -container-port 80",
		20: "nginx: master process nginx",
	}
	names := map[int]string{10: "docker-proxy", 11: "docker-proxy", 20: "nginx", 30: "rootlessport", 31: "pasta.avx2", 32: "com.docker.backend.exe"}
	var asked []string
	cmdlineOf = func(pid int) string { return cmdlines[pid] }
	imageNameOf = func(pid int) string { return names[pid] }
	containerByPort = func(port int, proto string) *model.ContainerMatch {
		asked = append(asked, fmt.Sprintf("%d/%s", port, proto))
		if port == 9999 {
			return nil
		}
		return &model.ContainerMatch{Name: "web"}
	}

	tests := []struct {
		name      string
		port      int
		pids      []int
		wantNames []string
		wantAsked string
	}{
		{"docker-proxy names its protocol", 5353, []int{10}, []string{"docker-proxy"}, "5353/udp"},
		{"docker-proxy beside a forwarder", 5353, []int{10, 30}, []string{"docker-proxy", "rootlessport"}, "5353/udp"},
		{"forwarders carry either protocol", 8080, []int{30, 31, 32}, []string{"rootlessport", "pasta.avx2", "com.docker.backend.exe"}, "8080/"},
		{"an ordinary listener among them", 8080, []int{30, 20}, nil, ""},
		{"a docker-proxy for another port", 5353, []int{11}, nil, ""},
		{"no container publishes it", 9999, []int{30}, nil, "9999/"},
	}
	for _, tt := range tests {
		asked = nil
		match, got := PublishedContainer(tt.port, tt.pids)
		if (match != nil) != (tt.wantNames != nil) || !slices.Equal(got, tt.wantNames) {
			t.Errorf("%s: match %v, names %q; want names %q", tt.name, match, got, tt.wantNames)
		}
		if a := strings.Join(asked, ","); a != tt.wantAsked {
			t.Errorf("%s: runtime asked %q, want %q", tt.name, a, tt.wantAsked)
		}
	}
}
