package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/pkg/model"
)

type treeMsg model.Result

type debounceMsg struct {
	id  int
	pid int
}

type tickMsg time.Time

// waitTick drives the periodic list refresh; 3s mirrors top's default cadence.
func waitTick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m.handleTick(msg)
	case tea.MouseMsg:
		return m.handleMouse(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		return m.handleResize(msg)
	case []model.Process:
		return m.handleProcessList(msg)
	case debounceMsg:
		return m.handleDebounce(msg)
	case []model.OpenPort:
		return m.handlePortList(msg)
	case []*model.ContainerMatch:
		return m.handleContainerList(msg)
	case []*model.LockedFile:
		return m.handleLockList(msg)
	case treeMsg:
		return m.handleTree(msg)
	case model.Result:
		return m.handleResult(msg)
	case *model.ContainerMatch:
		return m.handleContainerDetail(msg)
	case error:
		return m.handleError(msg)
	}
	return m, nil
}

func (m MainModel) handleTick(msg tickMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.state == stateList && !m.quitting && !m.actionActive() && !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() && !m.lockInput.Focused() && m.refreshDue() {
		m.lastRefresh = time.Now()
		m.refreshStartedAt = m.lastRefresh
		cmd = m.refreshProcesses()
		switch m.activeTab {
		case tabPorts:
			cmd = tea.Batch(cmd, m.refreshPorts())
		case tabContainers:
			cmd = tea.Batch(cmd, m.refreshContainers())
		case tabLocks:
			cmd = tea.Batch(cmd, m.refreshLocks())
		}
	}
	return m, tea.Batch(cmd, waitTick())
}

// refreshDue reports whether enough time has elapsed since the last background
// refresh, given the adaptive cadence. The timer still ticks at the base rate;
// this just decides whether a given tick does the (expensive) re-enumeration.
// refreshDue reports whether the next background refresh should run: the
// adaptive interval has elapsed and no refresh is already in flight, so a slow
// refresh can't overlap the next tick. A pathologically old in-flight marker is
// ignored so a lost result can't wedge refreshing permanently.
func (m MainModel) refreshDue() bool {
	if !m.refreshStartedAt.IsZero() && time.Since(m.refreshStartedAt) < maxRefreshInterval {
		return false
	}
	every := m.refreshEvery
	if every < refreshInterval {
		every = refreshInterval
	}
	return time.Since(m.lastRefresh) >= every
}

// adjustRefreshInterval applies one refresh-duration sample to the adaptive
// cadence. After backoffStreak consecutive refreshes over slowFraction of the
// interval it grows by refreshStep (capped at maxRefreshInterval); after
// backoffStreak under fastFraction it shrinks by refreshStep (floored at
// refreshInterval); the band between is stable. Returns the new interval and
// the updated slow/fast streak counters.
func adjustRefreshInterval(interval, took time.Duration, slow, fast int) (time.Duration, int, int) {
	switch {
	case took > time.Duration(float64(interval)*slowFraction):
		slow, fast = slow+1, 0
		if slow >= backoffStreak {
			interval += refreshStep
			if interval > maxRefreshInterval {
				interval = maxRefreshInterval
			}
			slow = 0
		}
	case took < time.Duration(float64(interval)*fastFraction):
		fast, slow = fast+1, 0
		if fast >= backoffStreak {
			interval -= refreshStep
			if interval < refreshInterval {
				interval = refreshInterval
			}
			fast = 0
		}
	default:
		slow, fast = 0, 0
	}
	return interval, slow, fast
}

func (m MainModel) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// An open action menu or prompt must be answered from the keyboard; a
	// stray click could otherwise move the selection or switch views under it.
	if m.actionActive() {
		return m, nil
	}
	m.statusMsg = "" // clear any transient error on interaction
	if msg.Action != tea.MouseActionPress && msg.Action != tea.MouseActionMotion && msg.Action != tea.MouseActionRelease {
		return m, nil
	}

	// Required for Windows: isClick is true only for real pointer presses (not scroll wheel).
	isWheel := msg.Button == tea.MouseButtonWheelUp ||
		msg.Button == tea.MouseButtonWheelDown ||
		msg.Button == tea.MouseButtonWheelLeft ||
		msg.Button == tea.MouseButtonWheelRight
	isClick := msg.Action == tea.MouseActionPress && !isWheel

	isDoubleClick := false
	if isClick {
		clickDuration := time.Since(m.lastClickTime)
		if clickDuration < doubleClickThreshold {
			distX := m.lastClickX - msg.X
			distY := m.lastClickY - msg.Y
			if distX < 0 {
				distX = -distX
			}
			if distY < 0 {
				distY = -distY
			}
			if distX <= 2 && distY <= 1 {
				isDoubleClick = true
			}
		}
		m.lastClickTime = time.Now()
		m.lastClickX = msg.X
		m.lastClickY = msg.Y
	}

	// Handle "witr" Title Click (Home)
	if msg.Y == 1 && isClick && msg.X >= 1 && msg.X <= 6 {
		m.state = stateList
		m.activeTab = tabProcesses
		return m, m.refreshProcesses()
	}

	// Handle Detail View Clicks
	if m.state == stateDetail {
		return m.handleDetailMouse(msg, isClick)
	}

	// Handle Search Clicks: blur search if clicking outside the input row
	if isClick && msg.Y != 5 {
		if m.input.Focused() {
			m.input.Blur()
		}
		if m.portInput.Focused() {
			m.portInput.Blur()
		}
		if m.containerInput.Focused() {
			m.containerInput.Blur()
		}
		if m.lockInput.Focused() {
			m.lockInput.Blur()
		}
	}

	// Tabs. X ranges are inactiveTabStyle widths: "1. Processes"=14,
	// "2. Ports"=10, "3. Containers"=15, "4. Locks"=10 (inc. 1ch padding).
	if msg.Y == 1 && isClick {
		if msg.X >= 8 && msg.X < 22 { // "1. Processes"
			if m.activeTab != tabProcesses {
				m.activeTab = tabProcesses
				m.listFocus = focusMain
				return m, nil
			}
		} else if msg.X >= 22 && msg.X < 32 { // "2. Ports"
			if m.activeTab != tabPorts {
				m.activeTab = tabPorts
				m.listFocus = focusMain
				return m, m.refreshPorts()
			}
		} else if msg.X >= 32 && msg.X < 47 { // "3. Containers"
			if m.activeTab != tabContainers {
				m.activeTab = tabContainers
				m.listFocus = focusMain
				return m, m.refreshContainers()
			}
		} else if locksTabEnabled && msg.X >= 47 && msg.X < 57 { // "4. Locks"
			if m.activeTab != tabLocks {
				m.activeTab = tabLocks
				m.listFocus = focusMain
				return m, m.refreshLocks()
			}
		}
	}

	// Handle Search Input Clicks
	if msg.Y == 5 && isClick && m.state == stateList {
		switch m.activeTab {
		case tabProcesses:
			m.input.Focus()
		case tabPorts:
			m.portInput.Focus()
		case tabContainers:
			m.containerInput.Focus()
		case tabLocks:
			m.lockInput.Focus()
		}
		return m, nil
	}

	// Handle Content Area Clicks
	if msg.Y >= 7 {
		contentX := msg.X - 2
		if contentX < 0 {
			return m, nil
		}

		switch m.activeTab {
		case tabProcesses:
			return m.handleProcessAreaMouse(msg, contentX, isClick, isWheel, isDoubleClick)
		case tabPorts:
			return m.handlePortAreaMouse(msg, contentX, isClick, isWheel, isDoubleClick)
		case tabContainers:
			return m.handleContainerAreaMouse(msg, contentX, isClick, isWheel, isDoubleClick)
		case tabLocks:
			return m.handleLockAreaMouse(msg, contentX, isClick, isWheel, isDoubleClick)
		}
	}
	return m, nil
}

