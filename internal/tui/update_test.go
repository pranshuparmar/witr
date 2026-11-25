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
	m, _ := step(t, InitialModel("test"), keyRunes("/"))
	if !m.input.Focused() {
		t.Error("'/' should focus the process filter input")
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

func TestUpdateProcessListMessage(t *testing.T) {
	m := InitialModel("test")
	m, cmd := step(t, m, []model.Process{{PID: 10, Command: "a"}, {PID: 20, Command: "b"}})
	if len(m.processes) != 2 || len(m.filtered) != 2 {
		t.Fatalf("process list message should populate processes/filtered, got %d/%d", len(m.processes), len(m.filtered))
	}
	if cmd == nil {
		t.Error("a non-empty process list should schedule a tree fetch for the selection")
	}
}

func TestUpdateDataListMessages(t *testing.T) {
	m := InitialModel("test")

	m, _ = step(t, m, []model.OpenPort{{Port: 80, Protocol: "tcp", State: "LISTEN"}})
	if len(m.ports) != 1 {
		t.Errorf("port list message: ports = %d, want 1", len(m.ports))
	}

	m, _ = step(t, m, []*model.ContainerMatch{{Name: "web"}})
	if len(m.containers) != 1 {
		t.Errorf("container list message: containers = %d, want 1", len(m.containers))
	}

	m, _ = step(t, m, []*model.LockedFile{{PID: 1, Path: "/a"}})
	if len(m.locks) != 1 {
		t.Errorf("lock list message: locks = %d, want 1", len(m.locks))
	}
}

func TestUpdateResultMessageSetsDetail(t *testing.T) {
	m := InitialModel("test")
	res := model.Result{Process: model.Process{PID: 1, Command: "x"}, Ancestry: []model.Process{{PID: 1, Command: "x"}}}
	m, _ = step(t, m, res)
	if m.selectedDetail == nil {
		t.Error("a Result message should populate selectedDetail")
	}
}

func TestUpdateErrorMessageRevertsToList(t *testing.T) {
	m := InitialModel("test")
	m.state = stateDetail
	m, cmd := step(t, m, error(fmt.Errorf("boom")))
	if m.state != stateList {
		t.Errorf("an error should revert to the list view, state = %v", m.state)
	}
	if m.statusMsg == "" {
		t.Error("an error should surface a status message")
	}
	if cmd == nil {
		t.Error("an error should trigger a process refresh")
	}
}

func TestHandleTickRefreshesWhenDue(t *testing.T) {
	m := InitialModel("test")
	old := time.Now().Add(-time.Hour) // long past the cadence -> a tick is due
	m.lastRefresh = old

	nm, cmd := m.handleTick(tickMsg(time.Now()))
	m = nm.(MainModel)
	if !m.lastRefresh.After(old) {
		t.Error("a due tick should advance lastRefresh")
	}
	if cmd == nil {
		t.Error("handleTick must always reschedule the next tick")
	}
}

func TestDetailKeyNavigation(t *testing.T) {
	base := func() MainModel {
		m := InitialModel("test")
		m.state = stateDetail
		m.selectedDetail = &model.Result{Process: model.Process{PID: 1}}
		return m
	}

	t.Run("esc returns to list", func(t *testing.T) {
		m, cmd := step(t, base(), tea.KeyMsg{Type: tea.KeyEsc})
		if m.state != stateList || m.selectedDetail != nil {
			t.Errorf("esc should return to list and clear detail; state=%v detail=%v", m.state, m.selectedDetail)
		}
		if cmd == nil {
			t.Error("leaving detail should refresh the list")
		}
	})

	t.Run("tab toggles detail/env focus", func(t *testing.T) {
		m := base()
		if m.detailFocus != focusDetail {
			t.Fatalf("precondition: detailFocus = %v, want focusDetail", m.detailFocus)
		}
		m, _ = step(t, m, tea.KeyMsg{Type: tea.KeyTab})
		if m.detailFocus != focusEnv {
			t.Errorf("tab should move focus to the env pane, got %v", m.detailFocus)
		}
	})
}

// actionModel returns a laid-out list view holding two processes.
func actionModel(t *testing.T) MainModel {
	t.Helper()
	m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 160, Height: 40})
	m.processes = []model.Process{{PID: 101, Command: "alpha"}, {PID: 202, Command: "bravo"}}
	m.filterProcesses()
	m.table.SetCursor(0)
	return m
}

