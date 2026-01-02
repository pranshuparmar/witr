//go:build linux || darwin

package tui

import (
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		if !m.paused {
			m.refreshing = true
			return m, tea.Batch(refreshCmd(m.pattern), tickCmd())
		}
		return m, tickCmd()

	case processesMsg:
		m.refreshing = false
		m.lastRefresh = time.Now()
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		m.err = nil
		m.processes = msg.processes
		m.applyFilter()

		// Fetch details for current process
		if p := m.currentProcess(); p != nil && m.detailsPID != p.PID {
			m.detailsPID = p.PID
			cmds = append(cmds, fetchDetailsCmd(p.PID))
		}
		return m, tea.Batch(cmds...)

	case detailsMsg:
		if msg.pid == m.detailsPID {
			m.details = &ProcessDetails{
				PID:      msg.pid,
				Ancestry: msg.ancestry,
				Ports:    msg.ports,
				Err:      msg.err,
			}
		}
		return m, nil

	case killResultMsg:
		m.killResults = &msg
		// Remove killed processes from selection
		for _, pid := range msg.success {
			delete(m.selected, pid)
		}
		// Trigger immediate refresh
		m.refreshing = true
		return m, refreshCmd(m.pattern)
	}

	return m, nil
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle filter mode input
	if m.filterMode {
		return m.handleFilterInput(msg)
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.keys.Up):
		if m.cursorIndex > 0 {
			m.cursorIndex--
			// Update details for new selection
			if p := m.currentProcess(); p != nil && m.detailsPID != p.PID {
				m.detailsPID = p.PID
				return m, fetchDetailsCmd(p.PID)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Down):
		if m.cursorIndex < len(m.filtered)-1 {
			m.cursorIndex++
			// Update details for new selection
			if p := m.currentProcess(); p != nil && m.detailsPID != p.PID {
				m.detailsPID = p.PID
				return m, fetchDetailsCmd(p.PID)
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.Select):
		if p := m.currentProcess(); p != nil {
			// Toggle selection
			if m.selected[p.PID] {
				delete(m.selected, p.PID)
			} else {
				m.selected[p.PID] = true
			}
			// Move to next row
			if m.cursorIndex < len(m.filtered)-1 {
				m.cursorIndex++
			}
		}
		return m, nil

	case key.Matches(msg, m.keys.SelectAll):
		// Select all visible processes
		for _, p := range m.filtered {
			m.selected[p.PID] = true
		}
		return m, nil

	case key.Matches(msg, m.keys.DeselectAll):
		// Deselect all
		m.selected = make(map[int]bool)
		return m, nil

	case key.Matches(msg, m.keys.Kill):
		return m.handleKill()

	case key.Matches(msg, m.keys.Pause):
		m.paused = !m.paused
		return m, nil

	case key.Matches(msg, m.keys.Filter):
		m.filterMode = true
		return m, nil

	case key.Matches(msg, m.keys.Sort):
		// Cycle through sort fields
		m.sortField = (m.sortField + 1) % 4
		m.sortProcesses()
		return m, nil
	}

	return m, nil
}

// handleFilterInput handles input in filter mode
func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Escape):
		m.filterMode = false
		return m, nil

	case msg.Type == tea.KeyEnter:
		m.filterMode = false
		return m, nil

	case msg.Type == tea.KeyBackspace:
		if len(m.filterText) > 0 {
			m.filterText = m.filterText[:len(m.filterText)-1]
			m.applyFilter()
		}
		return m, nil

	case msg.Type == tea.KeyRunes:
		m.filterText += string(msg.Runes)
		m.applyFilter()
		return m, nil
	}

	return m, nil
}

// handleKill kills selected processes (or current if none selected)
func (m Model) handleKill() (tea.Model, tea.Cmd) {
	pidsToKill := make([]int, 0)

	// If any selected, kill selected; otherwise kill current
	if m.selectedCount() > 0 {
		for _, p := range m.filtered {
			if m.selected[p.PID] {
				pidsToKill = append(pidsToKill, p.PID)
			}
		}
	} else if p := m.currentProcess(); p != nil {
		pidsToKill = append(pidsToKill, p.PID)
	}

	if len(pidsToKill) == 0 {
		return m, nil
	}

	// Kill asynchronously
	return m, killCmd(pidsToKill)
}

// killCmd sends SIGKILL to the specified PIDs
func killCmd(pids []int) tea.Cmd {
	return func() tea.Msg {
		success := make([]int, 0)
		failed := make(map[int]error)

		for _, pid := range pids {
			err := syscall.Kill(pid, syscall.SIGKILL)
			if err != nil {
				failed[pid] = err
			} else {
				success = append(success, pid)
			}
		}

		return killResultMsg{
			success: success,
			failed:  failed,
		}
	}
}
