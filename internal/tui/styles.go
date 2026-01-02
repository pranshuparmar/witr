//go:build linux || darwin

package tui

import "github.com/charmbracelet/lipgloss"

// Colors
var (
	colorPrimary   = lipgloss.Color("#7C3AED") // Purple
	colorSecondary = lipgloss.Color("#3B82F6") // Blue
	colorSuccess   = lipgloss.Color("#22C55E") // Green
	colorWarning   = lipgloss.Color("#F59E0B") // Amber
	colorDanger    = lipgloss.Color("#EF4444") // Red
	colorMuted     = lipgloss.Color("#6B7280") // Gray
	colorBorder    = lipgloss.Color("#374151") // Dark gray
	colorSelected  = lipgloss.Color("#4F46E5") // Indigo
)

// Styles
var (
	// Table styles
	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorSecondary).
				Padding(0, 1)

	tableRowStyle = lipgloss.NewStyle().
			Padding(0, 1)

	tableSelectedStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(lipgloss.Color("#FFFFFF")).
				Padding(0, 1)

	tableMultiSelectStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	// Panel styles
	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	detailsPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorBorder).
				Padding(0, 1)

	detailsTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				MarginBottom(1)

	detailsLabelStyle = lipgloss.NewStyle().
				Foreground(colorMuted)

	detailsValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	// Status bar styles
	helpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Padding(0, 1)

	statusKeyStyle = lipgloss.NewStyle().
			Foreground(colorSecondary).
			Bold(true)

	statusDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	// CPU/Memory color styles
	cpuHighStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	cpuMedStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	cpuNormalStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	memHighStyle = lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true)

	memMedStyle = lipgloss.NewStyle().
			Foreground(colorWarning)

	// Pause indicator
	pausedStyle = lipgloss.NewStyle().
			Foreground(colorWarning).
			Bold(true)

	refreshingStyle = lipgloss.NewStyle().
			Foreground(colorSuccess)

	// Filter input
	filterPromptStyle = lipgloss.NewStyle().
				Foreground(colorPrimary).
				Bold(true)

	filterInputStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))
)
