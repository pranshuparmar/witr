package proc

import "strings"

// detectContainerFromCmdline checks the command line for container runtime patterns.
// Used by darwin, freebsd, and windows where cgroup-based detection is not available.
func detectContainerFromCmdline(cmdline string) string {
	if cmdline == "" {
		return ""
	}
	lowerCmd := strings.ToLower(cmdline)

	switch {
	case strings.Contains(lowerCmd, "docker"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "docker: " + name
		}
		return "docker"
	case strings.Contains(lowerCmd, "podman"), strings.Contains(lowerCmd, "libpod"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "podman: " + name
		}
		return "podman"
	case strings.Contains(lowerCmd, "minikube"):
		if profile := extractFlagValue(cmdline, "-p", "--profile"); profile != "" {
			return "k8s: " + profile
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "kind"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "k8s: " + name
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "kubepods"):
		if id := findLongHexID(cmdline); id != "" {
			if name := resolveContainerName(id, "crictl"); name != "" {
				return "k8s: " + name
			}
			return "k8s (" + shortID(id) + ")"
		}
		return "kubernetes"
	case strings.Contains(lowerCmd, "colima"):
		if profile := extractFlagValue(cmdline, "-p", "--profile"); profile != "" {
			return "colima: " + profile
		}
		return "colima: default"
	case strings.Contains(lowerCmd, "nerdctl"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "containerd: " + name
		}
		return "containerd"
	case strings.Contains(lowerCmd, "containerd"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "containerd: " + name
		}
		return "containerd"
	case strings.Contains(lowerCmd, "apple.container"):
		if name := extractFlagValue(cmdline, "--name"); name != "" {
			return "container: " + name
		}
		return "container"
	}

	return ""
}
