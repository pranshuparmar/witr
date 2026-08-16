package tui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuparmar/witr/internal/output"
)

func (m MainModel) View() string {
	if m.quitting {
		return ""
	}

	outerStyle := baseStyle.
		Width(m.width-2).
		Height(m.height-2).
		Padding(0, 1)

	switch m.state {
	case stateList:
		return m.viewList(outerStyle)
	case stateDetail:
		if m.selectedDetail == nil && m.selectedContainer == nil && m.selectedNetwork == nil {
			// Host network detail reuses the network detail view with viewport content set.
			if m.activeTab == tabNetwork {
				return m.viewNetworkDetail(outerStyle)
			}
			return m.viewDetailLoading(outerStyle)
		}
		if m.selectedNetwork != nil || (m.activeTab == tabNetwork && m.selectedDetail == nil && m.selectedContainer == nil) {
			return m.viewNetworkDetail(outerStyle)
		}
		if m.selectedContainer != nil {
			return m.viewContainerDetail(outerStyle)
		}
		return m.viewProcessDetail(outerStyle)
	}
	return "Unknown state"
}

func (m MainModel) viewList(outerStyle lipgloss.Style) string {
	status := "Mode: Navigation (Press / to search)"
	if m.statusMsg != "" {
		status = errorStyle.Render(m.statusMsg)
	}
	inputView := m.input.View()

	switch m.activeTab {
	case tabPorts:
		if m.portInput.Focused() {
			status = "Mode: Searching (↑↓ to navigate, Esc/Enter to stop)"
		}
		inputView = m.portInput.View()
	case tabContainers:
		if m.containerInput.Focused() {
			status = "Mode: Searching (↑↓ to navigate, Esc/Enter to stop)"
		}
		inputView = m.containerInput.View()
	case tabLocks:
		if m.lockInput.Focused() {
			status = "Mode: Searching (↑↓ to navigate, Esc/Enter to stop)"
		}
		inputView = m.lockInput.View()
	case tabNetwork:
		if m.networkInput.Focused() {
			status = "Mode: Searching (↑↓ to navigate, Esc/Enter to stop)"
		}
		inputView = m.networkInput.View()
	default:
		if m.input.Focused() {
			status = "Mode: Searching (↑↓ to navigate, Esc/Enter to stop)"
		}
	}

	activeBorderColor := colorAccent
	dimBorderColor := colorBorderDim

	treeBorderColor := dimBorderColor
	treeHeaderColor := colorHeaderDim

	if m.listFocus == focusSide {
		treeBorderColor = activeBorderColor
		treeHeaderColor = activeBorderColor
	}

	treeContainerStyle := paneDividerStyle.
		BorderForeground(treeBorderColor).
		Height(m.table.Height())

	treeHeader := "Details"
	selected := m.table.SelectedRow()
	if len(selected) > 0 {
		treeHeader = fmt.Sprintf("PID %s", strings.TrimSpace(selected[0]))
	}

	if !m.treeViewport.AtTop() && !m.treeViewport.AtBottom() {
		treeHeader += " ↕"
	} else if !m.treeViewport.AtTop() {
		treeHeader += " ↑"
	} else if !m.treeViewport.AtBottom() {
		treeHeader += " ↓"
	}

	treeHeaderStyle := tableHeaderStyle.
		Width(m.treeViewport.Width).
		Foreground(treeHeaderColor).
		BorderForeground(treeBorderColor)

	s := cachedTableStyles
	if m.listFocus == focusMain {
		s.Header = tableHeaderStyle.BorderForeground(activeBorderColor)
	} else {
		s.Header = tableHeaderStyle.BorderForeground(dimBorderColor)
	}
	m.table.SetStyles(s)

	availableWidth := m.width - 6
	processListPaneWidth := int(float64(availableWidth) * listPaneRatio)
	if processListPaneWidth < 10 {
		processListPaneWidth = 10
	}

	mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(processListPaneWidth).Render(m.table.View()),
		treeContainerStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				treeHeaderStyle.Render(treeHeader),
				paddedStyle.Render(m.treeViewport.View()),
			),
		),
	)

	if m.activeTab == tabContainers {
		s1 := cachedTableStyles
		s1.Header = s.Header
		m.containerTable.SetStyles(s1)
		mainContent = lipgloss.NewStyle().Width(m.width - 4).Render(m.containerTable.View())
	}

	if m.activeTab == tabLocks {
		s1 := cachedTableStyles
		s1.Header = s.Header
		m.lockTable.SetStyles(s1)
		mainContent = lipgloss.NewStyle().Width(m.width - 4).Render(m.lockTable.View())
	}

	if m.activeTab == tabNetwork {
		s1 := cachedTableStyles
		s1.Header = s.Header
		m.networkTable.SetStyles(s1)

		sideBorderColor := dimBorderColor
		sideHeaderColor := colorHeaderDim
		if m.listFocus == focusSide {
			sideBorderColor = activeBorderColor
			sideHeaderColor = activeBorderColor
		}
		s2 := cachedTableStyles
		if m.listFocus == focusSide {
			s2.Header = tableHeaderStyle.BorderForeground(activeBorderColor)
		} else {
			s2.Header = tableHeaderStyle.BorderForeground(dimBorderColor)
		}
		m.networkDetailTable.SetStyles(s2)

		detailContainerStyle := detailDividerStyle.
			BorderForeground(sideBorderColor).
			Height(m.networkTable.Height())

		detailHeader := "Details"
		if e, ok := m.selectedNetworkEntry(); ok {
			if e.IsHost {
				detailHeader = "Host Interface"
			} else {
				detailHeader = "Docker Network"
			}
		}
		availableWidth := m.width - 6
		netPaneWidth := int(float64(availableWidth) * networkPaneRatio)
		headerWidth := availableWidth - netPaneWidth - 3

		detailHeaderStyle := tableHeaderStyle.
			Width(headerWidth).
			Foreground(sideHeaderColor).
			BorderForeground(sideBorderColor)

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(netPaneWidth).Render(m.networkTable.View()),
			detailContainerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					detailHeaderStyle.Render(detailHeader),
					m.networkDetailTable.View(),
				),
			),
		)
	}

	if m.activeTab == tabPorts {
		sideBorderColor := dimBorderColor
		sideHeaderColor := colorHeaderDim

		if m.listFocus == focusSide {
			sideBorderColor = activeBorderColor
			sideHeaderColor = activeBorderColor
		}

		// Reuse cached styles — only header border differs per table
		s1 := cachedTableStyles
		s1.Header = s.Header // same focus state as main table
		m.portTable.SetStyles(s1)

		s2 := cachedTableStyles
		if m.listFocus == focusSide {
			s2.Header = tableHeaderStyle.BorderForeground(activeBorderColor)
		} else {
			s2.Header = tableHeaderStyle.BorderForeground(dimBorderColor)
		}
		m.portDetailTable.SetStyles(s2)

		detailContainerStyle := detailDividerStyle.
			BorderForeground(sideBorderColor).
			Height(m.portTable.Height())

		detailHeader := "Attached Processes"
		availableWidth := m.width - 6
		portPaneWidth := int(float64(availableWidth) * portPaneRatio)
		headerWidth := availableWidth - portPaneWidth - 3

		detailHeaderStyle := tableHeaderStyle.
			Width(headerWidth).
			Foreground(sideHeaderColor).
			BorderForeground(sideBorderColor)

		mainContent = lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(portPaneWidth).Render(m.portTable.View()),
			detailContainerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					detailHeaderStyle.Render(detailHeader),
					m.portDetailTable.View(),
				),
			),
		)
	}

	helpText := fmt.Sprintf("Total: %d | Enter: Detail | p/n/u/c/m/t: Sort | Esc/q: Quit | Tab: Focus | Up/Down: Scroll", len(m.filtered))
	switch m.activeTab {
	case tabPorts:
		filterStatus := "LISTEN"
		if m.showAllPorts {
			filterStatus = "ALL"
		}
		helpText = fmt.Sprintf("Total: %d [%s] | p/t/n/s: Sort | a: Toggle All | Esc/q: Quit | Tab: Focus | Up/Down: Scroll", len(m.portTable.Rows()), filterStatus)
	case tabContainers:
		helpText = fmt.Sprintf("Total: %d | Enter: Detail | i/n/r/g/s: Sort | /: Search | Esc/q: Quit | Up/Down: Scroll", len(m.containerTable.Rows()))
	case tabLocks:
		suffix := ""
		if os.Geteuid() != 0 {
			suffix = " | (use sudo for full paths)"
		}
		mode := "LOCKED"
		if m.showAllFiles {
			mode = "OPEN"
		}
		shown := len(m.lockTable.Rows())
		total := len(m.filteredLocks)
		countText := fmt.Sprintf("Total: %d", total)
		if shown < total {
			countText = fmt.Sprintf("%d of %d", shown, total)
		}
		helpText = fmt.Sprintf("%s [%s] | Enter: Detail | a: Toggle Open Files | p/n/t/m/f: Sort | /: Search | Esc/q: Quit | Up/Down: Scroll%s", countText, mode, suffix)
	case tabNetwork:
		mode := "ALL"
		switch m.networkScope {
		case networkScopeHost:
			mode = "HOST"
		case networkScopeDocker:
			mode = "DOCKER"
		}
		total := len(m.networkTable.Rows())
		src := ""
		if m.networkHostSrc != "" {
			src = " | via " + m.networkHostSrc
		}
		errHint := ""
		if m.networkErr != "" && (m.networkScope == networkScopeDocker || m.networkScope == networkScopeAll) {
			errHint = " | " + m.networkErr
		}
		helpText = fmt.Sprintf("Total: %d [%s]%s | Enter: Detail | a: Filter ALL/HOST/DOCKER | s/n/k: Sort | /: Search | Tab: Focus | Esc/q: Quit%s",
			total, mode, src, errHint)
	}
	footerContent := helpText
	if m.version != "" {
		gap := m.width - 6 - lipgloss.Width(helpText) - lipgloss.Width(m.version)
		if gap > 0 {
			footerContent = helpText + strings.Repeat(" ", gap) + m.version
		}
	}

	processesTab := inactiveTabStyle.Render("1. Processes")
	portsTab := inactiveTabStyle.Render("2. Ports")
	containersTab := inactiveTabStyle.Render("3. Containers")
	locksTab := inactiveTabStyle.Render("4. Locks")
	networkTab := inactiveTabStyle.Render("5. Network")
	switch m.activeTab {
	case tabProcesses:
		processesTab = activeTabStyle.Render("1. Processes")
	case tabPorts:
		portsTab = activeTabStyle.Render("2. Ports")
	case tabContainers:
		containersTab = activeTabStyle.Render("3. Containers")
	case tabLocks:
		locksTab = activeTabStyle.Render("4. Locks")
	case tabNetwork:
		networkTab = activeTabStyle.Render("5. Network")
	}

	headerSegs := []string{
		titleStyle.Render("witr"),
		processesTab,
		portsTab,
		containersTab,
	}
	if locksTabEnabled {
		headerSegs = append(headerSegs, locksTab)
	}
	headerSegs = append(headerSegs, networkTab)
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerSegs...)

	return outerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			header,
			spacerStyle.Render(""),
			statusBarStyle.Render(status),
			statusBarStyle.Render(inputView),
			mainContent,
			spacerStyle.Render(""),
			footerStyle.Width(m.width-4).Render(footerContent),
		),
	)
}