func (m MainModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = "" // clear any transient error on interaction
	if msg.String() == "ctrl+c" {
		m.quitting = true
		return m, tea.Quit
	}
	// Action keys take precedence over tab, sort and quit keys in every view.
	if m.actionActive() {
		return m.handleActionKey(msg)
	}
	switch msg.String() {
	case "1":
		if !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() {
			m.activeTab = tabProcesses
			m.listFocus = focusMain
			return m, nil
		}
	case "2":
		if !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() {
			m.activeTab = tabPorts
			m.listFocus = focusMain
			m.portTable.Focus()
			m.portDetailTable.Blur()
			return m, m.refreshPorts()
		}
	case "3":
		if !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() && !m.lockInput.Focused() {
			m.activeTab = tabContainers
			m.listFocus = focusMain
			return m, m.refreshContainers()
		}
	case "4":
		if locksTabEnabled && !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() && !m.lockInput.Focused() {
			m.activeTab = tabLocks
			m.listFocus = focusMain
			return m, m.refreshLocks()
		}
	}

	if m.state == stateList {
		return m.handleListKey(msg)
	} else if m.state == stateDetail {
		return m.handleDetailKey(msg)
	}
	return m, nil
}

func (m MainModel) handleResize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height

	availableWidth := msg.Width - 6
	if availableWidth < 0 {
		availableWidth = 0
	}

	processListHeight := msg.Height - 11
	if processListHeight < 5 {
		processListHeight = 5
	}

	processListPaneWidth := int(float64(availableWidth) * listPaneRatio)
	if processListPaneWidth < 10 {
		processListPaneWidth = 10
	}

	processTablePadding := 4
	processListWidth := processListPaneWidth - processTablePadding
	if processListWidth < 10 {
		processListWidth = 10
	}

	fixedColumnsWidth := 84 // PID(8)+Name(20)+User(12)+CPU(9)+Mem(16)+Started(19)
	cmdWidth := processListWidth - fixedColumnsWidth - 12
	m.showCmdCol = cmdWidth >= 15
	if cmdWidth < 10 {
		cmdWidth = 10
	}

	columns := m.getColumns()
	if m.showCmdCol {
		columns[6].Width = cmdWidth
	}
	m.table.SetWidth(processListWidth)
	m.table.SetHeight(processListHeight)
	m.table.SetRows(nil)
	m.table.SetColumns(columns)
	m.filterProcesses()

	treeWidth := availableWidth - processListPaneWidth - 4
	if treeWidth < 10 {
		treeWidth = 10
	}
	m.treeViewport.Width = treeWidth
	m.treeViewport.Height = processListHeight - 2
	if m.treeViewport.Height < 0 {
		m.treeViewport.Height = 0
	}
	portListHeight := processListHeight

	portPaneWidth := int(float64(availableWidth) * portPaneRatio)
	if portPaneWidth < 0 {
		portPaneWidth = 0
	}

	tablePadding := 4
	portTableWidth := portPaneWidth - tablePadding
	if portTableWidth < 0 {
		portTableWidth = 0
	}

	portColumns := m.getPortColumns()
	fixedPortWidth := 36
	buffer := 6
	addrWidth := portTableWidth - fixedPortWidth - buffer
	if addrWidth < 10 {
		addrWidth = 10
	}
	if len(portColumns) > 2 {
		portColumns[2].Width = addrWidth
	}
	m.portTable.SetColumns(portColumns)
	m.portTable.SetWidth(portTableWidth)
	m.portTable.SetHeight(portListHeight)

	portDetailWidth := availableWidth - portPaneWidth - 5
	if portDetailWidth < 10 {
		portDetailWidth = 10
	}

	pdCols := m.portDetailTable.Columns()
	fixedPdWidth := 35
	buffer = 6
	cmdPdWidth := portDetailWidth - fixedPdWidth - buffer
	if cmdPdWidth < 10 {
		cmdPdWidth = 10
	}
	if len(pdCols) > 3 {
		pdCols[3].Width = cmdPdWidth
	}
	// Center-align PID header in port detail table
	if len(pdCols) > 0 {
		pdCols[0].Title = centerHeader("PID", pdCols[0].Width)
	}
	m.portDetailTable.SetColumns(pdCols)
	m.portDetailTable.SetWidth(portDetailWidth)
	m.portDetailTable.SetHeight(portListHeight - 2)

	m.containerTable.SetWidth(availableWidth)
	m.containerTable.SetHeight(processListHeight)
	// Size the trailing Command column so total column widths match the
	// table width — bubbles table draws the header underline across
	// SetWidth, so column overflow leaves a partial border.
	containerCols := m.containerTable.Columns()
	fixedCC := 0
	const cmdColIdx = 6
	for i, c := range containerCols {
		if i == cmdColIdx {
			continue
		}
		fixedCC += c.Width
	}
	const cellPadding = 2 // tableHeaderStyle Padding(0, 1) → 1 char per side
	paddingBudget := cellPadding * len(containerCols)
	if cmdColIdx < len(containerCols) {
		cmdWidth := availableWidth - fixedCC - paddingBudget
		if cmdWidth < 10 {
			cmdWidth = 10
		}
		containerCols[cmdColIdx].Width = cmdWidth
		m.containerTable.SetColumns(containerCols)
	}

	m.lockTable.SetWidth(availableWidth)
	m.lockTable.SetHeight(processListHeight)
	// Last column (Path) absorbs remaining width, same pattern as Command.
	lockCols := m.lockTable.Columns()
	fixedLC := 0
	pathColIdx := len(lockCols) - 1
	for i, c := range lockCols {
		if i == pathColIdx {
			continue
		}
		fixedLC += c.Width
	}
	if pathColIdx >= 0 {
		pathWidth := availableWidth - fixedLC - cellPadding*len(lockCols)
		if pathWidth < 10 {
			pathWidth = 10
		}
		lockCols[pathColIdx].Width = pathWidth
		m.lockTable.SetColumns(lockCols)
	}

	vpHeight := msg.Height - 9
	if vpHeight < 0 {
		vpHeight = 0
	}
	detailViewWidth := int(float64(availableWidth) * detailPaneRatio)
	envViewWidth := availableWidth - detailViewWidth - 4

	m.viewport.Width = detailViewWidth - 4
	m.viewport.Height = vpHeight

	m.envViewport.Width = envViewWidth
	if m.envViewport.Width < 0 {
		m.envViewport.Width = 0
	}
	m.envViewport.Height = vpHeight

	m.updatePortDetails()
	return m, nil
}

