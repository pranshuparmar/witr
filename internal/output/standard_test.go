package output

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestDisplayState(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in, want string
	}{
		{"LISTEN", "LISTENING"},
		{"ESTABLISHED", "ESTABLISHED"},
		{"CLOSE_WAIT", "CLOSE_WAIT"},
		{"OPEN", "OPEN"},
		{"", "?"},
	}

	for _, tt := range tests {
		if got := displayState(tt.in); got != tt.want {
			t.Errorf("displayState(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSocketSortRank(t *testing.T) {
	t.Parallel()

	// LISTEN must always rank above ESTABLISHED, and both above unknown.
	// OPEN sits between LISTEN and ESTABLISHED.
	if socketSortRank("LISTEN") >= socketSortRank("ESTABLISHED") {
		t.Errorf("LISTEN should rank below ESTABLISHED")
	}
	if socketSortRank("OPEN") <= socketSortRank("LISTEN") {
		t.Errorf("OPEN should rank above LISTEN")
	}
	if socketSortRank("ESTABLISHED") >= socketSortRank("TIME_WAIT") {
		t.Errorf("ESTABLISHED should rank above unknown states")
	}
	if socketSortRank("LISTEN") != 0 {
		t.Errorf("LISTEN should be rank 0 so it appears first")
	}
}

func TestFormatSocket(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		s    model.Socket
		want string
	}{
		{
			name: "tcp listener",
			s:    model.Socket{Address: "0.0.0.0", Port: 443, Protocol: "TCP", State: "LISTEN"},
			want: "0.0.0.0:443 (TCP | LISTENING)",
		},
		{
			name: "established",
			s:    model.Socket{Address: "127.0.0.1", Port: 43525, Protocol: "TCP", State: "ESTABLISHED"},
			want: "127.0.0.1:43525 (TCP | ESTABLISHED)",
		},
		{
			name: "ipv6 wraps host:port via JoinHostPort",
			s:    model.Socket{Address: "::1", Port: 8080, Protocol: "TCP6", State: "LISTEN"},
			want: "[::1]:8080 (TCP6 | LISTENING)",
		},
		{
			name: "udp open",
			s:    model.Socket{Address: "0.0.0.0", Port: 53, Protocol: "UDP", State: "OPEN"},
			want: "0.0.0.0:53 (UDP | OPEN)",
		},
		{
			name: "missing protocol falls back to ?",
			s:    model.Socket{Address: "127.0.0.1", Port: 9999, Protocol: "", State: "LISTEN"},
			want: "127.0.0.1:9999 (? | LISTENING)",
		},
		{
			name: "blank state renders as ?",
			s:    model.Socket{Address: "127.0.0.1", Port: 9999, Protocol: "TCP", State: ""},
			want: "127.0.0.1:9999 (TCP | ?)",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatSocket(socketRow{Socket: tt.s}); got != tt.want {
				t.Errorf("formatSocket(%+v) = %q, want %q", tt.s, got, tt.want)
			}
		})
	}
}

// A connection shows its remote end; a listener shows how many connections
// it accepted, and those connections don't get rows of their own.
func TestSocketRows(t *testing.T) {
	t.Parallel()

	in := []model.Socket{
		{Address: "192.168.1.10", Port: 51744, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "10.0.4.20", RemotePort: 5432},
		{Address: "127.0.0.1", Port: 5000, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "127.0.0.1", RemotePort: 40001},
		{Address: "::", Port: 5000, Protocol: "TCP6", State: "LISTEN"},
		{Address: "127.0.0.1", Port: 5000, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "127.0.0.1", RemotePort: 40002},
		{Address: "127.0.0.1", Port: 5000, Protocol: "TCP", State: "CLOSE_WAIT", RemoteAddress: "127.0.0.1", RemotePort: 40003},
		{Address: "2001:db8::10", Port: 51800, Protocol: "TCP6", State: "ESTABLISHED", RemoteAddress: "2001:db8::20", RemotePort: 443},
	}
	var got []string
	for _, r := range socketRows(in) {
		got = append(got, formatSocket(r))
	}
	want := []string{
		"[::]:5000 (TCP6 | LISTENING, 2 connections)",
		"192.168.1.10:51744 → 10.0.4.20:5432 (TCP | ESTABLISHED)",
		"[2001:db8::10]:51800 → [2001:db8::20]:443 (TCP6 | ESTABLISHED)",
		"127.0.0.1:5000 → 127.0.0.1:40003 (TCP | CLOSE_WAIT)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("socket rows:\n got: %q\nwant: %q", got, want)
	}
}