func TestListActionKeyOpensMenuForHighlightedProcess(t *testing.T) {
	if !actionsSupported {
		t.Skip("process actions are not supported on this platform")
	}
	m := actionModel(t)
	m.table.SetCursor(1)
	m, _ = step(t, m, keyRunes("a"))
	if !m.actionMenuOpen || m.actionTarget == nil || m.actionTarget.PID != 202 {
		t.Fatalf("a should open the menu for PID 202; open=%v target=%+v", m.actionMenuOpen, m.actionTarget)
	}
}

func TestActionKeysTakePrecedence(t *testing.T) {
	open := func() MainModel {
		m := actionModel(t)
		m.openActionMenu(m.filtered[0])
		return m
	}

	t.Run("menu key picks the action, not a sort", func(t *testing.T) {
		m, _ := step(t, open(), keyRunes("t"))
		if m.pendingAction != actionTerm || m.sortCol != "mem" {
			t.Errorf("pendingAction=%v sortCol=%q, want actionTerm and unchanged sort", m.pendingAction, m.sortCol)
		}
	})

	t.Run("tab-switch digits are ignored", func(t *testing.T) {
		m, _ := step(t, open(), keyRunes("2"))
		if m.activeTab != tabProcesses || !m.actionMenuOpen {
			t.Errorf("activeTab=%v menuOpen=%v, want the menu to stay open on the Processes tab", m.activeTab, m.actionMenuOpen)
		}
	})

	t.Run("esc cancels instead of quitting", func(t *testing.T) {
		m, _ := step(t, open(), tea.KeyMsg{Type: tea.KeyEsc})
		if m.quitting || m.actionActive() || m.actionTarget != nil {
			t.Errorf("quitting=%v active=%v target=%v, want the menu closed and the app running", m.quitting, m.actionActive(), m.actionTarget)
		}
	})

	t.Run("n at the prompt declines instead of sorting", func(t *testing.T) {
		m, _ := step(t, open(), keyRunes("k"))
		m, _ = step(t, m, keyRunes("n"))
		if m.actionActive() || m.sortCol != "mem" {
			t.Errorf("active=%v sortCol=%q, want the prompt declined and the sort unchanged", m.actionActive(), m.sortCol)
		}
	})

	t.Run("renice input accepts tab-switch digits", func(t *testing.T) {
		m, _ := step(t, open(), keyRunes("n"))
		m, _ = step(t, m, keyRunes("1"))
		m, _ = step(t, m, keyRunes("2"))
		if got := m.reniceInput.Value(); got != "12" || m.activeTab != tabProcesses {
			t.Errorf("renice input = %q on tab %v, want \"12\" on the Processes tab", got, m.activeTab)
		}
	})
}

func TestActionOpenPausesRefreshAndMouse(t *testing.T) {
	m := actionModel(t)
	m.openActionMenu(m.filtered[0])

	old := time.Now().Add(-time.Hour)
	m.lastRefresh = old
	nm, _ := m.handleTick(tickMsg(time.Now()))
	if !nm.(MainModel).lastRefresh.Equal(old) {
		t.Error("the list must not refresh while an action menu is open")
	}

	// A click on the "2. Ports" tab would normally switch tabs.
	m, _ = step(t, m, tea.MouseMsg{X: 25, Y: 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if m.activeTab != tabProcesses || !m.actionMenuOpen {
		t.Errorf("activeTab=%v menuOpen=%v, want mouse input ignored while the menu is open", m.activeTab, m.actionMenuOpen)
	}
}

func TestWithTargetsMatchExactlyAndListSkipped(t *testing.T) {
	m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 160, Height: 40})
	m = m.withTargets([]model.Target{
		{Type: model.TargetPort, Value: "80"},
		{Type: model.TargetPort, Value: "443"},
		{Type: model.TargetName, Value: "nginx"},
	}, true)

	// A seeded port matches exactly: not 8080, nor an fe80:: address.
	m.ports = []model.OpenPort{
		{Port: 80, Protocol: "tcp", Address: "0.0.0.0", State: "LISTEN"},
		{Port: 8080, Protocol: "tcp", Address: "0.0.0.0", State: "LISTEN"},
		{Port: 22, Protocol: "tcp", Address: "fe80::1", State: "LISTEN"},
	}
	m.updatePortTable()
	if got := len(m.portTable.Rows()); got != 1 {
		t.Errorf("port rows = %d, want only port 80", got)
	}

	// With -x, a seeded name matches the process name exactly.
	m.processes = []model.Process{{PID: 1, Command: "nginx"}, {PID: 2, Command: "nginx-helper"}}
	m.filterProcesses()
	if len(m.filtered) != 1 || m.filtered[0].PID != 1 {
		t.Errorf("filtered = %+v, want only the exact name match", m.filtered)
	}

	if !strings.Contains(m.statusMsg, "port 443") {
		t.Errorf("status = %q, want the second port listed as not shown", m.statusMsg)
	}
}

