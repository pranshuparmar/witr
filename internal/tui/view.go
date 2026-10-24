package tui

import (
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/pkg/model"
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
		if m.selectedDetail == nil && m.selectedContainer == nil {
			return m.viewDetailLoading(outerStyle)
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

	actionsHint := ""
	if actionsSupported {
		actionsHint = "a: Actions | "
	}
	helpText := fmt.Sprintf("Total: %d | Enter: Detail | %sp/n/u/c/m/t: Sort | Esc/q: Quit | Tab: Focus", len(m.filtered), actionsHint)
	switch m.activeTab {
	case tabPorts:
		filterStatus := "LISTEN"
		if m.showAllPorts {
			filterStatus = "ALL"
		}
		hint := ""
		if ownersHidden() {
			hint = " (use sudo to see all owners)"
		}
		helpText = fmt.Sprintf("Total: %d [%s]%s | p/t/n/s: Sort | a: Toggle All | Esc/q: Quit | Tab: Focus | Up/Down: Scroll", len(m.portTable.Rows()), filterStatus, hint)
	case tabContainers:
		helpText = fmt.Sprintf("Total: %d | Enter: Detail | i/n/r/g/s: Sort | /: Search | Esc/q: Quit | Up/Down: Scroll", len(m.containerTable.Rows()))
	case tabLocks:
		hint := ""
		if os.Geteuid() != 0 {
			hint = " (use sudo for full paths)"
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
		helpText = fmt.Sprintf("%s [%s]%s | Enter: Detail | a: Toggle Open Files | p/n/t/m/f: Sort | /: Search | Esc/q: Quit | Up/Down: Scroll", countText, mode, hint)
	}
	footerContent := helpText
	if m.isReplay && len(m.snapshots) > 0 {
		ts := m.snapshots[m.snapshotIndex].Timestamp.Format("2006-01-02 15:04:05")
		footerContent = fmt.Sprintf("[%s] [/]: Step | %s", ts, helpText)
	}
	if m.version != "" {
		gap := m.width - 6 - lipgloss.Width(footerContent) - lipgloss.Width(m.version)
		if gap > 0 {
			footerContent = footerContent + strings.Repeat(" ", gap) + m.version
		}
	}
	aboveFooter := spacerStyle.Render("")
	if m.actionActive() && m.actionTarget != nil {
		footerContent = m.actionFooter()
		aboveFooter = actionTargetStyle.Width(m.width - 4).Render(actionTargetLine(*m.actionTarget, m.width-6))
	}

	processesTab := inactiveTabStyle.Render("1. Processes")
	portsTab := inactiveTabStyle.Render("2. Ports")
	containersTab := inactiveTabStyle.Render("3. Containers")
	locksTab := inactiveTabStyle.Render("4. Locks")
	switch m.activeTab {
	case tabProcesses:
		processesTab = activeTabStyle.Render("1. Processes")
	case tabPorts:
		portsTab = activeTabStyle.Render("2. Ports")
	case tabContainers:
		containersTab = activeTabStyle.Render("3. Containers")
	case tabLocks:
		locksTab = activeTabStyle.Render("4. Locks")
	}

	titleText := "witr"
	if m.isReplay && len(m.snapshots) > 0 {
		titleText = fmt.Sprintf("witr [Replay %d/%d]", m.snapshotIndex+1, len(m.snapshots))
	}
	headerSegs := []string{
		titleStyle.Render(titleText),
		processesTab,
		portsTab,
		containersTab,
	}
	if locksTabEnabled {
		headerSegs = append(headerSegs, locksTab)
	}
	header := lipgloss.JoinHorizontal(lipgloss.Top, headerSegs...)

	return outerStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			header,
			spacerStyle.Render(""),
			statusBarStyle.Render(status),
			statusBarStyle.Render(inputView),
			mainContent,
			aboveFooter,
			footerStyle.Width(m.width-4).Render(footerContent),
		),
	)
}

// actionTargetLabel names a process in action prompts, e.g. "nginx (PID 42)".
func actionTargetLabel(p model.Process) string {
	return fmt.Sprintf("%s (PID %d)", output.SanitizeTerminalLine(p.Command), p.PID)
}

// actionFooter renders the open action menu, confirmation prompt or renice
// input for the current action target.
func (m MainModel) actionFooter() string {
	if m.actionTarget == nil {
		return ""
	}
	label := actionTargetLabel(*m.actionTarget)
	switch m.pendingAction {
	case actionKill:
		return confirmStyle.Render(fmt.Sprintf("Kill %s? [y]es / [n]o", label))
	case actionTerm:
		return confirmStyle.Render(fmt.Sprintf("Terminate %s? [y]es / [n]o", label))
	case actionPause:
		return confirmStyle.Render(fmt.Sprintf("Pause %s? [y]es / [n]o", label))
	case actionResume:
		return confirmStyle.Render(fmt.Sprintf("Resume %s? [y]es / [n]o", label))
	case actionRenice:
		return confirmStyle.Render(fmt.Sprintf("Nice value for %s (−20…19): ", label)) + m.reniceInput.View()
	}
	return actionMenuStyle.Render(fmt.Sprintf("Esc/q: cancel | %s → [k]ill  [t]erm  [p]ause  [r]esume  [n]ice", label))
}

// actionTargetLine identifies the action target in the list view, where the
// highlighted row alone isn't proof of which process was picked: the list
// re-sorts as it refreshes.
func actionTargetLine(p model.Process, width int) string {
	started, _ := output.FormatStartedAt(p.StartedAt)
	line := fmt.Sprintf("Target: %s · user %s · started %s",
		actionTargetLabel(p), output.SanitizeTerminalLine(p.User), started)
	if cmd := output.SanitizeTerminalLine(p.Cmdline); cmd != "" {
		if room := width - lipgloss.Width(line) - 3; room >= 10 {
			line += " · " + truncateMiddle(cmd, room)
		}
	}
	return truncateMiddle(line, width)
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
	switch {
	case m.actionActive():
		helpText = m.actionFooter()
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
	if m.version != "" && !m.actionActive() && m.statusMsg == "" {
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

// ownersHidden reports whether other users' processes are hidden from witr:
// Windows reports every socket's owner, elsewhere only root sees them all. A
// variable for tests.
var ownersHidden = func() bool {
	return runtime.GOOS != "windows" && os.Geteuid() != 0
}
