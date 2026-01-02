//go:build linux || darwin

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/SanCognition/witr/internal/batch"
)

// View renders the UI
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	// Calculate panel widths (70% table, 30% details)
	tableWidth := int(float64(m.width) * 0.68)
	detailsWidth := m.width - tableWidth - 3 // Account for borders

	// Build the panels
	tablePanel := m.renderTablePanel(tableWidth)
	detailsPanel := m.renderDetailsPanel(detailsWidth)

	// Join panels horizontally
	mainContent := lipgloss.JoinHorizontal(lipgloss.Top, tablePanel, detailsPanel)

	// Add status/help bar at the bottom
	helpBar := m.renderHelpBar()

	return lipgloss.JoinVertical(lipgloss.Left, mainContent, helpBar)
}

// renderTablePanel renders the process table
func (m Model) renderTablePanel(width int) string {
	// Calculate available height (minus header, separator, and help bar)
	availableHeight := m.height - 5

	var sb strings.Builder

	// Header
	header := m.renderTableHeader(width)
	sb.WriteString(header)
	sb.WriteString("\n")

	// Separator
	sb.WriteString(m.renderSeparator(width))
	sb.WriteString("\n")

	// Rows
	if len(m.filtered) == 0 {
		empty := lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(width).
			Align(lipgloss.Center).
			Render("No processes found")
		sb.WriteString(empty)
	} else {
		rowCount := min(len(m.filtered), availableHeight-2)
		for i := 0; i < rowCount; i++ {
			row := m.renderTableRow(i, width)
			sb.WriteString(row)
			if i < rowCount-1 {
				sb.WriteString("\n")
			}
		}
	}

	style := panelStyle.
		Width(width).
		Height(availableHeight)

	return style.Render(sb.String())
}

// renderTableHeader renders the table header
func (m Model) renderTableHeader(width int) string {
	// Column widths
	cols := []struct {
		name  string
		width int
	}{
		{"", 2},        // Selection marker
		{"PID", 7},
		{"CPU", 5},
		{"MEM", 6},
		{"AGE", 8},
		{"SOURCE", 10},
		{"SCRIPT", 15},
		{"WORKDIR", 20},
	}

	var parts []string
	for _, col := range cols {
		parts = append(parts, lipgloss.NewStyle().
			Width(col.width).
			Bold(true).
			Foreground(colorSecondary).
			Render(col.name))
	}

	return strings.Join(parts, " ")
}

// renderSeparator renders a separator line
func (m Model) renderSeparator(width int) string {
	return lipgloss.NewStyle().
		Foreground(colorBorder).
		Render(strings.Repeat("─", width-4))
}

// renderTableRow renders a single table row
func (m Model) renderTableRow(idx int, width int) string {
	p := m.filtered[idx]
	isSelected := m.selected[p.PID]
	isCursor := idx == m.cursorIndex

	// Selection marker
	marker := "  "
	if isSelected {
		marker = tableMultiSelectStyle.Render("• ")
	}
	if isCursor {
		marker = lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render("> ")
	}
	if isCursor && isSelected {
		marker = tableMultiSelectStyle.Render("▸ ")
	}

	// Format values
	pid := fmt.Sprintf("%d", p.PID)
	cpu := formatCPU(p.CPU)
	mem := formatMemory(p.MemoryMB)
	age := p.Age
	source := batch.Truncate(p.Source, 10)
	script := batch.Truncate(p.Script, 15)
	workdir := batch.Truncate(batch.ShortenPath(p.WorkDir), 20)

	// Build row
	cols := []struct {
		value string
		width int
	}{
		{pid, 7},
		{cpu, 5},
		{mem, 6},
		{age, 8},
		{source, 10},
		{script, 15},
		{workdir, 20},
	}

	var parts []string
	parts = append(parts, marker)
	for _, col := range cols {
		style := lipgloss.NewStyle().Width(col.width)
		parts = append(parts, style.Render(col.value))
	}

	row := strings.Join(parts, " ")

	// Apply row styling
	if isCursor {
		row = tableSelectedStyle.Render(row)
	}

	return row
}

// formatCPU formats CPU percentage with color
func formatCPU(cpu float64) string {
	str := fmt.Sprintf("%.0f%%", cpu)
	if cpu > 50 {
		return cpuHighStyle.Render(str)
	}
	if cpu > 20 {
		return cpuMedStyle.Render(str)
	}
	return cpuNormalStyle.Render(str)
}

// formatMemory formats memory with color
func formatMemory(mb int) string {
	var str string
	if mb >= 1024 {
		str = fmt.Sprintf("%.1fG", float64(mb)/1024)
	} else {
		str = fmt.Sprintf("%dM", mb)
	}

	if mb > 1024 {
		return memHighStyle.Render(str)
	}
	if mb > 512 {
		return memMedStyle.Render(str)
	}
	return str
}

// renderHelpBar renders the bottom help/status bar
func (m Model) renderHelpBar() string {
	var parts []string

	// Navigation
	parts = append(parts, fmt.Sprintf("%s move", statusKeyStyle.Render("↑↓")))
	parts = append(parts, fmt.Sprintf("%s select", statusKeyStyle.Render("tab")))
	parts = append(parts, fmt.Sprintf("%s kill", statusKeyStyle.Render("enter")))
	parts = append(parts, fmt.Sprintf("%s pause", statusKeyStyle.Render("space")))
	parts = append(parts, fmt.Sprintf("%s filter", statusKeyStyle.Render("/")))
	parts = append(parts, fmt.Sprintf("%s sort:%s", statusKeyStyle.Render("s"), m.sortField.String()))
	parts = append(parts, fmt.Sprintf("%s quit", statusKeyStyle.Render("q")))

	// Status indicators
	var status string
	if m.filterMode {
		status = filterPromptStyle.Render("/") + filterInputStyle.Render(m.filterText) + "_"
	} else if m.paused {
		status = pausedStyle.Render("⏸ PAUSED")
	} else if m.refreshing {
		status = refreshingStyle.Render("↻ refreshing...")
	} else {
		status = statusDescStyle.Render(fmt.Sprintf("updated %s ago", formatTimeSince(m.lastRefresh)))
	}

	// Selected count
	if count := m.selectedCount(); count > 0 {
		status += statusDescStyle.Render(fmt.Sprintf(" | %d selected", count))
	}

	helpText := strings.Join(parts, "  ")

	// Build the bar
	leftSide := helpStyle.Render(helpText)
	rightSide := helpStyle.Render(status)

	// Calculate spacing
	totalWidth := lipgloss.Width(leftSide) + lipgloss.Width(rightSide)
	spacing := m.width - totalWidth - 4
	if spacing < 0 {
		spacing = 0
	}

	return statusBarStyle.
		Width(m.width).
		Render(leftSide + strings.Repeat(" ", spacing) + rightSide)
}

// formatTimeSince formats duration since a time
func formatTimeSince(t interface{}) string {
	// Handle the time.Time zero value case
	switch v := t.(type) {
	case interface{ IsZero() bool }:
		if v.IsZero() {
			return "-"
		}
	}
	return "just now"
}
