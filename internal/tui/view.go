package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
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
				status = "Mode: Searching (Press Esc/Enter to stop)"
			}
			inputView = m.portInput.View()
		} else {
			if m.input.Focused() {
				status = "Mode: Searching (Press Esc/Enter to stop)"
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

		treeContainerStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			BorderForeground(treeBorderColor).
			PaddingLeft(2).
			Height(m.table.Height())

		treeHeader := "Details"
		selected := m.table.SelectedRow()
		if len(selected) > 0 {
			treeHeader = fmt.Sprintf("PID %s", selected[0])
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

		s := table.DefaultStyles()
		if m.listFocus == focusMain {
			s.Header = tableHeaderStyle.BorderForeground(activeBorderColor)
		} else {
			s.Header = tableHeaderStyle.BorderForeground(dimBorderColor)
		}
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("#ffffaf")). // Light Yellow
			Background(lipgloss.Color("#5f00d7")). // Purple
			Bold(false)
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
					lipgloss.NewStyle().PaddingLeft(1).Render(m.treeViewport.View()),
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

			s1 := table.DefaultStyles()
			if m.listFocus == focusMain {
				s1.Header = tableHeaderStyle.BorderForeground(activeBorderColor)
			} else {
				s1.Header = tableHeaderStyle.BorderForeground(dimBorderColor)
			}
			s1.Selected = s1.Selected.
				Foreground(lipgloss.Color("#ffffaf")). // Light Yellow
				Background(lipgloss.Color("#5f00d7")). // Purple
				Bold(false)
			m.portTable.SetStyles(s1)

			s2 := table.DefaultStyles()
			if m.listFocus == focusSide {
				s2.Header = tableHeaderStyle.BorderForeground(activeBorderColor)
			} else {
				s2.Header = tableHeaderStyle.BorderForeground(dimBorderColor)
			}
			s2.Selected = s2.Selected.
				Foreground(lipgloss.Color("#ffffaf")). // Light Yellow
				Background(lipgloss.Color("#5f00d7")). // Purple
				Bold(false)
			m.portDetailTable.SetStyles(s2)

			detailContainerStyle := lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(sideBorderColor).
				PaddingLeft(1).
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
				lipgloss.NewStyle().Height(1).Render(""),
				lipgloss.NewStyle().MarginBottom(1).PaddingLeft(1).Render(fmt.Sprintf("%s", status)),
				lipgloss.NewStyle().MarginBottom(1).PaddingLeft(1).Render(inputView),
				mainContent,
				lipgloss.NewStyle().Height(1).Render(""),
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
					lipgloss.NewStyle().Height(1).Render(""),
					lipgloss.NewStyle().Width(m.width-4).Height(m.height-7).Render("Loading details..."),
					lipgloss.NewStyle().Height(1).Render(""),
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

		envContainerStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(1).
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
					lipgloss.NewStyle().PaddingLeft(1).Render(m.viewport.View()),
				),
			),
			envContainerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					envHeader.Width(m.envViewport.Width).Render(envTitle),
					lipgloss.NewStyle().PaddingLeft(1).Render(m.envViewport.View()),
				),
			),
		)

		pidStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("#22aa22")). // Green
			Foreground(lipgloss.Color("#ffffff")). // White
			Padding(0, 1).
			Bold(true)

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
				lipgloss.NewStyle().Height(1).Render(""),
				splitContent,
				lipgloss.NewStyle().Height(1).Render(""),
				footerStyle.Width(m.width-4).Render(footerContent),
			),
		)
	}

	return "Unknown state"
}