func (m MainModel) handleProcessList(msg []model.Process) (tea.Model, tea.Cmd) {
	// Feed this background refresh's measured duration into the adaptive cadence.
	if !m.refreshStartedAt.IsZero() {
		took := time.Since(m.refreshStartedAt)
		m.refreshStartedAt = time.Time{}
		m.refreshEvery, m.slowStreak, m.fastStreak = adjustRefreshInterval(m.refreshEvery, took, m.slowStreak, m.fastStreak)
	}

	// Keep the selected process selected across refreshes. PID 0 is a valid
	// selection (FreeBSD's kernel), so track whether there is one separately.
	var currentPID int
	hasCurrent := false
	if row := m.table.SelectedRow(); len(row) > 0 {
		_, err := fmt.Sscanf(row[0], "%d", &currentPID)
		hasCurrent = err == nil
	}
	initial := m.initialPID > 0
	if initial {
		currentPID, hasCurrent = m.initialPID, true
		m.initialPID = 0
	}

	m.processes = msg
	m.sortProcesses()
	m.filterProcesses()

	newIdx := 0
	found := false
	if hasCurrent {
		for i, p := range m.filtered {
			if p.PID == currentPID {
				newIdx = i
				found = true
				break
			}
		}
	}

	if len(m.filtered) > 0 {
		if !found {
			newIdx = 0
		}
		if initial && found {
			centerCursor(&m.table, newIdx)
		} else {
			m.table.SetCursor(newIdx)
		}

		m.selectionID++
		p := m.filtered[newIdx]
		return m, m.fetchTree(p)
	}
	return m, nil
}

func (m MainModel) handleDebounce(msg debounceMsg) (tea.Model, tea.Cmd) {
	if msg.id == m.selectionID {
		var targetProc model.Process
		found := false
		row := m.table.SelectedRow()
		if len(row) > 0 {
			var pID int
			fmt.Sscanf(row[0], "%d", &pID)
			if pID == msg.pid {
				idx := m.table.Cursor()
				if idx >= 0 && idx < len(m.filtered) {
					targetProc = m.filtered[idx]
					found = true
				}
			}
		}
		if !found {
			for _, p := range m.processes {
				if p.PID == msg.pid {
					targetProc = p
					found = true
					break
				}
			}
		}

		if found {
			return m, m.fetchTree(targetProc)
		}
	}
	return m, nil
}

func (m MainModel) handlePortList(msg []model.OpenPort) (tea.Model, tea.Cmd) {
	m.ports = msg
	m.updatePortTable()
	m.updatePortDetails()
	return m, nil
}

func (m MainModel) handleContainerList(msg []*model.ContainerMatch) (tea.Model, tea.Cmd) {
	m.containers = msg
	m.updateContainerTable()
	return m, nil
}

func (m MainModel) handleLockList(msg []*model.LockedFile) (tea.Model, tea.Cmd) {
	m.locks = msg
	m.updateLockTable()
	return m, nil
}

func (m MainModel) handleTree(msg treeMsg) (tea.Model, tea.Cmd) {
	selected := m.table.SelectedRow()
	if len(selected) > 0 {
		var currentPID int
		fmt.Sscanf(selected[0], "%d", &currentPID)
		if model.Result(msg).Process.PID == currentPID {
			m.updateTreeViewport(model.Result(msg))
		}
	}
	return m, nil
}

func (m MainModel) handleResult(msg model.Result) (tea.Model, tea.Cmd) {
	m.selectedDetail = &msg
	m.selectedContainer = nil
	m.updateDetailViewport()
	m.updateEnvViewport()
	return m, nil
}

func (m MainModel) handleContainerDetail(msg *model.ContainerMatch) (tea.Model, tea.Cmd) {
	m.selectedContainer = msg
	m.selectedDetail = nil
	m.updateDetailViewport()
	return m, nil
}

func (m MainModel) handleError(msg error) (tea.Model, tea.Cmd) {
	// Revert to list view on any error
	m.state = stateList
	m.selectedDetail = nil
	m.selectedContainer = nil
	m.statusMsg = fmt.Sprintf("Error: %v", msg)
	return m, m.refreshProcesses()
}

