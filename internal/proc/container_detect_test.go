package proc

import "testing"

func TestDetectContainerFromCmdline(t *testing.T) {
	tests := []struct {
		cmdline string
		want    string
	}{
		{"", ""},
		{"/usr/sbin/nginx -g daemon off;", ""},
		{"/usr/bin/dockerd --containerd /run/containerd.sock", "docker"},
		{"docker run --name web nginx", "docker: web"},
		{"podman run --name db postgres", "podman: db"},
		{"kind create cluster --name dev", "k8s: dev"},
		{"nerdctl run --name app alpine", "containerd: app"},
		{"minikube start", "kubernetes"},
		{"minikube start -p dev", "k8s: dev"},
		{"colima start", "colima: default"},
		{"colima start --profile work", "colima: work"},
		{"/run/kubepods/besteffort/podxyz/shim", "kubernetes"},
		{"/usr/bin/containerd-shim-runc-v2 -namespace moby", "containerd"},
		{"podman", "podman"},
		{"/Applications/AppleContainer.app/Contents/MacOS/com.apple.container --name myapp", "container: myapp"},
		{"/usr/libexec/apple.container --name web", "container: web"},
		{"apple.container run nginx", "container"},
	}
	for _, tt := range tests {
		if got := detectContainerFromCmdline(tt.cmdline); got != tt.want {
			t.Errorf("detectContainerFromCmdline(%q) = %q, want %q", tt.cmdline, got, tt.want)
		}
	}
}
