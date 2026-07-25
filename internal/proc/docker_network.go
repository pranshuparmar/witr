package proc

import (
	"context"
	"encoding/json"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ListNetworkSnapshot gathers Docker networks, host interfaces, and (when
// available) veth→bridge mapping for the TUI Network tab. Best-effort: missing
// docker or elevated rights never returns an error to the caller.
func ListNetworkSnapshot() model.NetworkSnapshot {
	snap := model.NetworkSnapshot{
		VethByBridge: make(map[string][]string),
	}

	// OS-specific host adapters: ipconfig (Windows), ip addr (Linux), net fallback.
	snap.HostIfaces, snap.HostSource = listHostInterfacesWithSource()
	vethMap := listVethMap()
	for name, info := range vethMap {
		b := info.Bridge
		if b == "" {
			b = "unconnected"
		}
		snap.VethByBridge[b] = append(snap.VethByBridge[b], name)
	}
	for b := range snap.VethByBridge {
		sort.Strings(snap.VethByBridge[b])
	}

	dockerBin := resolveDockerBinary()
	if dockerBin == "" {
		snap.Error = "Docker not available (install Docker or add docker to PATH)"
		return snap
	}

	networks, contStatus, ok := collectDockerNetworks(dockerBin)
	snap.DockerOK = ok
	snap.Networks = networks
	if !ok && snap.Error == "" {
		snap.Error = "Failed to query Docker networks (is the engine running?)"
	}

	// Attach short status text from docker ps -a onto endpoints.
	for i := range snap.Networks {
		for j := range snap.Networks[i].Containers {
			cname := snap.Networks[i].Containers[j].Name
			if st, found := contStatus[cname]; found {
				snap.Networks[i].Containers[j].Status = st
			}
		}
	}

	return snap
}

// resolveDockerBinary returns a docker CLI path. LookPath first, then common
// Docker Desktop install locations (Windows terminals often lack the PATH entry).
func resolveDockerBinary() string {
	if binAvailable("docker") {
		return "docker"
	}
	return dockerBinaryCandidates()
}

type vethInfo struct {
	PeerIf  string
	Bridge  string
	NetnsID string
}

func collectDockerNetworks(dockerBin string) ([]model.DockerNetwork, map[string]string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	lsOut, err := exec.CommandContext(ctx, dockerBin, "network", "ls",
		"--format", "{{.Name}}|{{.ID}}|{{.Driver}}|{{.Scope}}").Output()
	if err != nil {
		return nil, nil, false
	}

	contStatus := dockerContainerStatusMap(ctx, dockerBin)

	var networks []model.DockerNetwork
	for _, line := range strings.Split(strings.TrimSpace(string(lsOut)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 4 {
			continue
		}
		name := parts[0]
		netID := parts[1]
		if len(netID) > 12 {
			netID = netID[:12]
		}
		dn := model.DockerNetwork{
			Name:   name,
			ID:     netID,
			Driver: parts[2],
			Scope:  parts[3],
		}
		enrichDockerNetwork(ctx, dockerBin, &dn)
		networks = append(networks, dn)
	}

	sort.Slice(networks, func(i, j int) bool {
		return networks[i].Name < networks[j].Name
	})
	return networks, contStatus, true
}

func dockerContainerStatusMap(ctx context.Context, dockerBin string) map[string]string {
	out := make(map[string]string)
	raw, err := exec.CommandContext(ctx, dockerBin, "ps", "-a",
		"--format", "{{.Names}}|{{.Status}}").Output()
	if err != nil {
		return out
	}
	for _, line := range strings.Split(strings.TrimSpace(string(raw)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) < 2 {
			continue
		}
		st := parts[1]
		// Compact status for table cells.
		switch {
		case strings.HasPrefix(st, "Up"):
			out[parts[0]] = "Up"
		case strings.HasPrefix(st, "Exited"):
			out[parts[0]] = "Exited"
		case strings.HasPrefix(st, "Created"):
			out[parts[0]] = "Created"
		default:
			if len(st) > 16 {
				st = st[:16]
			}
			out[parts[0]] = st
		}
	}
	return out
}

func enrichDockerNetwork(ctx context.Context, dockerBin string, dn *model.DockerNetwork) {
	raw, err := exec.CommandContext(ctx, dockerBin, "network", "inspect", dn.Name).Output()
	if err != nil {
		return
	}

	var data []map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil || len(data) == 0 {
		return
	}
	netData := data[0]
	parseDockerNetworkInspect(dn, netData)
}

// parseDockerNetworkInspect fills dn from one docker network inspect object.
// Exported for tests via the unexported name still package-visible.
func parseDockerNetworkInspect(dn *model.DockerNetwork, netData map[string]interface{}) {
	if opts, ok := netData["Options"].(map[string]interface{}); ok {
		if bname, ok := opts["com.docker.network.bridge.name"].(string); ok {
			dn.BridgeInterface = bname
		}
	}
	if dn.BridgeInterface == "" {
		switch dn.Name {
		case "bridge":
			dn.BridgeInterface = "docker0"
		case "host", "none":
			dn.BridgeInterface = dn.Name
		default:
			dn.BridgeInterface = "br-" + dn.ID
		}
	}

	if ipam, ok := netData["IPAM"].(map[string]interface{}); ok {
		if cfg, ok := ipam["Config"].([]interface{}); ok && len(cfg) > 0 {
			if c0, ok := cfg[0].(map[string]interface{}); ok {
				if s, ok := c0["Subnet"].(string); ok {
					dn.Subnet = s
				}
				if g, ok := c0["Gateway"].(string); ok {
					dn.Gateway = g
				}
			}
		}
	}
	if dn.Subnet == "" {
		dn.Subnet = "N/A"
	}
	if dn.Gateway == "" {
		dn.Gateway = "N/A"
	}

	if conts, ok := netData["Containers"].(map[string]interface{}); ok {
		for cid, cinfoRaw := range conts {
			cinfo, ok := cinfoRaw.(map[string]interface{})
			if !ok {
				continue
			}
			cname, _ := cinfo["Name"].(string)
			if cname == "" {
				cname = cid
				if len(cname) > 12 {
					cname = cname[:12]
				}
			}
			ip, _ := cinfo["IPv4Address"].(string)
			mac, _ := cinfo["MacAddress"].(string)
			epid, _ := cinfo["EndpointID"].(string)
			if len(epid) > 12 {
				epid = epid[:12]
			}
			shortID := cid
			if len(shortID) > 12 {
				shortID = shortID[:12]
			}
			dn.Containers = append(dn.Containers, model.DockerNetworkEndpoint{
				Name:       cname,
				ID:         shortID,
				IP:         ip,
				MAC:        mac,
				EndpointID: epid,
			})
		}
		sort.Slice(dn.Containers, func(i, j int) bool {
			return dn.Containers[i].Name < dn.Containers[j].Name
		})
	}
}