func manyProcesses(n int) []model.Process {
	procs := make([]model.Process, n)
	for i := range procs {
		// Distinct memory keeps the default sort (memory, descending) in
		// PID order.
		procs[i] = model.Process{PID: i, Command: fmt.Sprintf("p%d", i), MemoryRSS: uint64(n-i) * 1024}
	}
	return procs
}

// `witr -i --pid N` shows N's row near the middle of the table, even when it
// is far down the list.
func TestInitialPIDIsScrolledIntoView(t *testing.T) {
	m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 160, Height: 40})
	m = m.withTargets([]model.Target{{Type: model.TargetPID, Value: "250"}}, false)
	nm, _ := m.handleProcessList(manyProcesses(400))
	m = nm.(MainModel)

	if row := m.table.SelectedRow(); len(row) == 0 || strings.TrimSpace(row[0]) != "250" {
		t.Fatalf("selected row = %v, want PID 250", row)
	}
	var lines []string
	for _, l := range strings.Split(m.table.View(), "\n")[1:] {
		if strings.TrimSpace(l) != "" {
			lines = append(lines, l)
		}
	}
	at := -1
	for i, l := range lines {
		if f := strings.Fields(l); len(f) > 0 && f[0] == "250" {
			at = i
		}
	}
	if at < len(lines)/4 || at > len(lines)*3/4 {
		t.Errorf("PID 250 shows at line %d of %d, want it near the middle", at, len(lines))
	}
}

// A refresh keeps PID 0 selected rather than jumping back to the top.
func TestRefreshKeepsPIDZeroSelected(t *testing.T) {
	m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 160, Height: 40})
	procs := manyProcesses(50)
	nm, _ := m.handleProcessList(procs)
	m = nm.(MainModel)
	for i, p := range m.filtered {
		if p.PID == 0 {
			m.table.SetCursor(i)
		}
	}
	nm, _ = m.handleProcessList(procs)
	m = nm.(MainModel)
	if row := m.table.SelectedRow(); len(row) == 0 || strings.TrimSpace(row[0]) != "0" {
		t.Errorf("after a refresh the selection is %v, want PID 0", row)
	}
}

func TestWithTargetsSeedsInitialState(t *testing.T) {
	t.Run("pid target selects that process on first list", func(t *testing.T) {
		m := InitialModel("test").withTargets([]model.Target{{Type: model.TargetPID, Value: "2"}}, false)
		if m.initialPID != 2 {
			t.Fatalf("initialPID = %d, want 2", m.initialPID)
		}
		m, cmd := step(t, m, []model.Process{{PID: 1, Command: "a"}, {PID: 2, Command: "b"}, {PID: 3, Command: "c"}})
		if got := m.table.Cursor(); got != 1 {
			t.Errorf("cursor = %d, want 1 (row of pid 2)", got)
		}
		if cmd == nil {
			t.Error("selecting the pid should fetch its tree")
		}
		if m.initialPID != 0 {
			t.Errorf("initialPID should be cleared after first list, got %d", m.initialPID)
		}
	})

	t.Run("name target pre-fills the process filter", func(t *testing.T) {
		m := InitialModel("test").withTargets([]model.Target{{Type: model.TargetName, Value: "nginx"}}, false)
		m, _ = step(t, m, []model.Process{{PID: 1, Command: "nginx"}, {PID: 2, Command: "redis"}})
		if len(m.filtered) != 1 || m.filtered[0].Command != "nginx" {
			t.Errorf("name target should narrow to [nginx], got %v", m.filtered)
		}
	})

	t.Run("port target opens the ports tab with the filter set", func(t *testing.T) {
		m := InitialModel("test").withTargets([]model.Target{{Type: model.TargetPort, Value: "5432"}}, false)
		if m.activeTab != tabPorts {
			t.Errorf("activeTab = %v, want tabPorts", m.activeTab)
		}
		if m.portInput.Value() != "5432" {
			t.Errorf("port filter = %q, want \"5432\"", m.portInput.Value())
		}
		m, _ = step(t, m, []model.OpenPort{{Port: 5432, Protocol: "tcp", State: "LISTEN"}, {Port: 80, Protocol: "tcp", State: "LISTEN"}})
		if rows := m.portTable.Rows(); len(rows) != 1 {
			t.Errorf("port list should be narrowed to 1 row, got %d", len(rows))
		}
	})

	t.Run("container target opens the containers tab with the filter set", func(t *testing.T) {
		m := InitialModel("test").withTargets([]model.Target{{Type: model.TargetContainer, Value: "web"}}, false)
		if m.activeTab != tabContainers || m.containerInput.Value() != "web" {
			t.Errorf("activeTab = %v, filter = %q", m.activeTab, m.containerInput.Value())
		}
	})

	t.Run("no targets leaves defaults", func(t *testing.T) {
		m := InitialModel("test").withTargets(nil, false)
		if m.activeTab != tabProcesses || m.initialPID != 0 || m.input.Value() != "" {
			t.Errorf("unexpected seeded state: tab=%v pid=%d filter=%q", m.activeTab, m.initialPID, m.input.Value())
		}
	})
}

