//go:build linux

package proc

import (
	"net"
	"testing"
)

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
			name:     "IPv6 link-local",
			raw:      "000080FE00000000FF005450EDA1FFFE:1F90",
			ipv6:     true,
			wantAddr: "",
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
			if tt.wantAddr == "" && tt.name == "IPv6 link-local" {
				ip := net.ParseIP(gotAddr)
				if ip == nil || ip.To16() == nil || ip.To4() != nil {
					t.Errorf("parseAddr() gotAddr = %v, want a valid IPv6 address", gotAddr)
				}
			} else if tt.wantAddr != "" && gotAddr != tt.wantAddr {
				t.Errorf("parseAddr() gotAddr = %v, want %v", gotAddr, tt.wantAddr)
			}
			if gotPort != tt.wantPort {
				t.Errorf("parseAddr() gotPort = %v, want %v", gotPort, tt.wantPort)
			}
		})
	}
}