func (m MainModel) handleDetailMouse(msg tea.MouseMsg, isClick bool) (tea.Model, tea.Cmd) {
	// Container detail has a single full-width viewport — no env pane,
	// so everything (clicks + wheel) goes straight to m.viewport.
	if m.selectedContainer != nil {
		var cmd tea.Cmd
		detailMsg := msg
		detailMsg.Y -= 3
		detailMsg.X -= 1
		if detailMsg.X >= 0 {
			m.viewport, cmd = m.viewport.Update(detailMsg)
		}
		return m, cmd
	}

	if isClick {
		availableWidth := m.width - 6
		if availableWidth < 0 {
			availableWidth = 0
		}
		detailWidth := int(float64(availableWidth) * detailPaneRatio)
		contentX := msg.X - 2

		if contentX < detailWidth {
			m.detailFocus = focusDetail
		} else {
			m.detailFocus = focusEnv
		}
	}

	var cmd tea.Cmd
	detailMsg := msg
	detailMsg.Y -= 3

	if m.detailFocus == focusDetail {
		detailMsg.X -= 1
		if detailMsg.X >= 0 {
			m.viewport, cmd = m.viewport.Update(detailMsg)
		}
	} else {
		availableWidth := m.width - 6
		if availableWidth < 0 {
			availableWidth = 0
		}
		detailWidth := int(float64(availableWidth) * detailPaneRatio)
		detailMsg.X -= (detailWidth + 2)
		if detailMsg.X >= 0 {
			m.envViewport, cmd = m.envViewport.Update(detailMsg)
		}
	}
	return m, cmd
}

func (m MainModel) handleProcessAreaMouse(msg tea.MouseMsg, contentX int, isClick, isWheel, isDoubleClick bool) (tea.Model, tea.Cmd) {
	availableWidth := m.width - 6
	processListPaneWidth := int(float64(availableWidth) * listPaneRatio)
	if processListPaneWidth < 10 {
		processListPaneWidth = 10
	}

	if contentX < processListPaneWidth {
		if isClick {
			m.listFocus = focusMain

			if msg.Y == 7 {
				m.handleProcessHeaderClick(contentX)
				return m, nil
			}
		}

		var cmd tea.Cmd
		if isWheel {
			// Convert wheel to key so the table scrolls by one row
			// without jumping the cursor to the mouse Y position.
			var keyMsg tea.KeyMsg
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			case tea.MouseButtonWheelDown:
				keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			}
			prevCursor := m.table.Cursor()
			m.table, cmd = m.table.Update(keyMsg)
			if m.table.Cursor() != prevCursor {
				selected := m.table.SelectedRow()
				if len(selected) > 0 {
					pid := 0
					fmt.Sscanf(selected[0], "%d", &pid)
					m.selectionID++
					id := m.selectionID
					return m, tea.Batch(cmd, tea.Tick(selectionDebounce, func(_ time.Time) tea.Msg {
						return debounceMsg{id: id, pid: pid}
					}))
				}
			}
			return m, cmd
		}

		tableMsg := msg
		tableMsg.X -= 2
		tableMsg.Y -= 7

		// Manual Row Selection
		if isClick && tableMsg.Y >= 0 {
			view := m.table.View()
			lines := strings.Split(view, "\n")
			if tableMsg.Y < len(lines) {
				line := stripAnsi(lines[tableMsg.Y])
				fields := strings.Fields(line)
				var pid int
				found := false
				for _, f := range fields {
					if p, err := fmt.Sscanf(f, "%d", &pid); err == nil && p > 0 && pid > 0 {
						found = true
						break
					}
				}

				if found {
					targetIdx := -1
					rows := m.table.Rows()
					for i, row := range rows {
						if len(row) > 0 {
							var rowPID int
							if _, err := fmt.Sscanf(row[0], "%d", &rowPID); err == nil && rowPID == pid {
								targetIdx = i
								break
							}
						}
					}
					if targetIdx >= 0 {
						currentIdx := m.table.Cursor()
						diff := targetIdx - currentIdx
						stepKey := tea.KeyMsg{Type: tea.KeyDown}
						if diff < 0 {
							stepKey = tea.KeyMsg{Type: tea.KeyUp}
							diff = -diff
						}
						for j := 0; j < diff; j++ {
							m.table, _ = m.table.Update(stepKey)
						}
					}
				}
			}
		}

		// Row selection check using translated Y
		if isClick && tableMsg.Y > 0 && tableMsg.Y <= m.table.Height() {
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				pid := 0
				fmt.Sscanf(selected[0], "%d", &pid)

				// Double Click Action: Open Detail
				if isDoubleClick {
					m.state = stateDetail
					m.viewport.GotoTop()
					m.envViewport.GotoTop()
					return m, m.fetchProcessDetail(pid)
				}

				m.selectionID++
				id := m.selectionID
				debounceCmd := tea.Tick(selectionDebounce, func(_ time.Time) tea.Msg {
					return debounceMsg{id: id, pid: pid}
				})
				return m, tea.Batch(cmd, debounceCmd)
			}
		}
		return m, cmd

	} else {
		if isClick {
			m.listFocus = focusSide
			// Translate click to tree cursor position
			// Offset: border(1) + header(1) + spacer(1) + status(1) + input(1) + table-header-border(1) + table-header(1) + tree-header(1) + tree-header-border(1) + "Ancestry Tree:" label(1) = 10
			row := msg.Y - 10 + m.treeViewport.YOffset
			if row >= 0 && row < len(m.treeRows) && m.treeRows[row] >= 0 {
				m.treeCursor = m.treeRows[row]
				m.rerenderTree()

				if isDoubleClick {
					pid := m.treePIDs[m.treeCursor]
					if pid > 0 {
						m.state = stateDetail
						m.viewport.GotoTop()
						m.envViewport.GotoTop()
						return m, m.fetchProcessDetail(pid)
					}
				}
			}
		}
		var cmd tea.Cmd
		// Forward wheel events to viewport for scrolling
		if isWheel {
			treeMsg := msg
			treeMsg.X -= (6 + processListPaneWidth)
			treeMsg.Y -= 8
			if treeMsg.X >= 0 && treeMsg.Y >= 0 {
				m.treeViewport, cmd = m.treeViewport.Update(treeMsg)
			}
		}
		return m, cmd
	}
}

