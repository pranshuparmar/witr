package proc

import (
	"encoding/json"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestParseDockerNetworkInspect(t *testing.T) {
	const raw = `[
  {
    "Name": "bridge",
    "Id": "abc123def456789",
    "Driver": "bridge",
    "Scope": "local",
    "Options": {
      "com.docker.network.bridge.name": "docker0"
    },
    "IPAM": {
      "Config": [
        {
          "Subnet": "172.17.0.0/16",
          "Gateway": "172.17.0.1"
        }
      ]
    },
    "Containers": {
      "0123456789abcdef0123456789abcdef": {
        "Name": "web",
        "IPv4Address": "172.17.0.2/16",
        "MacAddress": "02:42:ac:11:00:02",
        "EndpointID": "endpoint1234567890"
      }
    }
  }
]`
	var data []map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &data); err != nil {
		t.Fatal(err)
	}
	dn := model.DockerNetwork{Name: "bridge", ID: "abc123def456"}
	parseDockerNetworkInspect(&dn, data[0])

	if dn.BridgeInterface != "docker0" {
		t.Errorf("BridgeInterface = %q, want docker0", dn.BridgeInterface)
	}
	if dn.Subnet != "172.17.0.0/16" {
		t.Errorf("Subnet = %q", dn.Subnet)
	}
	if dn.Gateway != "172.17.0.1" {
		t.Errorf("Gateway = %q", dn.Gateway)
	}
	if len(dn.Containers) != 1 {
		t.Fatalf("Containers len = %d", len(dn.Containers))
	}
	c := dn.Containers[0]
	if c.Name != "web" || c.IP != "172.17.0.2/16" {
		t.Errorf("endpoint = %+v", c)
	}
}

func TestDetectInterfaceType(t *testing.T) {
	cases := map[string]string{
		"eth0":    "physical",
		"enp0s3":  "physical",
		"wlan0":   "wireless",
		"docker0": "bridge",
		"br-abc":  "bridge",
		"veth123": "veth",
		"lo":      "loopback",
		"wg0":     "wireguard",
		"tun0":    "tun/tap",
	}
	for name, want := range cases {
		if got := detectInterfaceType(name); got != want {
			t.Errorf("detectInterfaceType(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestListNetworkSnapshotNoPanic(t *testing.T) {
	// Must not panic whether or not Docker is installed.
	snap := ListNetworkSnapshot()
	if snap.VethByBridge == nil {
		t.Error("VethByBridge should be non-nil map")
	}
	// Host ifaces should usually be non-empty on a real machine.
	if snap.HostIfaces == nil {
		t.Error("HostIfaces should not be nil")
	}
}