// A connection between two of the process's own sockets shows once, before
// any listener folding: here the process also connects to its own listener.
func TestSocketRowsInternalPairs(t *testing.T) {
	t.Parallel()

	in := []model.Socket{
		{Address: "127.0.0.1", Port: 49710, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "127.0.0.1", RemotePort: 49709},
		{Address: "127.0.0.1", Port: 49709, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "127.0.0.1", RemotePort: 49710},
		{Address: "0.0.0.0", Port: 3306, Protocol: "TCP", State: "LISTEN"},
		{Address: "127.0.0.1", Port: 3306, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "127.0.0.1", RemotePort: 50001},
		{Address: "127.0.0.1", Port: 50001, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "127.0.0.1", RemotePort: 3306},
		{Address: "127.0.0.1", Port: 3306, Protocol: "TCP", State: "ESTABLISHED", RemoteAddress: "127.0.0.1", RemotePort: 50100},
	}
	var got []string
	for _, r := range socketRows(in) {
		got = append(got, formatSocket(r))
	}
	want := []string{
		"0.0.0.0:3306 (TCP | LISTENING, 1 connection)",
		"127.0.0.1:3306 ↔ 127.0.0.1:50001 (TCP | ESTABLISHED, within the process)",
		"127.0.0.1:49709 ↔ 127.0.0.1:49710 (TCP | ESTABLISHED, within the process)",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("socket rows:\n got: %q\nwant: %q", got, want)
	}
}

func TestVisibleSocketsDropsIncomplete(t *testing.T) {
	t.Parallel()

	in := []model.Socket{
		{Address: "127.0.0.1", Port: 8080, State: "LISTEN"},
		{Address: "", Port: 80, State: "LISTEN"},         // missing address
		{Address: "127.0.0.1", Port: 0, State: "LISTEN"}, // missing port
		{Address: "0.0.0.0", Port: 443, State: "LISTEN"},
	}
	got := visibleSockets(in)
	if len(got) != 2 {
		t.Fatalf("visibleSockets dropped wrong number: got %d, want 2", len(got))
	}
	if got[0].Port != 8080 || got[1].Port != 443 {
		t.Errorf("visibleSockets unexpected order/content: %+v", got)
	}
}

func TestSortSockets(t *testing.T) {
	t.Parallel()

	// Scenario lifted from the real bug report: address grouping with a
	// LISTEN that must sit above its ESTABLISHED sibling on the same
	// address:port, and ports ascending within each address.
	in := []model.Socket{
		{Address: "192.168.176.122", Port: 58236, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36146, State: "ESTABLISHED"},
		{Address: "127.0.0.1", Port: 43525, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36170, State: "ESTABLISHED"},
		{Address: "127.0.0.1", Port: 43525, State: "LISTEN"},
		{Address: "192.168.176.122", Port: 36162, State: "ESTABLISHED"},
	}
	want := []model.Socket{
		{Address: "127.0.0.1", Port: 43525, State: "LISTEN"},
		{Address: "127.0.0.1", Port: 43525, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36146, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36162, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 36170, State: "ESTABLISHED"},
		{Address: "192.168.176.122", Port: 58236, State: "ESTABLISHED"},
	}

	sortSockets(in)
	if !reflect.DeepEqual(in, want) {
		t.Errorf("sortSockets order mismatch.\n got: %+v\nwant: %+v", in, want)
	}
}

func TestSortSocketsListenBeforeEstablishedSamePort(t *testing.T) {
	t.Parallel()

	// Insertion order is ESTABLISHED first, but the sort must surface
	// LISTEN above it.
	in := []model.Socket{
		{Address: "10.0.0.1", Port: 80, State: "ESTABLISHED"},
		{Address: "10.0.0.1", Port: 80, State: "LISTEN"},
	}
	sortSockets(in)
	if in[0].State != "LISTEN" {
		t.Errorf("LISTEN must sort above ESTABLISHED on same address:port; got %+v", in)
	}
}

func TestSortSocketsStableWithinSameRank(t *testing.T) {
	t.Parallel()

	// Two ESTABLISHED sockets on different ports of the same address should
	// be ordered by port. Stability is asserted via the deterministic input
	// ordering: lower port first regardless of input position.
	in := []model.Socket{
		{Address: "10.0.0.1", Port: 9000, State: "ESTABLISHED"},
		{Address: "10.0.0.1", Port: 8000, State: "ESTABLISHED"},
	}
	sortSockets(in)
	if in[0].Port != 8000 {
		t.Errorf("expected lower port first within same address; got %+v", in)
	}
}

