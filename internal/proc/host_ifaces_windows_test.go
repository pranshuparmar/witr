//go:build windows

package proc

import (
	"strings"
	"testing"
)

const sampleIPConfig = `
Windows IP Configuration

   Host Name . . . . . . . . . . . . : DESKTOP-TEST

Ethernet adapter Ethernet:

   Connection-specific DNS Suffix  . :
   Description . . . . . . . . . . . : Realtek PCIe GbE Family Controller
   Physical Address. . . . . . . . . : AA-BB-CC-DD-EE-FF
   DHCP Enabled. . . . . . . . . . . : Yes
   IPv4 Address. . . . . . . . . . . : 192.168.3.164(Preferred)
   Subnet Mask . . . . . . . . . . . : 255.255.255.0
   Default Gateway . . . . . . . . . : 192.168.3.1
   DNS Servers . . . . . . . . . . . : 1.1.1.1
                                       8.8.8.8

Wireless LAN adapter Wi-Fi:

   Media State . . . . . . . . . . . : Media disconnected
   Description . . . . . . . . . . . : Intel Wi-Fi
   Physical Address. . . . . . . . . : 11-22-33-44-55-66
   DHCP Enabled. . . . . . . . . . . : Yes

Ethernet adapter vEthernet (WSL):

   Connection-specific DNS Suffix  . :
   Description . . . . . . . . . . . : Hyper-V Virtual Ethernet Adapter
   Physical Address. . . . . . . . . : 00-15-5D-01-02-03
   DHCP Enabled. . . . . . . . . . . : No
   IPv4 Address. . . . . . . . . . . : 172.24.192.1(Preferred)
   Subnet Mask . . . . . . . . . . . : 255.255.240.0
   Default Gateway . . . . . . . . . :
`

func TestParseIPConfig(t *testing.T) {
	ifaces := parseIPConfig(sampleIPConfig)
	if len(ifaces) < 3 {
		t.Fatalf("got %d adapters, want >= 3", len(ifaces))
	}
	var eth *struct {
		name string
	}
	_ = eth
	foundEth := false
	for _, iface := range ifaces {
		if iface.Name == "Ethernet" {
			foundEth = true
			if iface.Type != "physical" {
				t.Errorf("Ethernet type = %q, want physical", iface.Type)
			}
			if iface.State != "UP" {
				t.Errorf("Ethernet state = %q, want UP", iface.State)
			}
			if len(iface.IPv4) == 0 || !strings.HasPrefix(iface.IPv4[0], "192.168.3.164") {
				t.Errorf("Ethernet IPv4 = %v", iface.IPv4)
			}
			if !strings.Contains(iface.IPv4[0], "/24") {
				t.Errorf("expected /24 prefix on IPv4, got %v", iface.IPv4)
			}
			if len(iface.Gateway) == 0 || iface.Gateway[0] != "192.168.3.1" {
				t.Errorf("Gateway = %v", iface.Gateway)
			}
			if iface.MAC != "aa:bb:cc:dd:ee:ff" {
				t.Errorf("MAC = %q", iface.MAC)
			}
			if len(iface.DNS) < 2 {
				t.Errorf("DNS = %v, want 1.1.1.1 and 8.8.8.8", iface.DNS)
			}
		}
		if iface.Name == "Wi-Fi" && iface.State != "DOWN" {
			t.Errorf("Wi-Fi state = %q, want DOWN", iface.State)
		}
		if strings.Contains(iface.Name, "WSL") && iface.Type != "wsl" && iface.Type != "hyper-v" {
			// vEthernet (WSL) classified as hyper-v or wsl
			if iface.Type == "unknown" {
				t.Errorf("WSL adapter type unknown: %+v", iface)
			}
		}
	}
	if !foundEth {
		t.Fatal("Ethernet adapter not found")
	}
}

func TestListHostInterfacesIPConfigLive(t *testing.T) {
	ifaces, src := listHostInterfacesWithSource()
	if len(ifaces) == 0 {
		t.Fatal("expected at least one host interface on Windows")
	}
	if src == "" {
		t.Error("HostSource should be set")
	}
	t.Logf("source=%s count=%d first=%q", src, len(ifaces), ifaces[0].Name)
}
