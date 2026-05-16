package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func containerSourceLabel(match *model.ContainerMatch) string {
	if match.Runtime == "docker" && match.ComposeProject != "" && match.ComposeService != "" {
		return fmt.Sprintf("docker-compose: %s/%s", match.ComposeProject, match.ComposeService)
	}
	if match.Runtime != "" {
		return match.Runtime
	}
	return "container"
}

// containerChain returns the conceptual ancestry segments for a container:
// runtime → [compose project] → container.
func containerChain(match *model.ContainerMatch) []string {
	runtime := match.Runtime
	if runtime == "" {
		runtime = "container"
	}
	segs := []string{runtime}
	if match.ComposeProject != "" {
		segs = append(segs, match.ComposeProject+" (docker-compose)")
	}
	segs = append(segs, match.Name)
	return segs
}

// containerStateTag returns the bracketed suffix for the Container line —
// mirrors the [zombie] / [stopped] convention used for processes. Returns ""
// when the container is healthy and running (no need to clutter the line).
func containerStateTag(match *model.ContainerMatch) string {
	state := strings.ToLower(match.State)
	health := strings.ToLower(match.Health)

	if health != "" && health != "healthy" {
		return health
	}
	if health == "healthy" {
		return "healthy"
	}
	if state != "" && state != "running" {
		return state
	}
	return ""
}

func ShortContainerID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// FormatContainerLine produces the value rendered after "Container   : ", in
// the same shape used by both the fallback render and the standard process
// render for `-c` targets: "<runtime>: <name> (id <short>) [state-or-health]".
// Empty tags are omitted.
func FormatContainerLine(match *model.ContainerMatch) string {
	if match == nil {
		return ""
	}
	parts := match.Name
	if match.Runtime != "" {
		parts = match.Runtime + ": " + parts
	}
	if id := ShortContainerID(match.ID); id != "" {
		parts += " (id " + id + ")"
	}
	if tag := containerStateTag(match); tag != "" {
		parts += " [" + tag + "]"
	}
	return parts
}

