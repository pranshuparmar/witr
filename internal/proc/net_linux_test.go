//go:build linux

package proc

import (
	"encoding/hex"
	"fmt"
	"net"
	"testing"
)

func encodeProcNetTCP6(ip net.IP, port int) string {
	ip16 := ip.To16()
	if ip16 == nil {
		return ""
	}

	// /proc/net/tcp6 stores IPv6 as 4 LE 32-bit groups
	// parseAddr reverses bytes within each 4-byte group to decode
	// so we just inverse the transformation for our tests
	stored := make([]byte, 16)
	for i := 0; i < 4; i++ {
		stored[i*4+0] = ip16[i*4+3]
		stored[i*4+1] = ip16[i*4+2]
		stored[i*4+2] = ip16[i*4+1]
		stored[i*4+3] = ip16[i*4+0]
	}

	return hex.EncodeToString(stored) + ":" + fmt.Sprintf("%04X", port)
}

func TestParseAddr(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		ipv6     bool
		wantAddr string
		wantPort int
	}{
		{
			name:     "IPv4 localhost",
			raw:      "0100007F:0277",
			ipv6:     false,
			wantAddr: "127.0.0.1",
			wantPort: 631,
		},
		{
			name:     "IPv4 all interfaces",
			raw:      "00000000:0050",
			ipv6:     false,
			wantAddr: "0.0.0.0",
			wantPort: 80,
		},
		{
			name:     "IPv6 loopback ::1",
			raw:      "00000000000000000000000001000000:0277",
			ipv6:     true,
			wantAddr: "::1",
			wantPort: 631,
		},
		{
			name:     "IPv6 all interfaces ::",
			raw:      "00000000000000000000000000000000:01BB",
			ipv6:     true,
			wantAddr: "::",
			wantPort: 443,
		},
		{
			name:     "IPv6 link-local fe80::1",
			raw:      encodeProcNetTCP6(net.ParseIP("fe80::1"), 8080),
			ipv6:     true,
			wantAddr: "fe80::1",
			wantPort: 8080,
		},
		// Edge cases
		{
			name:     "Empty input",
			raw:      "",
			ipv6:     false,
			wantAddr: "",
			wantPort: 0,
		},
		{
			name:     "Missing colon separator",
			raw:      "0100007F0277",
			ipv6:     false,
			wantAddr: "",
			wantPort: 0,
		},
		{
			name:     "Invalid hex in IPv4",
			raw:      "ZZZZZZZZ:0050",
			ipv6:     false,
			wantAddr: "",
			wantPort: 80,
		},
		{
			name:     "Invalid hex in IPv6",
			raw:      "ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ:0050",
			ipv6:     true,
			wantAddr: "",
			wantPort: 80,
		},
		{
			name:     "Wrong length IPv6 (too short)",
			raw:      "0000000000000000:0277",
			ipv6:     true,
			wantAddr: "::",
			wantPort: 631,
		},
		{
			name:     "Wrong length IPv4 (too short)",
			raw:      "01007F:0277",
			ipv6:     false,
			wantAddr: "",
			wantPort: 631,
		},
		{
			name:     "Only colon",
			raw:      ":",
			ipv6:     false,
			wantAddr: "",
			wantPort: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAddr, gotPort := parseAddr(tt.raw, tt.ipv6)
			if gotAddr != tt.wantAddr {
				t.Errorf("parseAddr() gotAddr = %v, want %v", gotAddr, tt.wantAddr)
			}
			if gotPort != tt.wantPort {
				t.Errorf("parseAddr() gotPort = %v, want %v", gotPort, tt.wantPort)
			}
		})

	}
}
