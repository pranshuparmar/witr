package proc

import "testing"

func TestParseOpenPortsNormalizesLocalizedListenerState(t *testing.T) {
	// German netstat emits ABHÖREN in the active console code page. The raw
	// 0x99 byte below is how Ö is represented in CP850 and is invalid UTF-8,
	// matching the garbled ABH?REN state reported in #227.
	out := append([]byte("TCP  0.0.0.0:8080  0.0.0.0:0  ABH"), 0x99)
	out = append(out, []byte("REN  4242\r\n")...)

	ports := parseOpenPorts(out)
	if len(ports) != 1 {
		t.Fatalf("parseOpenPorts() returned %d ports, want 1", len(ports))
	}
	if ports[0].State != "LISTEN" {
		t.Fatalf("listener state = %q, want LISTEN", ports[0].State)
	}
}