func (m MainModel) viewDetailLoading(outerStyle lipgloss.Style) string {
	helpText := "Esc/q: Back"
	footerContent := helpText
	if m.version != "" {
		gap := m.width - 6 - lipgloss.Width(helpText) - lipgloss.Width(m.version)
		if gap > 0 {
			footerContent = helpText + strings.Repeat(" ", gap) + m.version
		}
	}

	return outerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render("witr")),
			spacerStyle.Render(""),
			lipgloss.NewStyle().Width(m.width-4).Height(m.height-7).Render("Loading details..."),
			spacerStyle.Render(""),
			footerStyle.Width(m.width-4).Render(footerContent),
		),
	)
}

func (m MainModel) viewContainerDetail(outerStyle lipgloss.Style) string {
	activeBorderColor := colorAccent
	detailHeader := tableHeaderStyle.
		BorderForeground(activeBorderColor).
		Foreground(activeBorderColor)
	title := "Container Detail"
	if !m.viewport.AtTop() && !m.viewport.AtBottom() {
		title += " ↕"
	} else if !m.viewport.AtTop() {
		title += " ↑"
	} else if !m.viewport.AtBottom() {
		title += " ↓"
	}

	headerComponents := []string{titleStyle.Render("witr")}
	if id := output.ShortContainerID(m.selectedContainer.ID); id != "" {
		headerComponents = append(headerComponents, pidStyle.Render("ID "+id))
	}

	helpText := "Esc/q: Back | Up/Down: Scroll"
	footerContent := helpText
	if m.version != "" {
		gap := m.width - 6 - lipgloss.Width(helpText) - lipgloss.Width(m.version)
		if gap > 0 {
			footerContent = helpText + strings.Repeat(" ", gap) + m.version
		}
	}

	// Match the structure used for process detail: a single Width-
	// and Height-pinned pane that contains the title row plus the
	// scrollable viewport. The pane absorbs any content/scroll
	// variation so the surrounding layout stays put.
	paneWidth := m.width - 4
	if paneWidth < 1 {
		paneWidth = 1
	}
	contentPane := lipgloss.NewStyle().
		Width(paneWidth).
		Height(m.viewport.Height + 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			detailHeader.Width(m.viewport.Width).Render(title),
			paddedStyle.Render(m.viewport.View()),
		))

	return outerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Center, headerComponents...),
			spacerStyle.Render(""),
			contentPane,
			spacerStyle.Render(""),
			footerStyle.Width(m.width-4).Render(footerContent),
		),
	)
}