func RenderContainerFallback(w io.Writer, targetLabel string, match *model.ContainerMatch, colorEnabled bool, verbose bool) {
	out := NewPrinter(w)

	name := SanitizeTerminal(match.Name)
	image := SanitizeTerminal(match.Image)
	command := SanitizeTerminal(match.Command)
	id := SanitizeTerminal(ShortContainerID(match.ID))
	stateTag := SanitizeTerminal(containerStateTag(match))
	networks := SanitizeTerminal(match.Networks)
	ports := SanitizeTerminal(match.Ports)

	if colorEnabled {
		out.Printf("%sTarget%s      : %s\n\n", ColorBlue, ColorReset, targetLabel)
	} else {
		out.Printf("Target      : %s\n\n", targetLabel)
	}

	// Container line: name (id <short>) [state-or-health]
	if colorEnabled {
		out.Printf("%sContainer%s   : %s%s%s (%sid %s%s)",
			ColorBlue, ColorReset, ColorGreen, ansiString(name), ColorReset,
			ColorDim, id, ColorReset)
	} else {
		out.Printf("Container   : %s (id %s)", name, id)
	}
	if stateTag != "" {
		color := ColorRed
		if stateTag == "healthy" {
			color = ColorGreen
		}
		if colorEnabled {
			out.Printf(" %s[%s]%s", color, stateTag, ColorReset)
		} else {
			out.Printf(" [%s]", stateTag)
		}
	}
	out.Println("")

	if image != "" {
		if colorEnabled {
			out.Printf("%sImage%s       : %s\n", ColorBlue, ColorReset, ansiString(image))
		} else {
			out.Printf("Image       : %s\n", image)
		}
	}

	if command != "" {
		if colorEnabled {
			out.Printf("%sCommand%s     : %s\n", ColorBlue, ColorReset, command)
		} else {
			out.Printf("Command     : %s\n", command)
		}
	}

	if !match.StartedAt.IsZero() {
		rel, dtStr := FormatStartedAt(match.StartedAt)
		if colorEnabled {
			out.Printf("%sStarted%s     : %s (%s)\n", ColorMagenta, ColorReset, rel, dtStr)
		} else {
			out.Printf("Started     : %s (%s)\n", rel, dtStr)
		}
	}
	if !match.CreatedAt.IsZero() && (match.StartedAt.IsZero() || !match.CreatedAt.Equal(match.StartedAt)) {
		_, dtStr := FormatStartedAt(match.CreatedAt)
		if colorEnabled {
			out.Printf("%sCreated%s     : %s\n", ColorBlue, ColorReset, dtStr)
		} else {
			out.Printf("Created     : %s\n", dtStr)
		}
	}

	if networks != "" {
		if colorEnabled {
			out.Printf("%sNetwork%s     : %s\n", ColorBlue, ColorReset, networks)
		} else {
			out.Printf("Network     : %s\n", networks)
		}
	}

	if colorEnabled {
		out.Printf("\n%sWhy It Exists%s :\n  ", ColorMagenta, ColorReset)
	} else {
		out.Printf("\nWhy It Exists :\n  ")
	}
	writeContainerChainInline(out, containerChain(match), colorEnabled)
	out.Println("")

	sourceLabel := containerSourceLabel(match)
	if colorEnabled {
		out.Printf("\n%sSource%s      : %s\n", ColorCyan, ColorReset, sourceLabel)
	} else {
		out.Printf("\nSource      : %s\n", sourceLabel)
	}

	if ports != "" {
		printContainerSockets(out, match.Ports, colorEnabled)
	}

	if verbose {
		mounts := SanitizeTerminal(match.Mounts)
		if mounts != "" {
			if colorEnabled {
				out.Printf("\n%sMounts%s      : %s\n", ColorBlue, ColorReset, mounts)
			} else {
				out.Printf("\nMounts      : %s\n", mounts)
			}
		}
		if match.ComposeConfigFile != "" {
			if colorEnabled {
				out.Printf("%sCompose File%s: %s\n", ColorBlue, ColorReset, SanitizeTerminal(match.ComposeConfigFile))
			} else {
				out.Printf("Compose File: %s\n", SanitizeTerminal(match.ComposeConfigFile))
			}
		}
		if match.ComposeWorkingDir != "" {
			if colorEnabled {
				out.Printf("%sCompose Dir%s : %s\n", ColorBlue, ColorReset, SanitizeTerminal(match.ComposeWorkingDir))
			} else {
				out.Printf("Compose Dir : %s\n", SanitizeTerminal(match.ComposeWorkingDir))
			}
		}
	}

	if colorEnabled {
		out.Printf("\n%sNote%s        : The owning process is not visible in this environment.\n", ColorDimYellow, ColorReset)
	} else {
		out.Printf("\nNote        : The owning process is not visible in this environment.\n")
	}
}

// printContainerSockets parses the comma-separated docker port mapping string
// into per-row entries under the standard Sockets label.
func printContainerSockets(out Printer, ports string, colorEnabled bool) {
	entries := strings.Split(ports, ", ")
	for i, e := range entries {
		safe := SanitizeTerminal(strings.TrimSpace(e))
		switch {
		case i == 0 && colorEnabled:
			out.Printf("%sSockets%s     : %s\n", ColorGreen, ColorReset, safe)
		case i == 0:
			out.Printf("Sockets     : %s\n", safe)
		default:
			out.Printf("              %s\n", safe)
		}
	}
}

func RenderContainerFallbackShort(w io.Writer, _ string, match *model.ContainerMatch, colorEnabled bool) {
	out := NewPrinter(w)
	writeContainerChainInline(out, containerChain(match), colorEnabled)
	out.Println("")
}

