package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wrap"
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
	case model.NetworkSnapshot:
		return m.handleNetworkSnapshot(msg)
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
	if m.state == stateList && !m.quitting && !m.input.Focused() && !m.portInput.Focused() && !m.containerInput.Focused() && !m.lockInput.Focused() && !m.networkInput.Focused() && m.refreshDue() {
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
		case tabNetwork:
			cmd = tea.Batch(cmd, m.refreshNetworks())
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
		if m.networkInput.Focused() {
			m.networkInput.Blur()
		}
	}

	// Tabs. Ranges are derived from inactive tab label widths (padding 0,1):
	// "1. Processes"=14, "2. Ports"=10, "3. Containers"=15, "4. Locks"=10,
	// "5. Network"=12. Title "witr" starts at X≈2 with brand padding → content
	// from X=8. When Locks is hidden (Windows), Network shifts left.
	if msg.Y == 1 && isClick {
		x := msg.X
		// Brand title ~6 cells starting at 2 → tabs from 8.
		cursor := 8
		tabSpans := []struct {
			tab   tab
			width int
		}{
			{tabProcesses, 14},
			{tabPorts, 10},
			{tabContainers, 15},
		}
		if locksTabEnabled {
			tabSpans = append(tabSpans, struct {
				tab   tab
				width int
			}{tabLocks, 10})
		}
		tabSpans = append(tabSpans, struct {
			tab   tab
			width int
		}{tabNetwork, 12})

		for _, ts := range tabSpans {
			if x >= cursor && x < cursor+ts.width {
				if m.activeTab != ts.tab {
					m.activeTab = ts.tab
					m.listFocus = focusMain
					switch ts.tab {
					case tabPorts:
						return m, m.refreshPorts()
					case tabContainers:
						return m, m.refreshContainers()
					case tabLocks:
						return m, m.refreshLocks()
					case tabNetwork:
						return m, m.refreshNetworks()
					}
				}
				return m, nil
			}
			cursor += ts.width
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
		case tabNetwork:
			m.networkInput.Focus()
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
		case tabNetwork:
			return m.handleNetworkAreaMouse(msg, contentX, isClick, isWheel, isDoubleClick)
		}
	}
	return m, nil
}