// Enter on a port's owner row opens it: a visible process by its PID (read
// from the right-justified cell), and an owner that isn't visible ("-") through
// the container publishing the port, as --port does.
func TestPortsTabOpensOwner(t *testing.T) {
	orig, origHidden := resolvePublishingContainer, ownersHidden
	defer func() { resolvePublishingContainer, ownersHidden = orig, origHidden }()
	ownersHidden = func() bool { return true }
	var asked []any
	resolvePublishingContainer = func(port int, proto string) *model.ContainerMatch {
		asked = append(asked, port, proto)
		return nil
	}

	setup := func(owner int) MainModel {
		m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 160, Height: 40})
		m.activeTab = tabPorts
		m.processes = []model.Process{{PID: 4321, Command: "nginx", User: "root"}}
		m.ports = []model.OpenPort{{Port: 18090, Protocol: "TCP6", Address: "::", State: "LISTEN", PID: owner}}
		m.updatePortTable()
		m.updatePortDetails()
		m.listFocus = focusSide
		return m
	}

	m := setup(4321)
	if row := m.portDetailTable.SelectedRow(); len(row) == 0 || row[0] != "    4321" {
		t.Fatalf("owner row = %q, want the right-justified PID", row)
	}
	m, cmd := step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateDetail || cmd == nil {
		t.Errorf("Enter on a visible owner: state %v, cmd %v; want the detail view loading", m.state, cmd)
	}

	m = setup(0)
	if row := m.portDetailTable.SelectedRow(); len(row) == 0 || strings.TrimSpace(row[0]) != "-" {
		t.Fatalf("owner row = %q, want the not-visible row", row)
	}
	m, cmd = step(t, m, tea.KeyMsg{Type: tea.KeyEnter})
	if m.state != stateDetail || cmd == nil {
		t.Fatalf("Enter on a hidden owner: state %v, cmd %v; want the container lookup", m.state, cmd)
	}
	msg := cmd()
	if len(asked) != 2 || asked[0] != 18090 || asked[1] != "tcp" {
		t.Errorf("container lookup asked %v, want port 18090 over tcp", asked)
	}
	if err, ok := msg.(error); !ok || !strings.Contains(err.Error(), "port 18090") {
		t.Errorf("no publishing container: msg %v, want an error naming the port", msg)
	}
	m, _ = step(t, m, msg)
	if m.state != stateList || !strings.Contains(m.statusMsg, "isn't visible") {
		t.Errorf("after the lookup: state %v, status %q", m.state, m.statusMsg)
	}
}

// The row of a hidden owner names the user its socket belongs to, and says how
// to see the process when witr isn't running as root.
func TestPortsTabHiddenOwnerRow(t *testing.T) {
	orig := ownersHidden
	defer func() { ownersHidden = orig }()
	for _, hidden := range []bool{true, false} {
		ownersHidden = func() bool { return hidden }
		m, _ := step(t, InitialModel("test"), tea.WindowSizeMsg{Width: 160, Height: 40})
		m.activeTab = tabPorts
		m.ports = []model.OpenPort{{Port: 5432, Protocol: "TCP", Address: "127.0.0.1", State: "LISTEN", User: "postgres"}}
		m.updatePortTable()
		m.updatePortDetails()
		row := m.portDetailTable.SelectedRow()
		want := "owning process not visible"
		if hidden {
			want = "run witr with sudo to see it"
		}
		if len(row) != 4 || strings.TrimSpace(row[0]) != "-" || row[1] != "postgres" || row[3] != want {
			t.Errorf("hidden=%v: row %q, want user postgres and %q", hidden, row, want)
		}
	}
}
