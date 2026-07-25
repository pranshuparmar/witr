//go:build linux

package proc

import (
	"encoding/json"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

type ipAddrJSON []struct {
	IfIndex   int    `json:"ifindex"`
	IfName    string `json:"ifname"`
	OperState string `json:"operstate"`
	MTU       int    `json:"mtu"`
	Address   string `json:"address"`
	AddrInfo  []struct {
		Family    string `json:"family"`
		Local     string `json:"local"`
		PrefixLen int    `json:"prefixlen"`
		Scope     string `json:"scope"`
	} `json:"addr_info"`
}

func listHostInterfaces() []model.HostInterface {
	ifaces, _ := listHostInterfacesWithSource()
	return ifaces
}

func listHostInterfacesWithSource() ([]model.HostInterface, string) {
	// Prefer structured `ip -j addr show`, then plain `ip addr show`, then stdlib.
	out, err := exec.Command("ip", "-j", "addr", "show").Output()
	if err == nil && len(out) > 0 {
		var data ipAddrJSON
		if err := json.Unmarshal(out, &data); err == nil && len(data) > 0 {
			return hostIfacesFromIPJSON(data), "ip -j addr show"
		}
	}
	if text := fallbackHostInterfaces(); len(text) > 0 {
		return text, "ip addr show"
	}
	return listHostInterfacesStdlib(), "net"
}

func hostIfacesFromIPJSON(data ipAddrJSON) []model.HostInterface {
	var ifaces []model.HostInterface
	for _, iface := range data {
		name := iface.IfName
		if name == "" {
			continue
		}
		state := strings.ToUpper(iface.OperState)
		if state == "" {
			state = "UNKNOWN"
		}
		var ipv4, ipv6 []string
		for _, a := range iface.AddrInfo {
			if a.Family == "inet" && a.Local != "" {
				ipv4 = append(ipv4, fmtPrefix(a.Local, a.PrefixLen))
			}
			if a.Family == "inet6" && a.Local != "" && !strings.HasPrefix(a.Local, "fe80") {
				ipv6 = append(ipv6, fmtPrefix(a.Local, a.PrefixLen))
			}
		}
		ifaces = append(ifaces, model.HostInterface{
			Name:    name,
			Type:    detectInterfaceType(name),
			State:   state,
			MTU:     iface.MTU,
			MAC:     iface.Address,
			IPv4:    ipv4,
			IPv6:    ipv6,
			IfIndex: iface.IfIndex,
		})
	}
	sort.Slice(ifaces, func(i, j int) bool { return ifaces[i].Name < ifaces[j].Name })
	return ifaces
}

func fmtPrefix(local string, prefix int) string {
	if prefix > 0 {
		return local + "/" + itoa(prefix)
	}
	return local
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func fallbackHostInterfaces() []model.HostInterface {
	out, err := exec.Command("ip", "addr", "show").Output()
	if err != nil {
		return listHostInterfacesStdlib()
	}
	return parseIPAddrText(string(out))
}

func parseIPAddrText(out string) []model.HostInterface {
	ifaces := make(map[string]model.HostInterface)
	var order []string
	var current string

	reIface := regexp.MustCompile(`^(\d+):\s+(\S+):`)
	reMAC := regexp.MustCompile(`link/ether\s+(\S+)`)
	reIPv4 := regexp.MustCompile(`inet\s+(\S+)`)
	reIPv6 := regexp.MustCompile(`inet6\s+(\S+)\s+scope\s+global`)

	for _, line := range strings.Split(out, "\n") {
		if m := reIface.FindStringSubmatch(line); m != nil {
			current = strings.Split(m[2], "@")[0]
			state := "UNKNOWN"
			if strings.Contains(line, "UP") {
				state = "UP"
			} else if strings.Contains(line, "DOWN") {
				state = "DOWN"
			}
			ifaces[current] = model.HostInterface{
				Name:  current,
				Type:  detectInterfaceType(current),
				State: state,
				MAC:   "N/A",
			}
			order = append(order, current)
			continue
		}
		if current == "" {
			continue
		}
		ifi := ifaces[current]
		if m := reMAC.FindStringSubmatch(line); m != nil {
			ifi.MAC = m[1]
		}
		if m := reIPv4.FindStringSubmatch(line); m != nil {
			ifi.IPv4 = append(ifi.IPv4, m[1])
		}
		if m := reIPv6.FindStringSubmatch(line); m != nil {
			ifi.IPv6 = append(ifi.IPv6, m[1])
		}
		ifaces[current] = ifi
	}

	result := make([]model.HostInterface, 0, len(order))
	for _, n := range order {
		result = append(result, ifaces[n])
	}
	return result
}

func listVethMap() map[string]vethInfo {
	vethMap := make(map[string]vethInfo)
	out, err := exec.Command("ip", "link", "show").Output()
	if err != nil {
		return vethMap
	}

	reVeth := regexp.MustCompile(`^(\d+):\s+(veth[^:@]+)@if(\d+):`)
	reMaster := regexp.MustCompile(`master\s+(\S+)`)
	reNetns := regexp.MustCompile(`link-netnsid\s+(\d+)`)

	var current string
	for _, line := range strings.Split(string(out), "\n") {
		if m := reVeth.FindStringSubmatch(line); m != nil {
			current = m[2]
			vethMap[current] = vethInfo{PeerIf: m[3]}
			continue
		}
		if current == "" {
			continue
		}
		vi := vethMap[current]
		if m := reMaster.FindStringSubmatch(line); m != nil {
			vi.Bridge = m[1]
		}
		if m := reNetns.FindStringSubmatch(line); m != nil {
			vi.NetnsID = m[1]
			vethMap[current] = vi
			current = ""
			continue
		}
		vethMap[current] = vi
	}
	return vethMap
}

func detectInterfaceType(name string) string {
	switch {
	case strings.HasPrefix(name, "eth") || strings.HasPrefix(name, "en"):
		return "physical"
	case strings.HasPrefix(name, "wl"):
		return "wireless"
	case strings.HasPrefix(name, "br-") || name == "docker0" || strings.HasPrefix(name, "br"):
		return "bridge"
	case strings.HasPrefix(name, "veth"):
		return "veth"
	case strings.HasPrefix(name, "wg"):
		return "wireguard"
	case strings.HasPrefix(name, "tun") || strings.HasPrefix(name, "tap"):
		return "tun/tap"
	case name == "lo":
		return "loopback"
	case strings.HasPrefix(name, "virbr") || strings.HasPrefix(name, "vnet"):
		return "libvirt"
	case strings.Contains(strings.ToLower(name), "docker"):
		return "docker"
	default:
		return "unknown"
	}
}
