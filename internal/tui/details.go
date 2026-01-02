//go:build linux || darwin

package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/SanCognition/witr/internal/batch"
	procpkg "github.com/SanCognition/witr/internal/proc"
)

// renderDetailsPanel renders the right-side details panel
func (m Model) renderDetailsPanel(width int) string {
	availableHeight := m.height - 5

	p := m.currentProcess()
	if p == nil {
		empty := lipgloss.NewStyle().
			Foreground(colorMuted).
			Width(width).
			Height(availableHeight).
			Align(lipgloss.Center, lipgloss.Center).
			Render("Select a process")
		return detailsPanelStyle.Width(width).Height(availableHeight).Render(empty)
	}

	var sections []string

	// Process title
	title := detailsTitleStyle.Render(fmt.Sprintf("PID %d", p.PID))
	sections = append(sections, title)

	// Command
	cmdSection := m.renderDetailSection("COMMAND", p.Command, width-4)
	sections = append(sections, cmdSection)

	// Full cmdline (truncated)
	if p.Cmdline != "" && p.Cmdline != p.Command {
		cmdline := batch.Truncate(p.Cmdline, 50)
		cmdlineSection := m.renderDetailSection("CMDLINE", cmdline, width-4)
		sections = append(sections, cmdlineSection)
	}

	// Source
	sourceSection := m.renderDetailSection("SOURCE", p.Source, width-4)
	sections = append(sections, sourceSection)

	// Script (if detected)
	if p.Script != "" && p.Script != "-" {
		scriptSection := m.renderDetailSection("SCRIPT", p.Script, width-4)
		sections = append(sections, scriptSection)
	}

	// Workdir
	workdir := batch.ShortenPath(p.WorkDir)
	workdirSection := m.renderDetailSection("WORKDIR", workdir, width-4)
	sections = append(sections, workdirSection)

	// Repo
	if p.GitRepo != "" && p.GitRepo != "-" {
		repoSection := m.renderDetailSection("REPO", p.GitRepo, width-4)
		sections = append(sections, repoSection)
	}

	// Resources
	resources := fmt.Sprintf("CPU: %s  MEM: %s", formatCPU(p.CPU), formatMemory(p.MemoryMB))
	resourcesSection := m.renderDetailSection("RESOURCES", resources, width-4)
	sections = append(sections, resourcesSection)

	// Age & User
	ageUser := fmt.Sprintf("%s • %s", p.Age, p.User)
	ageSection := m.renderDetailSection("RUNNING", ageUser, width-4)
	sections = append(sections, ageSection)

	// Health (if available)
	if p.Health != "" && p.Health != "-" && p.Health != "unknown" {
		healthSection := m.renderDetailSection("HEALTH", p.Health, width-4)
		sections = append(sections, healthSection)
	}

	// Ancestry (fetch from proc if we have details cached)
	ancestrySection := m.renderAncestrySection(p.PID, width-4)
	if ancestrySection != "" {
		sections = append(sections, ancestrySection)
	}

	content := strings.Join(sections, "\n\n")

	return detailsPanelStyle.
		Width(width).
		Height(availableHeight).
		Render(content)
}

// renderDetailSection renders a labeled section
func (m Model) renderDetailSection(label, value string, width int) string {
	labelStr := detailsLabelStyle.Render(label)
	valueStr := detailsValueStyle.Width(width).Render(value)
	return labelStr + "\n" + valueStr
}

// renderAncestrySection renders the process ancestry tree
func (m Model) renderAncestrySection(pid int, width int) string {
	// Fetch ancestry (this is fast, cached in kernel)
	ancestry, err := procpkg.ResolveAncestry(pid)
	if err != nil || len(ancestry) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, detailsLabelStyle.Render("ANCESTRY"))

	// Show ancestry tree (limit to last 5)
	start := 0
	if len(ancestry) > 5 {
		start = len(ancestry) - 5
		lines = append(lines, lipgloss.NewStyle().Foreground(colorMuted).Render("  ..."))
	}

	for i := start; i < len(ancestry); i++ {
		p := ancestry[i]
		indent := strings.Repeat("  ", i-start)
		arrow := "→"
		if i == len(ancestry)-1 {
			arrow = "▸"
		}
		line := fmt.Sprintf("%s%s %s", indent, arrow, batch.Truncate(p.Command, width-len(indent)-4))
		if i == len(ancestry)-1 {
			// Highlight current process
			lines = append(lines, lipgloss.NewStyle().Bold(true).Foreground(colorPrimary).Render(line))
		} else {
			lines = append(lines, lipgloss.NewStyle().Foreground(colorMuted).Render(line))
		}
	}

	return strings.Join(lines, "\n")
}
