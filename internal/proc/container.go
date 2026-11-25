package proc

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ResolveContainerByPort returns the container publishing host port over
// proto ("tcp", "udp", or "" for either) from Docker, Podman or nerdctl,
// whichever are installed, or nil. It matches the published ports itself,
// since Podman has no publish filter, and lists the runtimes at once, since a
// rootless runtime takes most of a second to answer.
func ResolveContainerByPort(port int, proto string) *model.ContainerMatch {
	bins := []string{"docker", "podman", "nerdctl"}
	found := make([]*model.ContainerMatch, len(bins))
	var wg sync.WaitGroup
	for i, bin := range bins {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for _, m := range listRuntimeContainers(bin) {
				if publishesPort(m.Ports, port, proto) {
					found[i] = m
					return
				}
			}
		}()
	}
	wg.Wait()
	for _, m := range found {
		if m != nil {
			return m
		}
	}
	return nil
}

// listRuntimeContainers lists the running containers of a docker-compatible
// runtime, or nothing when it isn't installed. A variable for tests.
var listRuntimeContainers = func(bin string) []*model.ContainerMatch {
	if !binAvailable(bin) {
		return nil
	}
	return dockerLikeList(bin, dockerLikeRuntimeLabels[bin])
}

// publishesPort reports whether a container's published ports, as `ps` lists
// them ("0.0.0.0:8080->80/tcp, [::]:8000-8002->80-82/udp"), include host port
// over proto ("" for either).
func publishesPort(ports string, port int, proto string) bool {
	for _, e := range strings.Split(ports, ",") {
		host, target, ok := strings.Cut(strings.TrimSpace(e), "->")
		if !ok {
			continue // exposed, not published
		}
		p := "tcp"
		if _, s, ok := strings.Cut(target, "/"); ok {
			p = s
		}
		if proto != "" && p != proto {
			continue
		}
		first, last, isRange := strings.Cut(host[strings.LastIndex(host, ":")+1:], "-")
		lo, err := strconv.Atoi(first)
		if err != nil {
			continue
		}
		hi := lo
		if isRange {
			if hi, err = strconv.Atoi(last); err != nil {
				continue
			}
		}
		if port >= lo && port <= hi {
			return true
		}
	}
	return false
}

// ContainerByID returns the Docker, Podman or nerdctl container with the given
// ID, or nil when the runtime can't be queried or doesn't know it.
func ContainerByID(id, runtime string) *model.ContainerMatch {
	label, ok := dockerLikeRuntimeLabels[runtime]
	if !ok || !isValidContainerID(id) {
		return nil
	}
	if ms := dockerLikeList(runtime, label, "--filter", "id="+id); len(ms) == 1 {
		return ms[0]
	}
	return nil
}

// dockerLikeRuntimeLabels maps each docker-compatible CLI to the runtime name
// its containers are reported under.
var dockerLikeRuntimeLabels = map[string]string{
	"docker":  "docker",
	"podman":  "podman",
	"nerdctl": "containerd",
}

// ContainerDetails returns the container a process runs in and whether it has
// a healthcheck: "present", "absent", or "" when that can't be told. known, if
// set, is the container already looked up, sparing another runtime query.
func ContainerDetails(id, runtime string, known *model.ContainerMatch) (*model.ContainerMatch, string) {
	c := known
	if c == nil {
		// The list scan has no restart count, policy or start time.
		if c = ContainerByID(id, runtime); c != nil {
			EnrichContainer(c)
		}
	}
	switch {
	case c == nil:
		return nil, ContainerHealthcheckStatus(id, runtime)
	case runtime != "docker" && runtime != "podman":
		return c, ""
	case c.Health != "":
		// `ps` reports a health state exactly when a healthcheck is set.
		return c, "present"
	}
	return c, "absent"
}

// isValidContainerID reports whether id is a safe container identifier to hand
// to a runtime CLI: a non-empty token of [A-Za-z0-9_.-] that begins with an
// alphanumeric. Rejecting a leading dash (and any other metacharacter) keeps a
// malformed, cgroup-derived id from being parsed as a CLI option.
func isValidContainerID(id string) bool {
	if id == "" {
		return false
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		case (c == '_' || c == '.' || c == '-') && i > 0:
		default:
			return false
		}
	}
	return true
}

// resolveContainerName attempts to resolve a container ID to a name using the specified runtime CLI.
func resolveContainerName(id, runtime string) string {
	if !isValidContainerID(id) {
		return ""
	}
	var cmd *exec.Cmd
	var prefix string

	ctx := context.Background()
	switch runtime {
	case "docker", "podman", "nerdctl":
		return dockerLikeContainerName(id, runtime)
	case "crictl":
		if _, err := exec.LookPath("crictl"); err != nil {
			return ""
		}
		cmd = exec.CommandContext(ctx, "crictl", "inspect", id, "-o", "go-template", "--template", "{{.status.metadata.name}}")
		prefix = "" // crictl names are usually clean
	default:
		return ""
	}

	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	output := strings.TrimSpace(string(out))

	name := strings.TrimPrefix(output, "/")
	if name != "" {
		if prefix != "" {
			return prefix + name
		}
		return name
	}
	return ""
}

// dockerLikeContainerName labels a Docker, Podman or nerdctl container for
// the Container line: "docker: project/service (name)" for a Compose
// service, otherwise "<runtime>: name".
func dockerLikeContainerName(id, runtime string) string {
	d, ok := inspectContainer(runtime, id)
	name := strings.TrimPrefix(d.Name, "/")
	if !ok || name == "" {
		return ""
	}
	if runtime == "docker" {
		project, service := d.Config.Labels["com.docker.compose.project"], d.Config.Labels["com.docker.compose.service"]
		if project != "" && service != "" {
			return "docker: " + project + "/" + service + " (" + name + ")"
		}
	}
	return map[string]string{"docker": "docker: ", "podman": "podman: ", "nerdctl": "containerd: "}[runtime] + name
}

// ContainerHealthcheckStatus reports whether the container runtime has a
// healthcheck configured: "present", "absent", or "" when undeterminable
// (runtime unavailable, inspect error, or unsupported runtime).
func ContainerHealthcheckStatus(id, runtime string) string {
	if !isValidContainerID(id) || (runtime != "docker" && runtime != "podman") {
		return ""
	}
	d, ok := inspectContainer(runtime, id)
	switch {
	case !ok:
		return ""
	case len(d.Config.Healthcheck) > 0 && string(d.Config.Healthcheck) != "null":
		return "present"
	}
	return "absent"
}

// findLongHexID searches for a 64-character hexadecimal string in the input.
func findLongHexID(s string) string {
	for i := 0; i <= len(s)-64; i++ {
		c := s[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
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

// shortID returns the first 12 characters of a container ID, or the full
// string if it is shorter than 12 characters.
func shortID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
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