func RenderContainerFallbackTree(w io.Writer, match *model.ContainerMatch, colorEnabled bool) {
	out := NewPrinter(w)
	segs := containerChain(match)
	for i, s := range segs {
		s = SanitizeTerminal(s)
		indent := strings.Repeat("  ", i)
		switch {
		case i == 0 && colorEnabled && len(segs) == 1:
			out.Printf("%s%s%s\n", ColorGreen, s, ColorReset)
		case i == 0:
			out.Printf("%s\n", s)
		case i == len(segs)-1 && colorEnabled:
			out.Printf("%s%s└─ %s%s%s\n", indent, ColorMagenta, ColorGreen, s, ColorReset)
		case colorEnabled:
			out.Printf("%s%s└─ %s%s\n", indent, ColorMagenta, ColorReset, s)
		default:
			out.Printf("%s└─ %s\n", indent, s)
		}
	}
}

func RenderContainerFallbackWarnings(w io.Writer, match *model.ContainerMatch, colorEnabled bool) {
	out := NewPrinter(w)
	name := SanitizeTerminal(match.Name)
	if colorEnabled {
		out.Printf("%sContainer%s   : %s%s%s\n", ColorBlue, ColorReset, ColorGreen, ansiString(name), ColorReset)
		out.Printf("%sWarnings%s    : %sNo warnings (workload process not visible).%s\n", ColorRed, ColorReset, ColorGreen, ColorReset)
	} else {
		out.Printf("Container   : %s\n", name)
		out.Println("Warnings    : No warnings (workload process not visible).")
	}
}

func writeContainerChainInline(out Printer, segs []string, colorEnabled bool) {
	for i, s := range segs {
		s = SanitizeTerminal(s)
		if i > 0 {
			if colorEnabled {
				out.Printf(" %s→%s ", ColorMagenta, ColorReset)
			} else {
				out.Print(" → ")
			}
		}
		if i == len(segs)-1 && colorEnabled {
			out.Printf("%s%s%s", ColorGreen, s, ColorReset)
		} else {
			out.Print(s)
		}
	}
}

func ContainerFallbackToJSON(targetLabel string, match *model.ContainerMatch) (string, error) {
	type containerResult struct {
		Target            string
		Runtime           string
		ContainerID       string
		ContainerName     string
		Image             string
		Command           string `json:",omitempty"`
		State             string `json:",omitempty"`
		Status            string `json:",omitempty"`
		Health            string `json:",omitempty"`
		CreatedAt         string `json:",omitempty"`
		StartedAt         string `json:",omitempty"`
		Networks          string `json:",omitempty"`
		Mounts            string `json:",omitempty"`
		Ports             string `json:",omitempty"`
		ComposeProject    string `json:",omitempty"`
		ComposeService    string `json:",omitempty"`
		ComposeConfigFile string `json:",omitempty"`
		ComposeWorkingDir string `json:",omitempty"`
		Source            string
		Chain             []string
		Note              string
	}

	created := ""
	if !match.CreatedAt.IsZero() {
		created = match.CreatedAt.Format("Mon 2006-01-02 15:04:05 -07:00")
	}
	started := ""
	if !match.StartedAt.IsZero() {
		started = match.StartedAt.Format("Mon 2006-01-02 15:04:05 -07:00")
	}

	res := containerResult{
		Target:            targetLabel,
		Runtime:           match.Runtime,
		ContainerID:       match.ID,
		ContainerName:     match.Name,
		Image:             match.Image,
		Command:           match.Command,
		State:             match.State,
		Status:            match.Status,
		Health:            match.Health,
		CreatedAt:         created,
		StartedAt:         started,
		Networks:          match.Networks,
		Mounts:            match.Mounts,
		Ports:             match.Ports,
		ComposeProject:    match.ComposeProject,
		ComposeService:    match.ComposeService,
		ComposeConfigFile: match.ComposeConfigFile,
		ComposeWorkingDir: match.ComposeWorkingDir,
		Source:            containerSourceLabel(match),
		Chain:             containerChain(match),
		Note:              "The owning process is not visible in this environment. This is common when the runtime runs in a separate namespace (e.g., Docker Desktop, WSL2 distro, macOS VM).",
	}

	data, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