func (m MainModel) handlePortAreaMouse(msg tea.MouseMsg, contentX int, isClick, isWheel, isDoubleClick bool) (tea.Model, tea.Cmd) {
	availableWidth := m.width - 6
	portPaneWidth := int(float64(availableWidth) * portPaneRatio)

	if contentX < portPaneWidth {
		if isClick {
			m.listFocus = focusMain
			m.portDetailTable.Blur()
			m.portTable.Focus()

			if msg.Y == 7 {
				m.handlePortHeaderClick(contentX)
				return m, nil
			}
		}

		var cmd tea.Cmd
		prevSelected := m.portTable.Cursor()

		if isWheel {
			// Convert wheel to key so the port table scrolls by one row.
			var keyMsg tea.KeyMsg
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			case tea.MouseButtonWheelDown:
				keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			}
			m.portTable, cmd = m.portTable.Update(keyMsg)
			if m.portTable.Cursor() != prevSelected {
				m.updatePortDetails()
			}
			return m, cmd
		}

		// Translate for Port Table
		portMsg := msg
		portMsg.X -= 2
		portMsg.Y -= 7
		if portMsg.X >= 0 && portMsg.Y >= 0 {
			m.portTable, cmd = m.portTable.Update(portMsg)
		}

		// Manual Row Selection for Port Table
		if isClick && portMsg.Y >= 0 {
			view := m.portTable.View()
			lines := strings.Split(view, "\n")
			if portMsg.Y < len(lines) {
				line := stripAnsi(lines[portMsg.Y])
				fields := strings.Fields(line)
				var port int
				var protocol string

				if len(fields) >= 2 {
					if p, err := fmt.Sscanf(fields[0], "%d", &port); err == nil && p > 0 && port > 0 {
						protocol = fields[1]
					}
				}

				if port > 0 {
					rows := m.portTable.Rows()
					for i, row := range rows {
						if len(row) >= 2 {
							if p, err := fmt.Sscanf(row[0], "%d", new(int)); err == nil && p > 0 {
								var rowPort int
								fmt.Sscanf(row[0], "%d", &rowPort)
								if rowPort == port && strings.EqualFold(row[1], protocol) {
									m.portTable.SetCursor(i)
									break
								}
							}
						}
					}
				}
			}
		}

		if m.portTable.Cursor() != prevSelected {
			m.updatePortDetails()
		}

		// Double Click (Ports): Focus Attached Processes
		if isDoubleClick && isClick && portMsg.Y > 0 {
			if portMsg.Y <= m.portTable.Height() {
				m.listFocus = focusSide
				m.portTable.Blur()
				m.portDetailTable.Focus()
				return m, cmd
			}
		}

		return m, cmd

	} else {
		if isClick {
			m.listFocus = focusSide
			m.portTable.Blur()
			m.portDetailTable.Focus()
		}

		var cmd tea.Cmd
		detailMsg := msg
		detailMsg.X -= (4 + portPaneWidth)
		detailMsg.Y -= 9
		if detailMsg.X >= 0 && detailMsg.Y >= 0 {
			m.portDetailTable, cmd = m.portDetailTable.Update(detailMsg)
		}

		if msg.Action == tea.MouseActionPress && detailMsg.Y >= 0 {
			view := m.portDetailTable.View()
			lines := strings.Split(view, "\n")
			if detailMsg.Y < len(lines) {
				line := stripAnsi(lines[detailMsg.Y])
				fields := strings.Fields(line)
				var pid int
				found := false
				for _, f := range fields {
					if p, err := fmt.Sscanf(f, "%d", &pid); err == nil && p > 0 && pid > 0 {
						found = true
						break
					}
				}
				if found {
					rows := m.portDetailTable.Rows()
					for i, row := range rows {
						if len(row) > 0 {
							var rowPID int
							n, _ := fmt.Sscanf(row[0], "%d", &rowPID)
							if n == 1 && rowPID == pid {
								m.portDetailTable.SetCursor(i)
								break
							}
						}
					}
				}
			}
		}

		// Double Click (Attached Processes): Open Detail
		if isDoubleClick && isClick && detailMsg.Y > 0 {
			if opened, openCmd, ok := m.openPortOwner(); ok {
				return opened, openCmd
			}
		}
		return m, cmd
	}
}

func (m MainModel) handleContainerAreaMouse(msg tea.MouseMsg, contentX int, isClick, isWheel, isDoubleClick bool) (tea.Model, tea.Cmd) {
	if isClick {
		m.listFocus = focusMain
		if msg.Y == 7 {
			m.handleContainerHeaderClick(contentX)
			return m, nil
		}
	}

	if isWheel {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.containerTable.MoveUp(1)
		case tea.MouseButtonWheelDown:
			m.containerTable.MoveDown(1)
		}
		return m, nil
	}

	if isClick && msg.Y > 7 {
		// Step 1: move cursor to the clicked row (best-effort row match).
		tableY := msg.Y - 7
		lines := strings.Split(m.containerTable.View(), "\n")
		if tableY < len(lines) {
			clickedSig := normalizeRow(stripAnsi(lines[tableY]))
			for i, row := range m.containerTable.Rows() {
				if normalizeRow(strings.Join(row, " ")) == clickedSig {
					diff := i - m.containerTable.Cursor()
					if diff > 0 {
						m.containerTable.MoveDown(diff)
					} else if diff < 0 {
						m.containerTable.MoveUp(-diff)
					}
					break
				}
			}
		}

		// Step 2 (independent): double-click opens detail for whatever
		// row the cursor is currently on. Even if row matching above
		// failed, this still works as long as the cursor is valid.
		if isDoubleClick {
			idx := m.containerTable.Cursor()
			if idx >= 0 && idx < len(m.filteredContainers) {
				match := m.filteredContainers[idx]
				m.state = stateDetail
				m.selectedDetail = nil
				m.selectedContainer = nil
				m.viewport.GotoTop()
				return m, m.fetchContainerDetail(match)
			}
		}
	}
	return m, nil
}

