package proc

import (
	"net"
	"sort"

	"github.com/pranshuparmar/witr/pkg/model"
)

// listHostInterfacesStdlib is the portable fallback using Go's net package.
func listHostInterfacesStdlib() []model.HostInterface {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	var result []model.HostInterface
	for _, iface := range ifaces {
		state := "DOWN"
		if iface.Flags&net.FlagUp != 0 {
			state = "UP"
		}
		var ipv4, ipv6 []string
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			s := a.String()
			if ipnet, ok := a.(*net.IPNet); ok {
				if ip4 := ipnet.IP.To4(); ip4 != nil {
					ipv4 = append(ipv4, s)
				} else if ipnet.IP.To16() != nil && !ipnet.IP.IsLinkLocalUnicast() {
					ipv6 = append(ipv6, s)
				}
			} else {
				ipv4 = append(ipv4, s)
			}
		}
		mac := iface.HardwareAddr.String()
		if mac == "" {
			mac = "N/A"
		}
		mtu := iface.MTU
		if mtu < 0 {
			mtu = 0 // Windows loopback reports MTU=-1; treat as unknown.
		}
		result = append(result, model.HostInterface{
			Name:    iface.Name,
			Type:    detectInterfaceType(iface.Name),
			State:   state,
			MTU:     mtu,
			MAC:     mac,
			IPv4:    ipv4,
			IPv6:    ipv6,
			IfIndex: iface.Index,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}
