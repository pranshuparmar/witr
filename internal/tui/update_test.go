package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuparmar/witr/pkg/model"
)

// keyRunes builds a rune key message (e.g. "p", "/", "a") the way bubbletea
// delivers ordinary character presses.
func keyRunes(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

// step runs one Update and returns the concrete model, failing if the model
// type ever changes out from under us.
func step(t *testing.T, m MainModel, msg tea.Msg) (MainModel, tea.Cmd) {
	t.Helper()
	nm, cmd := m.Update(msg)
	mm, ok := nm.(MainModel)
	if !ok {
		t.Fatalf("Update returned %T, want MainModel", nm)
	}
	return mm, cmd
}

func TestUpdateQuitKeys(t *testing.T) {
	for _, key := range []tea.KeyMsg{{Type: tea.KeyCtrlC}, keyRunes("q"), {Type: tea.KeyEsc}} {
		m, cmd := step(t, InitialModel("test"), key)
		if !m.quitting {
			t.Errorf("%s should set quitting", key)
		}
		if cmd == nil {
			t.Errorf("%s should return a quit command", key)
		}
	}
}

func TestUpdateTabSwitch(t *testing.T) {
	tests := []struct {
		key  string
		want tab
	}{
		{"2", tabPorts},
		{"3", tabContainers},
		{"1", tabProcesses},
		{"5", tabNetwork},
	}
	for _, tt := range tests {
		// Start on a different tab so the switch is observable.
		m := InitialModel("test")
		m.activeTab = tabLocks
		m, _ = step(t, m, keyRunes(tt.key))
		if m.activeTab != tt.want {
			t.Errorf("key %q: activeTab = %v, want %v", tt.key, m.activeTab, tt.want)
		}
	}
}

func TestUpdateToggleNetworkScope(t *testing.T) {
	m := InitialModel("test")
	m.activeTab = tabNetwork
	if m.networkScope != networkScopeAll {
		t.Fatalf("precondition: scope should start ALL, got %v", m.networkScope)
	}
	m, _ = step(t, m, keyRunes("a"))
	if m.networkScope != networkScopeHost {
		t.Errorf("first 'a' => HOST, got %v", m.networkScope)
	}
	m, _ = step(t, m, keyRunes("a"))
	if m.networkScope != networkScopeDocker {
		t.Errorf("second 'a' => DOCKER, got %v", m.networkScope)
	}
	m, _ = step(t, m, keyRunes("a"))
	if m.networkScope != networkScopeAll {
		t.Errorf("third 'a' => ALL, got %v", m.networkScope)
	}
}

func TestHandleNetworkSnapshotCombinesHostAndDocker(t *testing.T) {
	m := InitialModel("test")
	m.activeTab = tabNetwork
	m, _ = step(t, m, tea.WindowSizeMsg{Width: 140, Height: 40})
	snap := model.NetworkSnapshot{
		HostIfaces: []model.HostInterface{
			{Name: "Ethernet", Type: "physical", State: "UP", MTU: 1500, IPv4: []string{"10.0.0.2/24"}, Gateway: []string{"10.0.0.1"}, MAC: "aa:bb:cc:dd:ee:ff"},
			{Name: "Wi-Fi", Type: "wireless", State: "UP", IPv4: []string{"10.0.0.3/24"}},
		},
		Networks: []model.DockerNetwork{
			{Name: "bridge", Driver: "bridge", Subnet: "172.17.0.0/16", Gateway: "172.17.0.1", BridgeInterface: "docker0"},
		},
		DockerOK:   true,
		HostSource: "ipconfig /all",
	}
	m, _ = step(t, m, snap)
	if len(m.hostIfaces) != 2 {
		t.Fatalf("hostIfaces = %d", len(m.hostIfaces))
	}
	if len(m.networks) != 1 {
		t.Fatalf("networks = %d", len(m.networks))
	}
	// ALL scope should show host + docker.
	if got := len(m.networkTable.Rows()); got != 3 {
		t.Fatalf("ALL table rows = %d, want 3 (2 host + 1 docker)", got)
	}
	view := m.View()
	if !strings.Contains(view, "Ethernet") {
		t.Errorf("view missing Ethernet; snippet %q", truncateForTest(view, 240))
	}
	if !strings.Contains(view, "bridge") && !strings.Contains(view, "Docker") {
		t.Errorf("view missing docker network; snippet %q", truncateForTest(view, 240))
	}

	// HOST filter
	m.networkScope = networkScopeHost
	m.updateNetworkTable()
	if got := len(m.networkTable.Rows()); got != 2 {
		t.Errorf("HOST filter rows = %d, want 2", got)
	}
	// DOCKER filter
	m.networkScope = networkScopeDocker
	m.updateNetworkTable()
	if got := len(m.networkTable.Rows()); got != 1 {
		t.Errorf("DOCKER filter rows = %d, want 1", got)
	}
}

func truncateForTest(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

func TestUpdateEnterOpensProcessDetail(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 4242, Command: "x"}}
	m.filterProcesses() // gives the table a selectable row at cursor 0

	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateDetail {
		t.Fatalf("enter on a process row should open detail; state = %v", m.state)
	}
	if cmd == nil {
		t.Error("enter should kick off the detail fetch command")
	}
}

