package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuparmar/witr/pkg/model"
)

// sized returns a model laid out at the given window size with a little data in
// every tab, so the view functions have something real to render.
func sized(t *testing.T, w, h int) MainModel {
	t.Helper()
	m := InitialModel("v1.2.3")
	m, _ = step(t, m, tea.WindowSizeMsg{Width: w, Height: h})
	m.processes = []model.Process{{PID: 1, Command: "nginx", User: "root"}}
	m.filterProcesses()
	m.ports = []model.OpenPort{{Port: 80, Protocol: "tcp", Address: "0.0.0.0", State: "LISTEN"}}
	m.updatePortTable()
	m.containers = []*model.ContainerMatch{{ID: "abc123", Name: "web", Runtime: "docker"}}
	m.updateContainerTable()
	m.locks = []*model.LockedFile{{PID: 1, Process: "nginx", Path: "/var/run/nginx.pid"}}
	m.updateLockTable()
	return m
}

func TestViewListRendersEveryTab(t *testing.T) {
	for _, sz := range [][2]int{{120, 40}, {80, 24}} {
		for _, tabName := range []tab{tabProcesses, tabPorts, tabContainers, tabLocks, tabNetwork} {
			m := sized(t, sz[0], sz[1])
			m.activeTab = tabName
			out := m.View()
			if out == "" {
				t.Errorf("View() empty at %dx%d tab %d", sz[0], sz[1], tabName)
			}
			if !strings.Contains(out, "witr") {
				t.Errorf("View() missing title at %dx%d tab %d", sz[0], sz[1], tabName)
			}
		}
	}
}

func TestViewListSearchAndStatus(t *testing.T) {
	// A transient status message takes over the status line.
	m := sized(t, 120, 40)
	m.statusMsg = "Error: nope"
	if out := m.View(); !strings.Contains(out, "nope") {
		t.Errorf("status message not rendered:\n%s", out)
	}

	// With the filter focused, the help line switches to search mode.
	m = sized(t, 120, 40)
	m.input.Focus()
	if out := m.View(); !strings.Contains(out, "Searching") {
		t.Errorf("focused filter should render search-mode status:\n%s", out)
	}
}

func TestViewProcessDetail(t *testing.T) {
	m := sized(t, 120, 40)
	m.state = stateDetail
	m.selectedDetail = &model.Result{
		Process:  model.Process{PID: 42, Command: "nginx"},
		Ancestry: []model.Process{{PID: 1, Command: "systemd"}, {PID: 42, Command: "nginx"}},
	}
	m.updateDetailViewport()
	m.updateEnvViewport()

	out := m.View()
	if !strings.Contains(out, "witr") || !strings.Contains(out, "PID 42") {
		t.Errorf("process detail view missing expected chrome:\n%s", out)
	}
}

func TestViewProcessDetailFooterStates(t *testing.T) {
	base := func() MainModel {
		m := sized(t, 120, 40)
		m.state = stateDetail
		m.selectedDetail = &model.Result{Process: model.Process{PID: 7, Command: "x"}}
		m.updateDetailViewport()
		return m
	}

	tests := []struct {
		name    string
		mutate  func(*MainModel)
		wantSub string
	}{
		{"action menu", func(m *MainModel) { m.actionMenuOpen = true }, "Actions"},
		{"confirm kill", func(m *MainModel) { m.pendingAction = actionKill }, "Kill PID 7"},
		{"confirm pause", func(m *MainModel) { m.pendingAction = actionPause }, "Pause PID 7"},
		{"renice prompt", func(m *MainModel) { m.pendingAction = actionRenice }, "Nice value"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := base()
			tt.mutate(&m)
			if out := m.View(); !strings.Contains(out, tt.wantSub) {
				t.Errorf("footer for %s missing %q:\n%s", tt.name, tt.wantSub, out)
			}
		})
	}
}

func TestViewContainerDetail(t *testing.T) {
	m := sized(t, 120, 40)
	m.state = stateDetail
	m.selectedContainer = &model.ContainerMatch{ID: "abc123def456", Name: "web", Runtime: "docker"}
	m.selectedDetail = nil
	m.updateDetailViewport()

	if out := m.View(); !strings.Contains(out, "witr") {
		t.Errorf("container detail view missing title:\n%s", out)
	}
}

func TestViewDetailLoading(t *testing.T) {
	m := sized(t, 120, 40)
	m.state = stateDetail
	m.selectedDetail = nil
	m.selectedContainer = nil

	if out := m.View(); !strings.Contains(out, "Loading") {
		t.Errorf("detail view with no data should show a loading placeholder:\n%s", out)
	}
}

func TestViewQuittingIsBlank(t *testing.T) {
	m := sized(t, 120, 40)
	m.quitting = true
	if out := m.View(); out != "" {
		t.Errorf("a quitting model should render nothing, got %q", out)
	}
}