func (m MainModel) handleLockAreaMouse(msg tea.MouseMsg, contentX int, isClick, isWheel, isDoubleClick bool) (tea.Model, tea.Cmd) {
	if isClick {
		m.listFocus = focusMain
		if msg.Y == 7 {
			m.handleLockHeaderClick(contentX)
			return m, nil
		}
	}

	if isWheel {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.lockTable.MoveUp(1)
		case tea.MouseButtonWheelDown:
			m.lockTable.MoveDown(1)
		}
		return m, nil
	}

	if isClick && msg.Y > 7 {
		// Step 1: move cursor to the clicked row.
		tableY := msg.Y - 7
		lines := strings.Split(m.lockTable.View(), "\n")
		if tableY < len(lines) {
			clickedSig := normalizeRow(stripAnsi(lines[tableY]))
			for i, row := range m.lockTable.Rows() {
				if normalizeRow(strings.Join(row, " ")) == clickedSig {
					diff := i - m.lockTable.Cursor()
					if diff > 0 {
						m.lockTable.MoveDown(diff)
					} else if diff < 0 {
						m.lockTable.MoveUp(-diff)
					}
					break
				}
			}
		}

		// Step 2 (independent): double-click opens process detail.
		if isDoubleClick {
			idx := m.lockTable.Cursor()
			if idx >= 0 && idx < len(m.filteredLocks) {
				dblPID := m.filteredLocks[idx].PID
				if dblPID > 0 {
					m.state = stateDetail
					m.selectedDetail = nil
					m.selectedContainer = nil
					m.viewport.GotoTop()
					m.envViewport.GotoTop()
					return m, m.fetchProcessDetail(dblPID)
				}
			}
		}
	}
	return m, nil
}

func (m MainModel) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if nm, cmd, handled := m.handleListFilterInput(msg); handled {
		return nm, cmd
	} else {
		m = nm
	}

	switch msg.String() {
	case "q", "Q", "esc":
		m.quitting = true
		return m, tea.Quit
	case "enter":
		if m.activeTab == tabLocks {
			cursor := m.lockTable.Cursor()
			if cursor >= 0 && cursor < len(m.filteredLocks) {
				pid := m.filteredLocks[cursor].PID
				if pid > 0 {
					m.state = stateDetail
					m.selectedDetail = nil
					m.selectedContainer = nil
					m.viewport.GotoTop()
					m.envViewport.GotoTop()
					return m, m.fetchProcessDetail(pid)
				}
			}
		}
		if m.activeTab == tabContainers {
			cursor := m.containerTable.Cursor()
			if cursor >= 0 && cursor < len(m.filteredContainers) {
				match := m.filteredContainers[cursor]
				m.state = stateDetail
				m.selectedDetail = nil
				m.selectedContainer = nil
				m.viewport.GotoTop()
				return m, m.fetchContainerDetail(match)
			}
		}
		if m.activeTab == tabProcesses && m.listFocus == focusMain {
			selected := m.table.SelectedRow()
			if len(selected) > 0 {
				pid := 0
				fmt.Sscanf(selected[0], "%d", &pid)
				if pid > 0 {
					m.state = stateDetail
					m.viewport.GotoTop()
					m.envViewport.GotoTop()
					return m, m.fetchProcessDetail(pid)
				}
			}
		} else if m.activeTab == tabProcesses && m.listFocus == focusSide {
			if m.treeCursor >= 0 && m.treeCursor < len(m.treePIDs) {
				pid := m.treePIDs[m.treeCursor]
				if pid > 0 {
					m.state = stateDetail
					m.viewport.GotoTop()
					m.envViewport.GotoTop()
					return m, m.fetchProcessDetail(pid)
				}
			}
		} else if m.activeTab == tabPorts {
			switch m.listFocus {
			case focusMain:
				m.listFocus = focusSide
				m.portTable.Blur()
				m.portDetailTable.Focus()
			case focusSide:
				if opened, openCmd, ok := m.openPortOwner(); ok {
					return opened, openCmd
				}
			}
		}

	// Focus Switching
	case "tab", "right", "left", "l", "L", "h", "H":
		if m.input.Focused() || m.portInput.Focused() || m.containerInput.Focused() || m.lockInput.Focused() {
			break
		}
		// Tabs without a side panel — these keys are no-ops there.
		if m.activeTab == tabContainers || m.activeTab == tabLocks {
			return m, nil
		}
		if msg.String() == "tab" || msg.String() == "right" || msg.String() == "l" || msg.String() == "L" {
			if m.listFocus == focusMain {
				m.listFocus = focusSide
				if m.activeTab == tabPorts {
					m.portTable.Blur()
					m.portDetailTable.Focus()
				}
			} else {
				m.listFocus = focusMain
				if m.activeTab == tabPorts {
					m.portDetailTable.Blur()
					m.portTable.Focus()
				}
			}
		} else if msg.String() == "shift+tab" || msg.String() == "left" || msg.String() == "h" || msg.String() == "H" {
			if m.listFocus == focusSide {
				m.listFocus = focusMain
				if m.activeTab == tabPorts {
					m.portDetailTable.Blur()
					m.portTable.Focus()
				}
			} else {
				m.listFocus = focusSide
				if m.activeTab == tabPorts {
					m.portTable.Blur()
					m.portDetailTable.Focus()
				}
			}
		}
		return m, nil

	// Process actions / Toggle All Ports / Toggle Open Files
	case "a", "A":
		if m.activeTab == tabProcesses && m.listFocus == focusMain {
			if idx := m.table.Cursor(); actionsSupported && idx >= 0 && idx < len(m.filtered) {
				m.openActionMenu(m.filtered[idx])
			}
			return m, nil
		}
		if m.activeTab == tabPorts {
			m.showAllPorts = !m.showAllPorts
			m.updatePortTable()
			return m, nil
		}
		if m.activeTab == tabLocks {
			m.showAllFiles = !m.showAllFiles
			m.locks = nil
			m.filteredLocks = nil
			m.lockTable.SetRows(nil)
			m.lockTable.SetCursor(0)
			return m, m.refreshLocks()
		}

	// Sorting Keys (union across all tabs; per-tab dispatch below picks the relevant ones)
	case "c", "C", "p", "P", "n", "N", "m", "M", "t", "T", "u", "U", "s", "S",
		"i", "I", "r", "R", "g", "G", "f", "F":
		if nm, cmd, handled := m.handleSortKey(msg); handled {
			return nm, cmd
		} else {
			m = nm
		}
	}

	return m.handleListNavKey(msg)
}