func (m MainModel) viewNetworkDetail(outerStyle lipgloss.Style) string {
	activeBorderColor := colorAccent
	detailHeader := tableHeaderStyle.
		BorderForeground(activeBorderColor).
		Foreground(activeBorderColor)
	title := "Network Detail"
	if !m.viewport.AtTop() && !m.viewport.AtBottom() {
		title += " ↕"
	} else if !m.viewport.AtTop() {
		title += " ↑"
	} else if !m.viewport.AtBottom() {
		title += " ↓"
	}

	headerComponents := []string{titleStyle.Render("witr")}
	if m.selectedNetwork != nil && m.selectedNetwork.Name != "" {
		headerComponents = append(headerComponents, pidStyle.Render(m.selectedNetwork.Name))
	} else if e, ok := m.selectedNetworkEntry(); ok && e.IsHost {
		headerComponents = append(headerComponents, pidStyle.Render(e.Host.Name))
	}

	helpText := "Esc/q: Back | Up/Down: Scroll"
	footerContent := helpText
	if m.version != "" {
		gap := m.width - 6 - lipgloss.Width(helpText) - lipgloss.Width(m.version)
		if gap > 0 {
			footerContent = helpText + strings.Repeat(" ", gap) + m.version
		}
	}

	paneWidth := m.width - 4
	if paneWidth < 1 {
		paneWidth = 1
	}
	contentPane := lipgloss.NewStyle().
		Width(paneWidth).
		Height(m.viewport.Height + 2).
		Render(lipgloss.JoinVertical(lipgloss.Left,
			detailHeader.Width(m.viewport.Width).Render(title),
			paddedStyle.Render(m.viewport.View()),
		))

	return outerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Center, headerComponents...),
			spacerStyle.Render(""),
			contentPane,
			spacerStyle.Render(""),
			footerStyle.Width(m.width-4).Render(footerContent),
		),
	)
}