func (m MainModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.statusMsg = "" // clear any transient error on interaction
	switch msg.String() {
	case "ctrl+c":
		m.quitting = true
		return m, tea.Quit
	case "1":
		if !m.anySearchFocused() {
			m.activeTab = tabProcesses
			m.listFocus = focusMain
			return m, nil
		}
	case "2":
		if !m.anySearchFocused() {
			m.activeTab = tabPorts
			m.listFocus = focusMain
			m.portTable.Focus()
			m.portDetailTable.Blur()
			return m, m.refreshPorts()
		}
	case "3":
		if !m.anySearchFocused() {
			m.activeTab = tabContainers
			m.listFocus = focusMain
			return m, m.refreshContainers()
		}
	case "4":
		if locksTabEnabled && !m.anySearchFocused() {
			m.activeTab = tabLocks
			m.listFocus = focusMain
			return m, m.refreshLocks()
		}
	case "5":
		if !m.anySearchFocused() {
			m.activeTab = tabNetwork
			m.listFocus = focusMain
			m.networkTable.Focus()
			m.networkDetailTable.Blur()
			return m, m.refreshNetworks()
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

	fixedColumnsWidth := 81 // PID(8)+Name(20)+User(12)+CPU(6)+Mem(16)+Started(19)
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

	// Network split pane (Host + Docker combined table).
	netPaneWidth := int(float64(availableWidth) * networkPaneRatio)
	if netPaneWidth < 0 {
		netPaneWidth = 0
	}
	netTableWidth := netPaneWidth - tablePadding
	if netTableWidth < 0 {
		netTableWidth = 0
	}
	netCols := m.getNetworkColumns()
	// Stretch Address column.
	fixedNet := 0
	addrIdx := 4
	for i, c := range netCols {
		if i == addrIdx {
			continue
		}
		fixedNet += c.Width
	}
	if addrIdx < len(netCols) {
		sw := netTableWidth - fixedNet - cellPadding*len(netCols)
		if sw < 10 {
			sw = 10
		}
		netCols[addrIdx].Width = sw
	}
	m.networkTable.SetColumns(netCols)
	m.networkTable.SetWidth(netTableWidth)
	m.networkTable.SetHeight(processListHeight)

	netDetailWidth := availableWidth - netPaneWidth - 5
	if netDetailWidth < 10 {
		netDetailWidth = 10
	}
	m.ensureNetworkDetailKVColumns()
	ndCols := m.networkDetailTable.Columns()
	fixedNd := 0
	flexIdx := 1
	for i, c := range ndCols {
		if i == flexIdx {
			continue
		}
		fixedNd += c.Width
	}
	if flexIdx < len(ndCols) {
		nw := netDetailWidth - fixedNd - cellPadding*len(ndCols)
		if nw < 10 {
			nw = 10
		}
		ndCols[flexIdx].Width = nw
	}
	m.networkDetailTable.SetColumns(ndCols)
	m.networkDetailTable.SetWidth(netDetailWidth)
	m.networkDetailTable.SetHeight(processListHeight - 2)

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
	// Rebuild network rows after column widths change so HOST/NETWORKS data
	// is not left on stale empty rows from before the first WindowSizeMsg.
	if len(m.hostIfaces) > 0 || len(m.networks) > 0 {
		m.updateNetworkTable()
	} else {
		m.updateNetworkDetails()
	}
	return m, nil
}

func (m MainModel) anySearchFocused() bool {
	return m.input.Focused() || m.portInput.Focused() || m.containerInput.Focused() || m.lockInput.Focused() || m.networkInput.Focused()
}

func (m MainModel) handleProcessList(msg []model.Process) (tea.Model, tea.Cmd) {
	// Feed this background refresh's measured duration into the adaptive cadence.
	if !m.refreshStartedAt.IsZero() {
		took := time.Since(m.refreshStartedAt)
		m.refreshStartedAt = time.Time{}
		m.refreshEvery, m.slowStreak, m.fastStreak = adjustRefreshInterval(m.refreshEvery, took, m.slowStreak, m.fastStreak)
	}

	var currentPID int
	selectedRow := m.table.SelectedRow()
	if len(selectedRow) > 0 {
		fmt.Sscanf(selectedRow[0], "%d", &currentPID)
	}

	m.processes = msg
	m.sortProcesses()
	m.filterProcesses()

	newIdx := 0
	found := false
	if currentPID > 0 {
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
		m.table.SetCursor(newIdx)

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

func (m MainModel) handleNetworkSnapshot(msg model.NetworkSnapshot) (tea.Model, tea.Cmd) {
	// Preserve selection by source+name across refresh.
	var selSource, selName string
	if e, ok := m.selectedNetworkEntry(); ok {
		if e.IsHost {
			selSource, selName = "Host", e.Host.Name
		} else {
			selSource, selName = "Docker", e.Net.Name
		}
	}

	m.networks = msg.Networks
	m.hostIfaces = msg.HostIfaces
	m.vethByBridge = msg.VethByBridge
	m.networkHostSrc = msg.HostSource
	m.networkErr = msg.Error
	if !msg.DockerOK && msg.Error == "" {
		m.networkErr = "Docker not available"
	}
	m.updateNetworkTable()

	if selName != "" {
		for i, e := range m.filteredEntries {
			src, name := "Docker", e.Net.Name
			if e.IsHost {
				src, name = "Host", e.Host.Name
			}
			if src == selSource && name == selName {
				m.networkTable.SetCursor(i)
				break
			}
		}
		m.updateNetworkDetails()
	}
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
	m.selectedNetwork = nil
	m.updateDetailViewport()
	m.updateEnvViewport()
	return m, nil
}

func (m MainModel) handleContainerDetail(msg *model.ContainerMatch) (tea.Model, tea.Cmd) {
	m.selectedContainer = msg
	m.selectedDetail = nil
	m.selectedNetwork = nil
	m.updateDetailViewport()
	return m, nil
}

func (m MainModel) handleError(msg error) (tea.Model, tea.Cmd) {
	// Revert to list view on any error
	m.state = stateList
	m.selectedDetail = nil
	m.selectedContainer = nil
	m.selectedNetwork = nil
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
			treeY := msg.Y - 10
			treeY += m.treeViewport.YOffset
			if treeY >= 0 && treeY < len(m.treePIDs) {
				m.treeCursor = treeY
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

func (m MainModel) handleNetworkAreaMouse(msg tea.MouseMsg, contentX int, isClick, isWheel, isDoubleClick bool) (tea.Model, tea.Cmd) {
	availableWidth := m.width - 6
	netPaneWidth := int(float64(availableWidth) * networkPaneRatio)

	if contentX < netPaneWidth {
		if isClick {
			m.listFocus = focusMain
			m.networkDetailTable.Blur()
			m.networkTable.Focus()
			if msg.Y == 7 {
				m.handleNetworkHeaderClick(contentX)
				return m, nil
			}
		}

		var cmd tea.Cmd
		prevSelected := m.networkTable.Cursor()
		if isWheel {
			var keyMsg tea.KeyMsg
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				keyMsg = tea.KeyMsg{Type: tea.KeyUp}
			case tea.MouseButtonWheelDown:
				keyMsg = tea.KeyMsg{Type: tea.KeyDown}
			}
			m.networkTable, cmd = m.networkTable.Update(keyMsg)
			if m.networkTable.Cursor() != prevSelected {
				m.updateNetworkDetails()
			}
			return m, cmd
		}

		netMsg := msg
		netMsg.X -= 2
		netMsg.Y -= 7
		if netMsg.X >= 0 && netMsg.Y >= 0 {
			m.networkTable, cmd = m.networkTable.Update(netMsg)
		}
		if m.networkTable.Cursor() != prevSelected {
			m.updateNetworkDetails()
		}
		if isDoubleClick && isClick && netMsg.Y > 0 {
			if e, ok := m.selectedNetworkEntry(); ok {
				m.state = stateDetail
				m.selectedDetail = nil
				m.selectedContainer = nil
				if e.IsHost {
					m.selectedNetwork = nil
				} else {
					net := e.Net
					m.selectedNetwork = &net
				}
				if w := m.width - 6; w > 0 {
					m.viewport.Width = w
				}
				content := m.networkDetailContent()
				if m.viewport.Width > 0 {
					content = wrap.String(content, m.viewport.Width)
				}
				m.viewport.SetContent(content)
				m.viewport.GotoTop()
				return m, nil
			}
		}
		return m, cmd
	}

	// Side panel (interface details or attached containers)
	if isClick {
		m.listFocus = focusSide
		m.networkTable.Blur()
		m.networkDetailTable.Focus()
	}
	var cmd tea.Cmd
	if isWheel {
		var keyMsg tea.KeyMsg
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			keyMsg = tea.KeyMsg{Type: tea.KeyUp}
		case tea.MouseButtonWheelDown:
			keyMsg = tea.KeyMsg{Type: tea.KeyDown}
		}
		m.networkDetailTable, cmd = m.networkDetailTable.Update(keyMsg)
		return m, cmd
	}
	sideMsg := msg
	sideMsg.X -= netPaneWidth + 2
	sideMsg.Y -= 8
	if sideMsg.X >= 0 && sideMsg.Y >= 0 {
		m.networkDetailTable, cmd = m.networkDetailTable.Update(sideMsg)
	}
	return m, cmd
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
			selected := m.portDetailTable.SelectedRow()
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
					m.selectedNetwork = nil
					m.viewport.GotoTop()
					m.envViewport.GotoTop()
					return m, m.fetchProcessDetail(pid)
				}
			}
		}
		if m.activeTab == tabNetwork {
			if e, ok := m.selectedNetworkEntry(); ok {
				m.state = stateDetail
				m.selectedDetail = nil
				m.selectedContainer = nil
				if e.IsHost {
					m.selectedNetwork = nil
				} else {
					net := e.Net
					m.selectedNetwork = &net
				}
				if w := m.width - 6; w > 0 {
					m.viewport.Width = w
				}
				content := m.networkDetailContent()
				if m.viewport.Width > 0 {
					content = wrap.String(content, m.viewport.Width)
				}
				m.viewport.SetContent(content)
				m.viewport.GotoTop()
				return m, nil
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
				selected := m.portDetailTable.SelectedRow()
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
			}
		}

	// Focus Switching
	case "tab", "right", "left", "l", "L", "h", "H":
		if m.anySearchFocused() {
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
				if m.activeTab == tabNetwork {
					m.networkTable.Blur()
					m.networkDetailTable.Focus()
				}
			} else {
				m.listFocus = focusMain
				if m.activeTab == tabPorts {
					m.portDetailTable.Blur()
					m.portTable.Focus()
				}
				if m.activeTab == tabNetwork {
					m.networkDetailTable.Blur()
					m.networkTable.Focus()
				}
			}
		} else if msg.String() == "shift+tab" || msg.String() == "left" || msg.String() == "h" || msg.String() == "H" {
			if m.listFocus == focusSide {
				m.listFocus = focusMain
				if m.activeTab == tabPorts {
					m.portDetailTable.Blur()
					m.portTable.Focus()
				}
				if m.activeTab == tabNetwork {
					m.networkDetailTable.Blur()
					m.networkTable.Focus()
				}
			} else {
				m.listFocus = focusSide
				if m.activeTab == tabPorts {
					m.portTable.Blur()
					m.portDetailTable.Focus()
				}
				if m.activeTab == tabNetwork {
					m.networkTable.Blur()
					m.networkDetailTable.Focus()
				}
			}
		}
		return m, nil

	// Toggle All Ports / open files / host ifaces
	case "a", "A":
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
		if m.activeTab == tabNetwork {
			// Cycle ALL → HOST → DOCKER → ALL
			switch m.networkScope {
			case networkScopeAll:
				m.networkScope = networkScopeHost
			case networkScopeHost:
				m.networkScope = networkScopeDocker
			default:
				m.networkScope = networkScopeAll
			}
			m.listFocus = focusMain
			m.networkTable.Focus()
			m.networkDetailTable.Blur()
			m.networkTable.SetCursor(0)
			m.updateNetworkTable()
			return m, nil
		}

	// Sorting Keys (union across all tabs; per-tab dispatch below picks the relevant ones)
	case "c", "C", "p", "P", "n", "N", "m", "M", "t", "T", "u", "U", "s", "S",
		"i", "I", "r", "R", "g", "G", "f", "F", "d", "D", "b", "B", "k", "K":
		if nm, cmd, handled := m.handleSortKey(msg); handled {
			return nm, cmd
		} else {
			m = nm
		}
	}

	return m.handleListNavKey(msg)
}