// actionActive reports whether the action menu, a confirmation prompt or the
// renice input is open.
func (m MainModel) actionActive() bool {
	return m.actionMenuOpen || m.pendingAction != actionNone
}

// openActionMenu opens the action menu for a snapshot of p.
func (m *MainModel) openActionMenu(p model.Process) {
	m.actionTarget = &p
	m.actionMenuOpen = true
}

// closeAction dismisses the action menu, prompt and renice input.
func (m *MainModel) closeAction() {
	m.actionMenuOpen = false
	m.pendingAction = actionNone
	m.actionTarget = nil
	m.reniceInput.SetValue("")
	m.reniceInput.Blur()
}

// targetChanged reports whether the live process for t.PID is no longer t —
// it exited, or the PID was reused. Identity is the PID plus its start time,
// so a signal or renice can't land on an unrelated process.
func targetChanged(t model.Process) bool {
	cur, err := proc.ReadProcess(t.PID)
	return err != nil || !cur.StartedAt.Equal(t.StartedAt)
}

func (m MainModel) handleActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.actionTarget == nil {
		m.closeAction()
		return m, nil
	}
	t := *m.actionTarget

	switch {
	case m.pendingAction == actionRenice:
		switch msg.String() {
		case "esc":
			m.closeAction()
		case "enter":
			val, verr := validateNiceValue(m.reniceInput.Value())
			switch {
			case verr != nil:
				m.statusMsg = "Invalid nice value — enter a number between −20 and 19"
			case targetChanged(t):
				m.statusMsg = fmt.Sprintf("PID %d changed since selected — refresh and retry", t.PID)
			default:
				if err := setNice(t.PID, val); err != nil {
					m.statusMsg = fmt.Sprintf("Renice failed: %v", err)
				} else {
					m.statusMsg = fmt.Sprintf("%s reniced to %d", actionTargetLabel(t), val)
				}
			}
			m.closeAction()
		default:
			var inputCmd tea.Cmd
			m.reniceInput, inputCmd = m.reniceInput.Update(msg)
			return m, inputCmd
		}
		return m, nil

	case m.pendingAction != actionNone:
		switch confirmKey(msg.String()) {
		case confirmExecute:
			return m.executeAction(t)
		case confirmCancel:
			m.closeAction()
		}
		return m, nil

	default:
		if pending, closeMenu := actionMenuSelect(msg.String()); closeMenu {
			if pending == actionNone {
				m.closeAction()
				return m, nil
			}
			m.actionMenuOpen = false
			m.pendingAction = pending
			if pending == actionRenice {
				m.reniceInput.Focus()
				return m, nil
			}
		}
		return m, nil
	}
}

// executeAction sends the confirmed signal to t.
func (m MainModel) executeAction(t model.Process) (tea.Model, tea.Cmd) {
	action := m.pendingAction
	m.closeAction()
	if targetChanged(t) {
		m.statusMsg = fmt.Sprintf("PID %d changed since selected — refresh and retry", t.PID)
		return m, nil
	}

	var err error
	switch action {
	case actionKill:
		err = killProcess(t.PID)
	case actionTerm:
		err = termProcess(t.PID)
	case actionPause:
		err = pauseProcess(t.PID)
	case actionResume:
		err = resumeProcess(t.PID)
	}
	if err != nil {
		m.statusMsg = fmt.Sprintf("Error: %v", err)
		return m, nil
	}
	m.statusMsg = fmt.Sprintf("Signal sent to %s", actionTargetLabel(t))
	if action == actionKill || action == actionTerm {
		// The process is gone, so leave its detail view for the list.
		m.state = stateList
		m.selectedDetail = nil
		return m, m.refreshProcesses()
	}
	return m, nil
}

func (m MainModel) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "Q", "backspace":
		m.state = stateList
		m.selectedDetail = nil
		m.selectedContainer = nil
		m.detailFocus = focusDetail
		if m.activeTab == tabContainers {
			return m, m.refreshContainers()
		}
		return m, m.refreshProcesses()
	case "a", "A":
		if actionsSupported && m.selectedDetail != nil {
			m.openActionMenu(m.selectedDetail.Process)
		}
		return m, nil
	case "left", "h", "H":
		m.detailFocus = focusDetail
		return m, nil
	case "right", "l", "L":
		m.detailFocus = focusEnv
		return m, nil
	case "tab":
		if m.detailFocus == focusDetail {
			m.detailFocus = focusEnv
		} else {
			m.detailFocus = focusDetail
		}
		return m, nil
	default:
		var cmd tea.Cmd
		if m.detailFocus == focusDetail {
			m.viewport, cmd = m.viewport.Update(msg)
		} else {
			m.envViewport, cmd = m.envViewport.Update(msg)
		}
		return m, cmd
	}
}