func (m MainModel) viewProcessDetail(outerStyle lipgloss.Style) string {
	availableWidth := m.width - 6
	if availableWidth < 0 {
		availableWidth = 0
	}
	detailWidth := int(float64(availableWidth) * detailPaneRatio)
	envWidth := availableWidth - detailWidth

	envContainerStyle := envPanelStyle.
		Width(envWidth).
		Height(m.viewport.Height + 2)

	detailHeader := tableHeaderStyle
	envHeader := tableHeaderStyle

	activeBorderColor := colorAccent
	dimColor := colorHeaderDim
	dimBorderColor := colorBorderDim

	if m.detailFocus == focusDetail {
		detailHeader = detailHeader.BorderForeground(activeBorderColor).Foreground(activeBorderColor)
		envHeader = envHeader.BorderForeground(dimBorderColor).Foreground(dimColor)
		envContainerStyle = envContainerStyle.BorderForeground(dimBorderColor)
	} else {
		detailHeader = detailHeader.BorderForeground(dimBorderColor).Foreground(dimColor)
		envHeader = envHeader.BorderForeground(activeBorderColor).Foreground(activeBorderColor)
		envContainerStyle = envContainerStyle.BorderForeground(activeBorderColor)
	}

	detailTitle := "Process Detail"
	if !m.viewport.AtTop() && !m.viewport.AtBottom() {
		detailTitle += " ↕"
	} else if !m.viewport.AtTop() {
		detailTitle += " ↑"
	} else if !m.viewport.AtBottom() {
		detailTitle += " ↓"
	}

	envTitle := "Environment Variables"
	if !m.envViewport.AtTop() && !m.envViewport.AtBottom() {
		envTitle += " ↕"
	} else if !m.envViewport.AtTop() {
		envTitle += " ↑"
	} else if !m.envViewport.AtBottom() {
		envTitle += " ↓"
	}

	splitContent := lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(detailWidth).Render(
			lipgloss.JoinVertical(lipgloss.Left,
				detailHeader.Width(m.viewport.Width).Render(detailTitle),
				paddedStyle.Render(m.viewport.View()),
			),
		),
		envContainerStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				envHeader.Width(m.envViewport.Width).Render(envTitle),
				paddedStyle.Render(m.envViewport.View()),
			),
		),
	)

	headerComponents := []string{
		titleStyle.Render("witr"),
	}
	if m.selectedDetail != nil {
		headerComponents = append(headerComponents, pidStyle.Render(fmt.Sprintf("PID %d", m.selectedDetail.Process.PID)))
	}

	var helpText string
	pid := 0
	if m.selectedDetail != nil {
		pid = m.selectedDetail.Process.PID
	}
	switch {
	case m.actionMenuOpen:
		helpText = actionMenuStyle.Render("Esc/q: cancel | Actions:  [k]ill  [t]erm  [p]ause  [r]esume  [n]ice")
	case m.pendingAction == actionKill:
		helpText = confirmStyle.Render(fmt.Sprintf("Kill PID %d? [y]es / [n]o", pid))
	case m.pendingAction == actionTerm:
		helpText = confirmStyle.Render(fmt.Sprintf("Terminate PID %d? [y]es / [n]o", pid))
	case m.pendingAction == actionPause:
		helpText = confirmStyle.Render(fmt.Sprintf("Pause PID %d? [y]es / [n]o", pid))
	case m.pendingAction == actionResume:
		helpText = confirmStyle.Render(fmt.Sprintf("Resume PID %d? [y]es / [n]o", pid))
	case m.pendingAction == actionRenice:
		helpText = confirmStyle.Render(fmt.Sprintf("Nice value for PID %d (−20…19): ", pid)) + m.reniceInput.View()
	case m.statusMsg != "":
		helpText = errorStyle.Render(m.statusMsg)
	default:
		if actionsSupported {
			helpText = "a: Actions | Esc/q: Back | Tab: Focus | Up/Down: Scroll"
		} else {
			helpText = "Esc/q: Back | Tab: Focus | Up/Down: Scroll"
		}
	}
	footerContent := helpText
	if m.version != "" && !m.actionMenuOpen && m.pendingAction == actionNone && m.statusMsg == "" {
		gap := m.width - 6 - lipgloss.Width(helpText) - lipgloss.Width(m.version)
		if gap > 0 {
			footerContent = helpText + strings.Repeat(" ", gap) + m.version
		}
	}

	return outerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			lipgloss.JoinHorizontal(lipgloss.Center, headerComponents...),
			spacerStyle.Render(""),
			splitContent,
			spacerStyle.Render(""),
			footerStyle.Width(m.width-4).Render(footerContent),
		),
	)
}
