package proc

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
	"unicode"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ResolveContainerByPort queries the Docker CLI for a container publishing the given port.
// Returns nil if Docker is not available or no container matches.
func ResolveContainerByPort(port int) *model.DockerPortMatch {
	if _, err := exec.LookPath("docker"); err != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	format := "{{.ID}}|{{.Names}}|{{.Image}}|{{.Ports}}|{{.Label \"com.docker.compose.project\"}}|{{.Label \"com.docker.compose.service\"}}"
	cmd := exec.CommandContext(ctx, "docker", "ps", "--filter", fmt.Sprintf("publish=%d", port), "--format", format)
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	line := strings.TrimSpace(string(out))
	if line == "" {
		return nil
	}

	// Take the first matching container if multiple lines
	if idx := strings.Index(line, "\n"); idx >= 0 {
		line = line[:idx]
	}

	parts := strings.SplitN(line, "|", 6)
	if len(parts) < 6 {
		return nil
	}

	return &model.DockerPortMatch{
		ID:             parts[0],
		Name:           parts[1],
		Image:          parts[2],
		Ports:          parts[3],
		ComposeProject: parts[4],
		ComposeService: parts[5],
	}
}

// resolveContainerName attempts to resolve a container ID to a name using the specified runtime CLI.
func resolveContainerName(id, runtime string) string {
	var cmd *exec.Cmd
	var prefix string

	switch runtime {
	case "docker":
		if _, err := exec.LookPath("docker"); err != nil {
			return ""
		}
		cmd = exec.Command("docker", "inspect", id, "--format", "{{.Name}}|{{index .Config.Labels \"com.docker.compose.project\"}}|{{index .Config.Labels \"com.docker.compose.service\"}}")
		prefix = "docker: "
	case "podman":
		if _, err := exec.LookPath("podman"); err != nil {
			return ""
		}
		cmd = exec.Command("podman", "inspect", id, "--format", "{{.Name}}")
		prefix = "podman: "
	case "crictl":
		if _, err := exec.LookPath("crictl"); err != nil {
			return ""
		}
		cmd = exec.Command("crictl", "inspect", id, "-o", "go-template", "--template", "{{.status.metadata.name}}")
		prefix = "" // crictl names are usually clean
	case "nerdctl":
		if _, err := exec.LookPath("nerdctl"); err != nil {
			return ""
		}
		cmd = exec.Command("nerdctl", "inspect", id, "--format", "{{.Name}}")
		prefix = "containerd: "
	default:
		return ""
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	output := strings.TrimSpace(string(out))

	if runtime == "docker" {
		parts := strings.Split(output, "|")
		if len(parts) == 3 {
			name := strings.TrimPrefix(parts[0], "/")
			project := parts[1]
			service := parts[2]

			if project != "" && service != "" {
				return "docker: " + project + "/" + service + " (" + name + ")"
			}
			if name != "" {
				return "docker: " + name
			}
			return ""
		}
	}

	name := strings.TrimPrefix(output, "/")
	if name != "" {
		if prefix != "" {
			return prefix + name
		}
		return name
	}
	return ""
}

// findLongHexID searches for a 64-character hexadecimal string in the input.
func findLongHexID(s string) string {
	for i := 0; i <= len(s)-64; i++ {
		if s[i] < '0' || (s[i] > '9' && s[i] < 'a') {
			continue
		}
		sub := s[i : i+64]
		isHex := true
		for j := 0; j < 64; j++ {
			c := sub[j]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				isHex = false
				break
			}
		}
		if isHex {
			return sub
		}
	}
	return ""
}

// extractFlagValue extracts the value of a specific flag from a command line string.
func extractFlagValue(cmdline string, flags ...string) string {
	args := splitCmdline(cmdline)
	for i, arg := range args {
		for _, flag := range flags {
			if arg == flag && i+1 < len(args) {
				return args[i+1]
			}
		}
	}
	return ""
}

// splitCmdline splits a command line string into arguments, handling quotes and escapes.
func splitCmdline(cmdline string) []string {
	var args []string
	var current strings.Builder
	var quote rune
	escaped := false
	for _, r := range cmdline {
		switch {
		case escaped:
			current.WriteRune(r)
			escaped = false
		case r == '\\':
			escaped = true
		case r == '"' || r == '\'':
			if quote == 0 {
				quote = r
				continue
			}
			if quote == r {
				quote = 0
				continue
			}
			current.WriteRune(r)
		case unicode.IsSpace(r) && quote == 0:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}