func (m MainModel) handleListFilterInput(msg tea.KeyMsg) (MainModel, tea.Cmd, bool) {
	if m.activeTab == tabLocks {
		if m.lockInput.Focused() {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.lockInput.Blur()
				return m, nil, true
			}
			if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
				m.lockInput.Blur()
			} else {
				var inputCmd tea.Cmd
				m.lockInput, inputCmd = m.lockInput.Update(msg)
				m.updateLockTable()
				m.lockTable.SetCursor(0)
				return m, inputCmd, true
			}
		}

		if msg.String() == "/" {
			m.lockInput.Focus()
			return m, nil, true
		}
	} else if m.activeTab == tabContainers {
		if m.containerInput.Focused() {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.containerInput.Blur()
				return m, nil, true
			}
			if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
				m.containerInput.Blur()
			} else {
				var inputCmd tea.Cmd
				m.containerInput, inputCmd = m.containerInput.Update(msg)
				m.updateContainerTable()
				m.containerTable.SetCursor(0)
				return m, inputCmd, true
			}
		}

		if msg.String() == "/" {
			m.containerInput.Focus()
			return m, nil, true
		}
	} else if m.activeTab == tabPorts {
		if m.portInput.Focused() {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.portInput.Blur()
				return m, nil, true
			}
			if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
				m.portInput.Blur()
			} else {
				var inputCmd tea.Cmd
				m.portInput, inputCmd = m.portInput.Update(msg)
				m.exactPort = false
				m.updatePortTable()
				m.portTable.SetCursor(0)
				return m, inputCmd, true
			}
		}

		if msg.String() == "/" {
			m.portInput.Focus()
			return m, nil, true
		}
	} else {
		if m.input.Focused() {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.input.Blur()
				return m, nil, true
			}
			if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
				m.input.Blur()
			} else {
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				m.exactName = false
				m.filterProcesses()

				m.table.SetCursor(0)
				var treeCmd tea.Cmd
				if len(m.filtered) > 0 {
					selected := m.table.SelectedRow()
					if len(selected) > 0 {
						pid := 0
						fmt.Sscanf(selected[0], "%d", &pid)
						m.selectionID++
						id := m.selectionID
						treeCmd = tea.Tick(selectionDebounce, func(_ time.Time) tea.Msg {
							return debounceMsg{id: id, pid: pid}
						})
					}
				} else {
					m.treeViewport.SetContent("")
				}
				return m, tea.Batch(inputCmd, treeCmd), true
			}
		}

		if msg.String() == "/" {
			m.input.Focus()
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m MainModel) handleSortKey(msg tea.KeyMsg) (MainModel, tea.Cmd, bool) {
	switch m.activeTab {
	case tabProcesses:
		newCol := ""
		switch msg.String() {
		case "c", "C":
			newCol = "cpu"
		case "p", "P":
			newCol = "pid"
		case "n", "N":
			newCol = "name"
		case "m", "M":
			newCol = "mem"
		case "t", "T":
			newCol = "time"
		case "u", "U":
			newCol = "user"
		}

		if newCol != "" {
			if m.sortCol == newCol {
				m.sortDesc = !m.sortDesc
			} else {
				m.sortCol = newCol
				m.sortDesc = true
			}
			m.sortProcesses()
			m.filterProcesses()
			cols := m.table.Columns()
			newCols := m.getColumns()
			for i := range cols {
				if i < len(newCols) {
					newCols[i].Width = cols[i].Width
				}
			}
			m.table.SetColumns(newCols)
			return m, nil, true
		}
	case tabPorts:
		newCol := ""
		switch msg.String() {
		case "p", "P":
			newCol = "port"
		case "t", "T":
			newCol = "proto"
		case "n", "N":
			newCol = "addr"
		case "s", "S":
			newCol = "state"
		}

		if newCol != "" {
			if m.sortPortCol == newCol {
				m.sortPortDesc = !m.sortPortDesc
			} else {
				m.sortPortCol = newCol
				m.sortPortDesc = false
			}
			m.updatePortTable()
			return m, nil, true
		}

	case tabContainers:
		newCol := ""
		switch msg.String() {
		case "i", "I":
			newCol = "id"
		case "n", "N":
			newCol = "name"
		case "r", "R":
			newCol = "runtime"
		case "g", "G":
			newCol = "image"
		case "s", "S":
			newCol = "status"
		}
		if newCol != "" {
			if m.sortContainerCol == newCol {
				m.sortContainerDesc = !m.sortContainerDesc
			} else {
				m.sortContainerCol = newCol
				m.sortContainerDesc = false
			}
			m.updateContainerTable()
			return m, nil, true
		}

	case tabLocks:
		newCol := ""
		switch msg.String() {
		case "p", "P":
			newCol = "pid"
		case "n", "N":
			newCol = "process"
		case "t", "T":
			newCol = "type"
		case "m", "M":
			newCol = "mode"
		case "f", "F":
			newCol = "path"
		}
		if newCol != "" {
			if m.sortLockCol == newCol {
				m.sortLockDesc = !m.sortLockDesc
			} else {
				m.sortLockCol = newCol
				m.sortLockDesc = false
			}
			m.updateLockTable()
			return m, nil, true
		}
	}
	return m, nil, false
}

func (m MainModel) handleListNavKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if m.listFocus == focusMain {
		if m.activeTab == tabProcesses {
			prevSelected := -1
			if len(m.filtered) > 0 {
				prevSelected = m.table.Cursor()
			}

			m.table, cmd = m.table.Update(msg)

			if len(m.filtered) > 0 && m.table.Cursor() != prevSelected {
				selected := m.table.SelectedRow()
				if len(selected) > 0 {
					idx := m.table.Cursor()
					if idx >= 0 && idx < len(m.filtered) {
						m.selectionID++
						id := m.selectionID
						p := m.filtered[idx]
						debounceCmd := tea.Tick(selectionDebounce, func(_ time.Time) tea.Msg {
							return debounceMsg{id: id, pid: p.PID}
						})
						return m, tea.Batch(cmd, debounceCmd)
					}
				}
			}
			return m, cmd
		} else if m.activeTab == tabContainers {
			m.containerTable, cmd = m.containerTable.Update(msg)
			return m, cmd
		} else if m.activeTab == tabLocks {
			m.lockTable, cmd = m.lockTable.Update(msg)
			return m, cmd
		} else {
			prevSelected := m.portTable.Cursor()
			m.portTable, cmd = m.portTable.Update(msg)
			if m.portTable.Cursor() != prevSelected {
				m.updatePortDetails()
			}
			return m, cmd
		}
	} else {
		if m.activeTab == tabProcesses {
			switch msg.Type {
			case tea.KeyUp:
				if m.treeCursor > 0 {
					m.treeCursor--
					m.rerenderTree()
				}
			case tea.KeyDown:
				if m.treeCursor < len(m.treePIDs)-1 {
					m.treeCursor++
					m.rerenderTree()
				}
			case tea.KeyEnter:
				if m.treeCursor >= 0 && m.treeCursor < len(m.treePIDs) {
					pid := m.treePIDs[m.treeCursor]
					if pid > 0 {
						m.state = stateDetail
						m.viewport.GotoTop()
						m.envViewport.GotoTop()
						return m, m.fetchProcessDetail(pid)
					}
				}
			default:
				m.treeViewport, cmd = m.treeViewport.Update(msg)
			}
		} else {
			m.portDetailTable, cmd = m.portDetailTable.Update(msg)
		}
		return m, cmd
	}
}