func TestUpdateFocusSwitch(t *testing.T) {
	// Tab moves focus from the main list to the side (tree) pane on Processes.
	m, _ := step(t, InitialModel("test"), tea.KeyMsg{Type: tea.KeyTab})
	if m.listFocus != focusSide {
		t.Errorf("tab should move focus to the side pane, got %v", m.listFocus)
	}
}

func TestUpdateListNavMovesCursor(t *testing.T) {
	m := InitialModel("test")
	m.processes = []model.Process{{PID: 1}, {PID: 2}, {PID: 3}}
	m.filterProcesses()

	m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyDown})
	if got := m.table.Cursor(); got != 1 {
		t.Errorf("down should move the cursor to 1, got %d", got)
	}
}

func TestUpdateTogglesShowAllPorts(t *testing.T) {
	m := InitialModel("test")
	m.activeTab = tabPorts
	if m.showAllPorts {
		t.Fatal("precondition: showAllPorts should start false")
	}
	m, _ = step(t, m, keyRunes("a"))
	if !m.showAllPorts {
		t.Error("'a' on the Ports tab should toggle showAllPorts on")
	}
}

func TestUpdateSlashFocusesFilter(t *testing.T) {
	m, cmd := step(t, InitialModel("test"), keyRunes("/"))
	if !m.input.Focused() {
		t.Error("'/' should focus the process filter input")
	}
	if cmd == nil {
		t.Error("'/' should return the cursor-blink command")
	}

	// Typing into the focused filter narrows the list.
	m.processes = []model.Process{{PID: 1, Command: "nginx"}, {PID: 2, Command: "redis"}}
	m, _ = step(t, m, keyRunes("n"))
	if m.input.Value() != "n" {
		t.Fatalf("filter value = %q, want \"n\"", m.input.Value())
	}
	if len(m.filtered) != 1 || m.filtered[0].Command != "nginx" {
		t.Errorf("typing 'n' should narrow to [nginx], got %v", m.filtered)
	}
}

func TestHandleSortKey(t *testing.T) {
	t.Run("processes new column defaults to desc, re-press toggles", func(t *testing.T) {
		m := InitialModel("test") // starts sorted by mem
		m2, _, handled := m.handleSortKey(keyRunes("p"))
		if !handled || m2.sortCol != "pid" || !m2.sortDesc {
			t.Fatalf("'p' => handled=%v col=%q desc=%v, want true/pid/true", handled, m2.sortCol, m2.sortDesc)
		}
		m3, _, _ := m2.handleSortKey(keyRunes("p"))
		if m3.sortDesc {
			t.Errorf("re-pressing 'p' should flip to ascending")
		}
	})

	t.Run("ports column defaults to asc", func(t *testing.T) {
		m := InitialModel("test")
		m.activeTab = tabPorts
		m2, _, handled := m.handleSortKey(keyRunes("s"))
		if !handled || m2.sortPortCol != "state" || m2.sortPortDesc {
			t.Errorf("'s' on Ports => handled=%v col=%q desc=%v, want true/state/false", handled, m2.sortPortCol, m2.sortPortDesc)
		}
	})

	t.Run("containers and locks map keys to columns", func(t *testing.T) {
		mc := InitialModel("test")
		mc.activeTab = tabContainers
		if m2, _, h := mc.handleSortKey(keyRunes("r")); !h || m2.sortContainerCol != "runtime" {
			t.Errorf("'r' on Containers => handled=%v col=%q, want true/runtime", h, m2.sortContainerCol)
		}

		ml := InitialModel("test")
		ml.activeTab = tabLocks
		if m2, _, h := ml.handleSortKey(keyRunes("f")); !h || m2.sortLockCol != "path" {
			t.Errorf("'f' on Locks => handled=%v col=%q, want true/path", h, m2.sortLockCol)
		}
	})

	t.Run("irrelevant key is not handled", func(t *testing.T) {
		m := InitialModel("test")
		if _, _, handled := m.handleSortKey(keyRunes("z")); handled {
			t.Error("'z' is not a sort key and must report not-handled")
		}
	})
}

func TestUpdateResizeSetsDimsAndCmdColumn(t *testing.T) {
	// A wide window leaves room for the Command column.
	m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 200, Height: 50})
	if m.width != 200 || m.height != 50 {
		t.Errorf("dims = %dx%d, want 200x50", m.width, m.height)
	}
	if !m.showCmdCol {
		t.Error("a 200-wide window should show the Command column")
	}

	// A narrow window hides it.
	m, _ = step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 40, Height: 20})
	if m.showCmdCol {
		t.Error("a 40-wide window should hide the Command column")
	}
}

// Keep time import used if other tests need it.
var _ = time.Now
var _ = fmt.Sprintf