// pidIdentityChanged reports whether the live process for pid no longer matches
// the one captured in selectedDetail — i.e. it exited and the PID was recycled.
// It is the shared guard the destructive actions use so a signal or renice can't
// land on an unrelated process. Returns false when there is no snapshot to
// compare against.
func (m MainModel) pidIdentityChanged(pid int) bool {
	if m.selectedDetail == nil {
		return false
	}
	cur, err := proc.ReadProcess(pid)
	return err != nil || !cur.StartedAt.Equal(m.selectedDetail.Process.StartedAt)
}

func (m MainModel) handleDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	pid := 0
	if m.selectedDetail != nil {
		pid = m.selectedDetail.Process.PID
	}

	// renice text input
	if m.pendingAction == actionRenice {
		switch msg.String() {
		case "esc":
			m.pendingAction = actionNone
			m.reniceInput.SetValue("")
			m.reniceInput.Blur()
		case "enter":
			val, verr := validateNiceValue(m.reniceInput.Value())
			switch {
			case verr != nil:
				m.statusMsg = "Invalid nice value — enter a number between −20 and 19"
			case m.pidIdentityChanged(pid):
				m.statusMsg = fmt.Sprintf("PID %d changed since opened — refresh and retry", pid)
			default:
				if err := setNice(pid, val); err != nil {
					m.statusMsg = fmt.Sprintf("Renice failed: %v", err)
				} else {
					m.statusMsg = fmt.Sprintf("PID %d reniced to %d", pid, val)
				}
			}
			m.pendingAction = actionNone
			m.reniceInput.SetValue("")
			m.reniceInput.Blur()
		default:
			var inputCmd tea.Cmd
			m.reniceInput, inputCmd = m.reniceInput.Update(msg)
			return m, inputCmd
		}
		return m, nil
	}

	// confirmation prompt
	if m.pendingAction != actionNone {
		switch confirmKey(msg.String()) {
		case confirmExecute:
			// Re-validate the target: if the process exited and its PID
			// was reused since the detail view opened, refuse to signal a
			// different process. Identity is the PID plus its start time.
			if m.pidIdentityChanged(pid) {
				m.pendingAction = actionNone
				m.statusMsg = fmt.Sprintf("PID %d changed since opened — refresh and retry", pid)
				return m, nil
			}
			originalAction := m.pendingAction
			var execErr error
			switch originalAction {
			case actionKill:
				execErr = killProcess(pid)
			case actionTerm:
				execErr = termProcess(pid)
			case actionPause:
				execErr = pauseProcess(pid)
			case actionResume:
				execErr = resumeProcess(pid)
			}
			m.pendingAction = actionNone
			if execErr != nil {
				m.statusMsg = fmt.Sprintf("Error: %v", execErr)
				return m, nil
			}
			switch originalAction {
			case actionKill, actionTerm:
				// Process is gone — go back to list
				m.state = stateList
				m.selectedDetail = nil
				m.statusMsg = fmt.Sprintf("Signal sent to PID %d", pid)
				return m, m.refreshProcesses()
			default:
				// Pause/Resume succeeded — stay in detail view
				m.statusMsg = "Done"
				return m, nil
			}
		case confirmCancel:
			m.pendingAction = actionNone
		}
		return m, nil
	}

	// action menu
	if m.actionMenuOpen {
		if pending, closeMenu := actionMenuSelect(msg.String()); closeMenu {
			m.actionMenuOpen = false
			m.pendingAction = pending
			if pending == actionRenice {
				m.reniceInput.Focus()
				return m, textinput.Blink
			}
		}
		return m, nil
	}

	// Network and container detail are single-pane (no env split).
	// Host network detail uses selectedNetwork==nil but stateDetail + viewport content.
	if m.selectedNetwork != nil || m.selectedContainer != nil || (m.activeTab == tabNetwork && m.state == stateDetail && m.selectedDetail == nil) {
		switch msg.String() {
		case "esc", "q", "Q", "backspace":
			m.state = stateList
			m.selectedDetail = nil
			m.selectedContainer = nil
			m.selectedNetwork = nil
			if m.activeTab == tabNetwork {
				return m, m.refreshNetworks()
			}
			if m.activeTab == tabContainers {
				return m, m.refreshContainers()
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}
	}

	// detail view navigation
	switch msg.String() {
	case "esc", "q", "Q", "backspace":
		m.state = stateList
		m.selectedDetail = nil
		m.selectedContainer = nil
		m.selectedNetwork = nil
		m.detailFocus = focusDetail
		m.actionMenuOpen = false
		m.pendingAction = actionNone
		m.reniceInput.SetValue("")
		m.reniceInput.Blur()
		if m.activeTab == tabContainers {
			return m, m.refreshContainers()
		}
		if m.activeTab == tabNetwork {
			return m, m.refreshNetworks()
		}
		return m, m.refreshProcesses()
	case "a", "A":
		if actionsSupported && m.selectedDetail != nil {
			m.actionMenuOpen = true
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
	if m.activeTab == tabNetwork {
		if m.networkInput.Focused() {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.networkInput.Blur()
				return m, nil, true
			}
			if msg.Type == tea.KeyUp || msg.Type == tea.KeyDown {
				m.networkInput.Blur()
			} else {
				var inputCmd tea.Cmd
				m.networkInput, inputCmd = m.networkInput.Update(msg)
				m.updateNetworkTable()
				m.networkTable.SetCursor(0)
				return m, inputCmd, true
			}
		}
		if msg.String() == "/" {
			m.networkInput.Focus()
			return m, textinput.Blink, true
		}
	} else if m.activeTab == tabLocks {
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
			return m, textinput.Blink, true
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
			return m, textinput.Blink, true
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
				m.updatePortTable()
				m.portTable.SetCursor(0)
				return m, inputCmd, true
			}
		}

		if msg.String() == "/" {
			m.portInput.Focus()
			return m, textinput.Blink, true
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
			return m, textinput.Blink, true
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

	case tabNetwork:
		newCol := ""
		switch msg.String() {
		case "s", "S":
			newCol = "source"
		case "n", "N":
			newCol = "name"
		case "k", "K":
			newCol = "kind"
		case "t", "T":
			newCol = "state"
		case "d", "D":
			newCol = "address"
		case "g", "G":
			newCol = "extra"
		}
		if newCol != "" {
			if m.sortNetworkCol == newCol {
				m.sortNetworkDesc = !m.sortNetworkDesc
			} else {
				m.sortNetworkCol = newCol
				m.sortNetworkDesc = false
			}
			m.updateNetworkTable()
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
		} else if m.activeTab == tabNetwork {
			prevSelected := m.networkTable.Cursor()
			m.networkTable, cmd = m.networkTable.Update(msg)
			if m.networkTable.Cursor() != prevSelected {
				m.updateNetworkDetails()
			}
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
		} else if m.activeTab == tabNetwork {
			m.networkDetailTable, cmd = m.networkDetailTable.Update(msg)
		} else {
			m.portDetailTable, cmd = m.portDetailTable.Update(msg)
		}
		return m, cmd
	}
}
