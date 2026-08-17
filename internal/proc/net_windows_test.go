//go:build windows

package proc

import "testing"

func TestNormalizeWindowsTCPState(t *testing.T) {
	cases := []struct {
		name       string
		state      string
		remoteAddr string
		want       string
	}{
		{"english listener", "LISTENING", "0.0.0.0:0", "LISTEN"},
		{"german listener", "ABH\xd6REN", "0.0.0.0:0", "LISTEN"},
		{"french listener", "ECOUTE", "0.0.0.0:0", "LISTEN"},
		{"ipv6 listener", "LISTENING", "[::]:0", "LISTEN"},

		// A zero remote port is not unique to a listener, so these must keep
		// their own state rather than being reported as listeners.
		{"bound is not a listener", "BOUND", "0.0.0.0:0", "BOUND"},
		{"closed is not a listener", "CLOSED", "0.0.0.0:0", "CLOSED"},
		{"delete_tcb is not a listener", "DELETE_TCB", "0.0.0.0:0", "DELETE_TCB"},

		{"established passes through", "ESTABLISHED", "127.0.0.1:443", "ESTABLISHED"},
		{"time_wait passes through", "TIME_WAIT", "127.0.0.1:443", "TIME_WAIT"},

		// Unrecognized and with a real peer: keep it, but make it printable.
		{"unknown localized state with a peer", "\xc9TABLI", "127.0.0.1:443", "?TABLI"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeWindowsTCPState(tc.state, tc.remoteAddr); got != tc.want {
				t.Fatalf("normalizeWindowsTCPState(%q, %q) = %q, want %q",
					tc.state, tc.remoteAddr, got, tc.want)
			}
		})
	}
}
