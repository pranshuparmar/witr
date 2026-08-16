//go:build !linux && !windows

package proc

import (
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func listHostInterfaces() []model.HostInterface {
	return listHostInterfacesStdlib()
}

func listHostInterfacesWithSource() ([]model.HostInterface, string) {
	return listHostInterfacesStdlib(), "net"
}

func listVethMap() map[string]vethInfo {
	return map[string]vethInfo{}
}

func detectInterfaceType(name string) string {
	lower := strings.ToLower(name)
	switch {
	case lower == "lo" || strings.Contains(lower, "loopback") || strings.HasPrefix(lower, "lo"):
		return "loopback"
	case strings.HasPrefix(lower, "eth") || strings.HasPrefix(lower, "en") || strings.Contains(lower, "ethernet"):
		return "physical"
	case strings.HasPrefix(lower, "wl") || strings.Contains(lower, "wi-fi") || strings.Contains(lower, "wlan"):
		return "wireless"
	case strings.HasPrefix(lower, "veth"):
		return "veth"
	case strings.Contains(lower, "docker") || strings.HasPrefix(lower, "br-"):
		return "bridge"
	case strings.HasPrefix(lower, "wg"):
		return "wireguard"
	case strings.HasPrefix(lower, "tun") || strings.HasPrefix(lower, "tap"):
		return "tun/tap"
	default:
		return "unknown"
	}
}
