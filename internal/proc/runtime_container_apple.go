package proc

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func init() { registerRuntime(appleContainerRuntime{}) }

type appleContainerRuntime struct{}

func (appleContainerRuntime) Name() string    { return "container" }
func (appleContainerRuntime) Available() bool { return binAvailable("container") }

func (appleContainerRuntime) List() []*model.ContainerMatch {
	return appleContainerList()
}

// HostPID returns 0 because Apple container runs containers as lightweight VMs
// using the macOS Virtualization framework; there is no direct host PID mapping
// for container processes.
func (appleContainerRuntime) HostPID(id string) int { return 0 }

func (appleContainerRuntime) Enrich(match *model.ContainerMatch) {
	if match == nil || match.ID == "" {
		return
	}
	// Use container inspect to get the actual start time, similar to dockerLikeEnrich.
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "container", "inspect", match.ID).Output()
	if err != nil {
		return
	}
	var raw appleContainerEntry
	if err := json.Unmarshal(out, &raw); err != nil {
		return
	}
	if raw.Status.StartedDate != "" {
		if t, err := time.Parse(time.RFC3339Nano, raw.Status.StartedDate); err == nil {
			match.StartedAt = t
		}
	}
}

// ---------------------------------------------------------------------------
// JSON structures matching `container ls --format json --all` output
// ---------------------------------------------------------------------------

// appleContainerEntry mirrors the top-level ManagedContainer JSON shape.
type appleContainerEntry struct {
	Configuration appleContainerConfig `json:"configuration"`
	Status        appleContainerStatus `json:"status"`
}

type appleContainerConfig struct {
	ID             string               `json:"id"`
	Image          appleImageDesc       `json:"image"`
	Platform       applePlatform        `json:"platform"`
	InitProcess    appleProcessConfig   `json:"initProcess"`
	Resources      appleResources       `json:"resources"`
	Mounts         []appleMount         `json:"mounts"`
	PublishedPorts []applePublishedPort `json:"publishedPorts"`
	Networks       []appleNetworkConfig `json:"networks"`
	CreationDate   string               `json:"creationDate"`
}

type appleImageDesc struct {
	Reference string `json:"reference"`
}

type applePlatform struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

type appleProcessConfig struct {
	Executable string   `json:"executable"`
	Arguments  []string `json:"arguments"`
}

type appleResources struct {
	CPUs          int    `json:"cpus"`
	MemoryInBytes uint64 `json:"memoryInBytes"`
}

type appleMount struct {
	HostPath      string `json:"hostPath"`
	ContainerPath string `json:"containerPath"`
}

type applePublishedPort struct {
	HostPort      uint16 `json:"hostPort"`
	ContainerPort uint16 `json:"containerPort"`
	Protocol      string `json:"proto"`
}

type appleNetworkConfig struct {
	Network string `json:"network"`
}

type appleContainerStatus struct {
	State       string         `json:"state"`
	Networks    []appleNetwork `json:"networks"`
	StartedDate string         `json:"startedDate"`
}

type appleNetwork struct {
	Network     string `json:"network"`
	Hostname    string `json:"hostname"`
	IPv4Address string `json:"ipv4Address"`
	IPv4Gateway string `json:"ipv4Gateway"`
}

// ---------------------------------------------------------------------------
// List implementation
// ---------------------------------------------------------------------------

func appleContainerList() []*model.ContainerMatch {
	ctx, cancel := context.WithTimeout(context.Background(), runtimeQueryTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, "container", "ls", "--format", "json", "--all").Output()
	if err != nil {
		return nil
	}

	var entries []appleContainerEntry
	if err := json.Unmarshal(out, &entries); err != nil {
		return nil
	}

	matches := make([]*model.ContainerMatch, 0, len(entries))
	for _, e := range entries {
		cfg := e.Configuration
		status := e.Status

		m := &model.ContainerMatch{
			Runtime:   "container",
			ID:        cfg.ID,
			Name:      cfg.ID, // Apple container uses ID as name
			Image:     cfg.Image.Reference,
			Command:   buildCommand(cfg.InitProcess),
			State:     status.State,
			Status:    status.State,
			CreatedAt: parseRFC3339(cfg.CreationDate),
			StartedAt: parseRFC3339(status.StartedDate),
			Networks:  buildNetworks(status.Networks),
			Mounts:    buildMounts(cfg.Mounts),
			Ports:     buildPorts(cfg.PublishedPorts),
		}
		matches = append(matches, m)
	}
	return matches
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func buildCommand(p appleProcessConfig) string {
	if p.Executable == "" {
		return ""
	}
	if len(p.Arguments) == 0 {
		return p.Executable
	}
	return p.Executable + " " + strings.Join(p.Arguments, " ")
}

func buildNetworks(nets []appleNetwork) string {
	parts := make([]string, 0, len(nets))
	for _, n := range nets {
		if n.IPv4Address != "" {
			parts = append(parts, n.Network+":"+n.IPv4Address)
		}
	}
	return strings.Join(parts, ", ")
}

func buildMounts(mounts []appleMount) string {
	parts := make([]string, 0, len(mounts))
	for _, m := range mounts {
		parts = append(parts, m.HostPath+":"+m.ContainerPath)
	}
	return strings.Join(parts, ", ")
}

func buildPorts(ports []applePublishedPort) string {
	parts := make([]string, 0, len(ports))
	for _, p := range ports {
		parts = append(parts, formatPort(p))
	}
	return strings.Join(parts, ", ")
}

func formatPort(p applePublishedPort) string {
	proto := p.Protocol
	if proto == "" {
		proto = "tcp"
	}
	return itoaU16(p.HostPort) + "->" + itoaU16(p.ContainerPort) + "/" + proto
}

func parseRFC3339(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func itoaU16(n uint16) string {
	return strconv.FormatUint(uint64(n), 10)
}
