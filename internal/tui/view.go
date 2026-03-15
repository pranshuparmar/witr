package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m MainModel) View() string {
	if m.quitting {
		return ""
	}

	outerStyle := baseStyle.
		Width(m.width-2).
		Height(m.height-2).
		Padding(0, 1)

	if m.state == stateList {
		status := "Mode: Navigation (Press / to search)"
		if m.statusMsg != "" {
			status = errorStyle.Render(m.statusMsg)
		}
		inputView := m.input.View()

		if m.activeTab == tabPorts {
			if m.portInput.Focused() {
				status = "Mode: Searching (↑↓ to navigate, Esc/Enter to stop)"
			}
			inputView = m.portInput.View()
		} else {
			if m.input.Focused() {
				status = "Mode: Searching (↑↓ to navigate, Esc/Enter to stop)"
			}
		}

		activeBorderColor := lipgloss.Color("#5f5fd7") // Purple/Blue
		dimBorderColor := lipgloss.Color("#585858")    // Dark Gray

		treeBorderColor := dimBorderColor
		treeHeaderColor := dimBorderColor

		if m.listFocus == focusSide {
			treeBorderColor = activeBorderColor
			treeHeaderColor = activeBorderColor
		} else {
			treeHeaderColor = lipgloss.Color("#bcbcbc") // Light Gray
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
		processListPaneWidth := int(float64(availableWidth) * 0.7)
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

		if m.activeTab == tabPorts {
			sideBorderColor := dimBorderColor
			sideHeaderColor := lipgloss.Color("#bcbcbc") // Light Gray

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
			portPaneWidth := int(float64(availableWidth) * 0.5)
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
		if m.activeTab == tabPorts {
			filterStatus := "LISTEN"
			if m.showAllPorts {
				filterStatus = "ALL"
			}
			helpText = fmt.Sprintf("Total: %d [%s] | p/t/n/s: Sort | a: Toggle All | Esc/q: Quit | Tab: Focus | Up/Down: Scroll", len(m.portTable.Rows()), filterStatus)
		}
		footerContent := helpText
		if m.version != "" {
			gap := m.width - 6 - lipgloss.Width(helpText) - lipgloss.Width(m.version)
			if gap > 0 {
				footerContent = helpText + strings.Repeat(" ", gap) + m.version
			}
		}

		var processesTab, portsTab string
		if m.activeTab == tabProcesses {
			processesTab = activeTabStyle.Render("1. Processes")
			portsTab = inactiveTabStyle.Render("2. Ports")
		} else {
			processesTab = inactiveTabStyle.Render("1. Processes")
			portsTab = activeTabStyle.Render("2. Ports")
		}

		header := lipgloss.JoinHorizontal(lipgloss.Top,
			titleStyle.Render("witr"),
			processesTab,
			portsTab,
		)

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

	if m.state == stateDetail {
		if m.selectedDetail == nil {
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

		availableWidth := m.width - 6
		if availableWidth < 0 {
			availableWidth = 0
		}
		detailWidth := int(float64(availableWidth) * 0.7)
		envWidth := availableWidth - detailWidth

		envContainerStyle := envPanelStyle.
			Width(envWidth).
			Height(m.viewport.Height + 2)

		detailHeader := tableHeaderStyle
		envHeader := tableHeaderStyle

		activeBorderColor := lipgloss.Color("#5f5fd7") // Purple
		dimColor := lipgloss.Color("#bcbcbc")          // Lighter Gray
		dimBorderColor := lipgloss.Color("#585858")    // Dark Gray

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
			helpText = "a: Actions | Esc/q: Back | Tab: Focus | Up/Down: Scroll"
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

	return "Unknown state"
}