// A Windows integrity level other than Medium and a confining Linux security
// label show on the report; unconfined processes say nothing.
func TestIntegrityAndSecurityLabel(t *testing.T) {
	t.Parallel()
	for level, want := range map[string]string{"": "", "Medium": "", "High": " (elevated)", "System": " (system integrity)", "Low": " (low integrity)"} {
		if got := integrityNote(level); got != want {
			t.Errorf("integrityNote(%q) = %q, want %q", level, got, want)
		}
	}
	tests := []struct {
		module, label, want string
	}{
		{"AppArmor", "/usr/sbin/cupsd (enforce)", "AppArmor /usr/sbin/cupsd (enforce)"},
		{"AppArmor", "unconfined", ""},
		{"SELinux", "system_u:system_r:httpd_t:s0", "SELinux system_u:system_r:httpd_t:s0"},
		{"SELinux", "unconfined_u:unconfined_r:unconfined_t:s0-s0:c0.c1023", ""},
		{"", "", ""},
		{"", "system_u:system_r:httpd_t:s0", "system_u:system_r:httpd_t:s0"},
	}
	for _, tt := range tests {
		if got := securityLabel(model.Process{SecurityModule: tt.module, SecurityLabel: tt.label}); got != tt.want {
			t.Errorf("securityLabel(%q, %q) = %q, want %q", tt.module, tt.label, got, tt.want)
		}
	}
	var b bytes.Buffer
	httpd := model.Process{PID: 7, Command: "httpd", User: "apache", SecurityModule: "SELinux", SecurityLabel: "system_u:system_r:httpd_t:s0", IntegrityLevel: "High"}
	RenderStandard(&b, model.Result{Process: httpd, Ancestry: []model.Process{httpd}}, false, false)
	for _, want := range []string{"User        : apache (elevated)", "Security    : SELinux system_u:system_r:httpd_t:s0"} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("report missing %q:\n%s", want, b.String())
		}
	}
}

// renderPlain renders the standard report without color.
func renderPlain(r model.Result) string {
	var buf bytes.Buffer
	RenderStandard(&buf, r, false, false)
	return buf.String()
}

// A start time witr couldn't read (a protected Windows service, say) shows as
// plain "unknown", not "unknown ()".
func TestRenderStandardStarted(t *testing.T) {
	t.Parallel()

	unknown := renderPlain(model.Result{Ancestry: []model.Process{{PID: 7, Command: "svchost.exe"}}})
	if !strings.Contains(unknown, "Started     : unknown\n") {
		t.Errorf("unknown start time:\n%s", unknown)
	}
	known := renderPlain(model.Result{Ancestry: []model.Process{{PID: 7, Command: "nginx", StartedAt: time.Now().Add(-3 * time.Hour)}}})
	if !strings.Contains(known, "Started     : 3 hours ago (") {
		t.Errorf("known start time:\n%s", known)
	}
}

// The Process line carries the health and fork tags; the source's file gets
// the label its manager uses.
func TestRenderStandardTagsAndSourceFile(t *testing.T) {
	t.Parallel()

	got := renderPlain(model.Result{Ancestry: []model.Process{{PID: 5, Command: "worker", Health: "zombie", Forked: "forked"}}})
	if !strings.Contains(got, "Process     : worker (pid 5) [zombie] {forked}\n") {
		t.Errorf("health and fork tags:\n%s", got)
	}
	got = renderPlain(model.Result{Ancestry: []model.Process{{PID: 5, Command: "worker", Health: "healthy", Forked: "not-forked"}}})
	if !strings.Contains(got, "Process     : worker (pid 5)\n") {
		t.Errorf("a healthy, unforked process has no tags:\n%s", got)
	}

	tests := []struct {
		typ  model.SourceType
		want string
	}{
		{model.SourceSystemd, "Unit File   : /etc/x\n"},
		{model.SourceLaunchd, "Plist File  : /etc/x\n"},
		{model.SourceWindowsService, "Registry Key : /etc/x\n"},
		{model.SourceBsdRc, "Rc Script   : /etc/x\n"},
	}
	for _, tt := range tests {
		r := model.Result{
			Ancestry: []model.Process{{PID: 5, Command: "worker"}},
			Source:   model.Source{Type: tt.typ, Name: "worker", UnitFile: "/etc/x"},
		}
		if got := renderPlain(r); !strings.Contains(got, tt.want) {
			t.Errorf("%s source file line: want %q in\n%s", tt.typ, tt.want, got)
		}
	}

	r := model.Result{
		Ancestry: []model.Process{{PID: 5, Command: "worker"}},
		Source:   model.Source{Type: model.SourceSystemd, Name: "worker", UnitFile: "/etc/\x1b[31mx"},
	}
	if got := renderPlain(r); strings.Contains(got, "\x1b") {
		t.Errorf("unit file path not sanitized:\n%q", got)
	}
}

func TestContainerStateTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state, health, want string
	}{
		{"running", "", ""},
		{"running", "healthy", "healthy"},
		{"running", "unhealthy", "unhealthy"},
		{"running", "starting", "starting"},
		{"exited", "", "exited"},
		{"Restarting", "", "restarting"},
		{"exited", "unhealthy", "unhealthy"},
	}
	for _, tt := range tests {
		if got := containerStateTag(&model.ContainerMatch{State: tt.state, Health: tt.health}); got != tt.want {
			t.Errorf("containerStateTag(%q, %q) = %q, want %q", tt.state, tt.health, got, tt.want)
		}
	}
}
