package model

// HostInterface is a host network interface snapshot for the TUI Network tab.
// Populated from OS-specific tools: ipconfig (Windows), ip addr (Linux), or
// the portable Go net package as a fallback.
type HostInterface struct {
	Name        string
	Type        string
	State       string
	MTU         int
	MAC         string
	IPv4        []string
	IPv6        []string
	Gateway     []string
	DNS         []string
	Description string
	DHCP        string // "Yes" / "No" / ""
	IfIndex     int
}

// DockerNetworkEndpoint is a container attached to a Docker network.
type DockerNetworkEndpoint struct {
	Name       string
	ID         string
	IP         string
	MAC        string
	EndpointID string
	Status     string // short status text from docker ps when available
}

// DockerNetwork describes a Docker network and its endpoints.
type DockerNetwork struct {
	Name            string
	ID              string
	Driver          string
	Scope           string
	BridgeInterface string
	Subnet          string
	Gateway         string
	Containers      []DockerNetworkEndpoint
}

// NetworkSnapshot is the full payload returned by a Network-tab refresh.
type NetworkSnapshot struct {
	Networks     []DockerNetwork
	HostIfaces   []HostInterface
	VethByBridge map[string][]string // bridge name → veth interface names (Linux best-effort)
	DockerOK     bool
	Error        string
	HostSource   string // e.g. "ipconfig", "ip addr", "net" — for status hints
}
