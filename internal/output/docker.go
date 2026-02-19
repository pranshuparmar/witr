package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pranshuparmar/witr/pkg/model"
)

// dockerSourceLabel builds the source label from a DockerPortMatch.
func dockerSourceLabel(match *model.DockerPortMatch) string {
	if match.ComposeProject != "" && match.ComposeService != "" {
		return fmt.Sprintf("docker-compose: %s/%s", match.ComposeProject, match.ComposeService)
	}
	return "docker"
}

// RenderDockerFallback renders Docker container info when /proc scanning fails
// but a container was identified via Docker CLI.
func RenderDockerFallback(w io.Writer, portValue string, match *model.DockerPortMatch, colorEnabled bool) {
	out := NewPrinter(w)

	name := SanitizeTerminal(match.Name)
	image := SanitizeTerminal(match.Image)
	ports := SanitizeTerminal(match.Ports)

	// Target
	if colorEnabled {
		out.Printf("%sTarget%s      : port %s\n\n", ColorBlue, ColorReset, portValue)
	} else {
		out.Printf("Target      : port %s\n\n", portValue)
	}

	// Container
	if colorEnabled {
		out.Printf("%sContainer%s   : %s%s%s\n", ColorBlue, ColorReset, ColorGreen, ansiString(name), ColorReset)
	} else {
		out.Printf("Container   : %s\n", name)
	}

	// Image
	if colorEnabled {
		out.Printf("%sImage%s       : %s\n", ColorBlue, ColorReset, ansiString(image))
	} else {
		out.Printf("Image       : %s\n", image)
	}

	// Ports
	if ports != "" {
		if colorEnabled {
			out.Printf("%sPorts%s       : %s\n", ColorBlue, ColorReset, ansiString(ports))
		} else {
			out.Printf("Ports       : %s\n", ports)
		}
	}

	// Why It Exists
	if colorEnabled {
		out.Printf("\n%sWhy It Exists%s :\n  ", ColorMagenta, ColorReset)
		out.Print("Docker Desktop (process not visible in current namespace)\n\n")
	} else {
		out.Printf("\nWhy It Exists :\n  ")
		out.Print("Docker Desktop (process not visible in current namespace)\n\n")
	}

	// Source
	sourceLabel := dockerSourceLabel(match)
	if colorEnabled {
		out.Printf("%sSource%s      : %s\n", ColorCyan, ColorReset, sourceLabel)
	} else {
		out.Printf("Source      : %s\n", sourceLabel)
	}

	// Note
	if colorEnabled {
		out.Printf("\n%sNote%s        : The owning process is not visible in this environment.\n", ColorDimYellow, ColorReset)
		out.Printf("              This is common when Docker Desktop runs in a separate namespace\n")
		out.Printf("              (e.g., WSL2 distro, macOS VM).\n")
	} else {
		out.Printf("\nNote        : The owning process is not visible in this environment.\n")
		out.Printf("              This is common when Docker Desktop runs in a separate namespace\n")
		out.Printf("              (e.g., WSL2 distro, macOS VM).\n")
	}
}

// RenderDockerFallbackShort renders a single-line summary for --short mode.
func RenderDockerFallbackShort(w io.Writer, portValue string, match *model.DockerPortMatch, colorEnabled bool) {
	out := NewPrinter(w)
	name := SanitizeTerminal(match.Name)
	image := SanitizeTerminal(match.Image)

	if colorEnabled {
		out.Printf("port %s → %s%s%s (%s) [%s]\n", portValue, ColorGreen, ansiString(name), ColorReset, ansiString(image), dockerSourceLabel(match))
	} else {
		out.Printf("port %s → %s (%s) [%s]\n", portValue, name, image, dockerSourceLabel(match))
	}
}

// DockerFallbackToJSON returns JSON output for a Docker fallback match.
func DockerFallbackToJSON(portValue string, match *model.DockerPortMatch) (string, error) {
	type dockerResult struct {
		Target         string
		ContainerID    string
		ContainerName  string
		Image          string
		Ports          string `json:",omitempty"`
		ComposeProject string `json:",omitempty"`
		ComposeService string `json:",omitempty"`
		Source         string
		Note           string
	}

	res := dockerResult{
		Target:         "port " + portValue,
		ContainerID:    match.ID,
		ContainerName:  match.Name,
		Image:          match.Image,
		Ports:          match.Ports,
		ComposeProject: match.ComposeProject,
		ComposeService: match.ComposeService,
		Source:         dockerSourceLabel(match),
		Note:           "The owning process is not visible in this environment. This is common when Docker Desktop runs in a separate namespace (e.g., WSL2 distro, macOS VM).",
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
