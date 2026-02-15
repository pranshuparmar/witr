package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	"github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	// Detail view styles
	detailHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("229")).
				Bold(true).
				MarginBottom(1)

	labelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Width(15)

	flagStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	// New styles for visual refinement
	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("62")). // Softer Blue/Purple
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, true, false). // Bottom border only
				BorderForeground(lipgloss.Color("240")).
				Padding(0, 1) // Keep padding for text spacing

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")). // Match header color
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).                          // Dimmed gray
			Border(lipgloss.NormalBorder(), true, false, false, false). // Top border only
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1).
			Width(100) // Will be updated dynamically in View/Update if needed, or rely on container
)

type tickMsg time.Time

type modelState int

const (
	stateList modelState = iota
	stateDetail
)

type focusState int

const (
	focusDetail focusState = iota
	focusEnv
)

type MainModel struct {
	state          modelState
	table          table.Model
	input          textinput.Model
	viewport       viewport.Model // For full detail view
	treeViewport   viewport.Model // For split view (30%)
	envViewport    viewport.Model // For detail split view (30% Env)
	processes      []model.Process
	filtered       []model.Process
	selectedDetail *model.Result
	detailFocus    focusState // Track which pane is focused in detail view
	err            error
	width          int
	height         int
	quitting       bool

	// Flags
	flagExact    bool
	flagTree     bool
	flagWarnings bool
	flagVerbose  bool // Always true for detail fetch currently
}

func InitialModel() MainModel {
	// Initialize table
	columns := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "Process", Width: 15},
		{Title: "User", Width: 10},
		{Title: "Started", Width: 20},
		{Title: "Command", Width: 40},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = tableHeaderStyle // Use new header style
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	t.SetStyles(s)

	// Initialize input
	ti := textinput.New()
	ti.Placeholder = "Type / to search..."
	ti.CharLimit = 156
	ti.Width = 30
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle // Green/Cyan prompt
	ti.Blur()                    // Start blurred to allow navigation keys

	vp := viewport.New(0, 0)
	vp.YPosition = 0

	tvp := viewport.New(0, 0) // Tree viewport
	tvp.YPosition = 0

	evp := viewport.New(0, 0) // Env viewport
	evp.YPosition = 0

	return MainModel{
		state:        stateList,
		table:        t,
		input:        ti,
		viewport:     vp,
		treeViewport: tvp,
		envViewport:  evp,
		detailFocus:  focusDetail, // Default focus
		flagVerbose:  true,
	}
}

func Start() error {
	p := tea.NewProgram(InitialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running tui: %w", err)
	}
	return nil
}

func (m MainModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.refreshProcesses(),
	)
}

func (m MainModel) refreshProcesses() tea.Cmd {
	return func() tea.Msg {
		procs, err := proc.ListProcesses()
		if err != nil {
			return err
		}
		// Sort by Started (Desc) by default
		sort.Slice(procs, func(i, j int) bool {
			return procs[i].StartedAt.After(procs[j].StartedAt)
		})
		return procs
	}
}

type treeMsg model.Result // Wrapper type for tree fetch result

func (m MainModel) fetchTree(pid int) tea.Cmd {
	return func() tea.Msg {
		// Fetch simple tree (no verbose extended info for speed)?
		// pipeline.AnalyzePID resolves ancestry.
		// We can just use the same analysis but maybe control verbosity?
		// For the tree, we really just need ancestry.
		// AnalyzePID fetches extended info if verbose is true.
		// We'll set verbose=false for speed in the split view update.
		// But User wants "witr's tree flag output".
		res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
			PID:     pid,
			Verbose: false, // Fast fetch for split view
			Tree:    true,
		})
		if err != nil {
			return treeMsg{} // Return empty result to clear view on error
		}
		return treeMsg(res)
	}
}

func (m MainModel) fetchProcessDetail(pid int) tea.Cmd {
	return func() tea.Msg {
		res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
			PID:     pid,
			Verbose: true,
			Tree:    true,
		})
		if err != nil {
			return err
		}
		return res
	}
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global keys
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

		if m.state == stateList {
			// Input handling
			if m.input.Focused() {
				if msg.String() == "enter" || msg.String() == "esc" {
					m.input.Blur()
					return m, nil
				}
				// Pass to input
				var inputCmd tea.Cmd
				m.input, inputCmd = m.input.Update(msg)
				m.filterProcesses()

				// Auto-select first result and update tree on search
				m.table.SetCursor(0)
				var treeCmd tea.Cmd
				if len(m.filtered) > 0 {
					selected := m.table.SelectedRow()
					if len(selected) > 0 {
						pid := 0
						fmt.Sscanf(selected[0], "%d", &pid)
						if pid > 0 {
							treeCmd = m.fetchTree(pid)
						}
					}
				} else {
					// Clear tree view if no results found
					m.treeViewport.SetContent("")
				}
				return m, tea.Batch(inputCmd, treeCmd)
			}

			// Navigation Mode Keys (Input Blurred)
			switch msg.String() {
			case "/":
				m.input.Focus()
				return m, textinput.Blink
			case "q", "esc":
				m.quitting = true
				return m, tea.Quit
			case "enter":
				if m.table.Focused() {
					selected := m.table.SelectedRow()
					if len(selected) > 0 {
						pid := 0
						fmt.Sscanf(selected[0], "%d", &pid)
						if pid > 0 {
							m.state = stateDetail
							return m, m.fetchProcessDetail(pid)
						}
					}
				}
			}

			// Table navigation
			prevSelected := -1
			if len(m.filtered) > 0 {
				prevSelected = m.table.Cursor()
			}

			m.table, cmd = m.table.Update(msg)

			// Detect selection change to trigger tree fetch
			if len(m.filtered) > 0 && m.table.Cursor() != prevSelected {
				selected := m.table.SelectedRow()
				if len(selected) > 0 {
					pid := 0
					fmt.Sscanf(selected[0], "%d", &pid)
					if pid > 0 {
						// Return batch: table cmd + tree fetch
						return m, tea.Batch(cmd, m.fetchTree(pid))
					}
				}
			}

			return m, cmd

		} else if m.state == stateDetail {
			// Detail View Keys
			switch msg.String() {
			case "esc", "q", "backspace":
				m.state = stateList
				m.selectedDetail = nil
				m.detailFocus = focusDetail // Reset focus
				return m, nil
			case "left", "h":
				m.detailFocus = focusDetail
				return m, nil
			case "right", "l":
				m.detailFocus = focusEnv
				return m, nil
			// Removed w and t toggles
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate available width for content inside baseStyle
		// baseStyle has Border (2) + Padding (2) = 4
		// Extra safety buffer = 2 to ensure right border is visible
		availableWidth := msg.Width - 6
		if availableWidth < 0 {
			availableWidth = 0
		}

		// Vertical Split:
		// Left: List (70%)
		// Right: Tree (30%)
		listWidth := int(float64(availableWidth) * 0.7)
		treeWidth := availableWidth - listWidth - 2 // minus split border/padding

		// Calculate available height for Main Content
		contentHeight := msg.Height - 11
		if contentHeight < 5 {
			contentHeight = 5 // Minimum height
		}

		// Fixed widths for PID(8)+Process(15)+User(10)+Started(20) = 53
		// Table usually has some internal padding between columns? Bubbletea table uses explicit widths.
		// Let's assume some buffer.
		fixedColumnsWidth := 53
		// Dynamic Command Width
		// Total list width - fixed columns - safe buffer (approx 12 for borders/spacing)
		// Increased buffer to leave gap on right
		cmdWidth := listWidth - fixedColumnsWidth - 12
		if cmdWidth < 10 {
			cmdWidth = 10
		}

		columns := []table.Column{
			{Title: "PID", Width: 8},
			{Title: "Process", Width: 15},
			{Title: "User", Width: 10},
			{Title: "Started", Width: 20},
			{Title: "Command", Width: cmdWidth},
		}
		m.table.SetColumns(columns)

		m.table.SetWidth(listWidth)
		m.table.SetHeight(contentHeight)

		// Tree view has an internal header "Ancestry Tree" (1 line) + PaddingBottom(1)
		// Tree Viewport Height = contentHeight - 2
		treeVpHeight := contentHeight - 2
		if treeVpHeight < 0 {
			treeVpHeight = 0
		}

		m.treeViewport.Width = treeWidth
		m.treeViewport.Height = treeVpHeight

		// Detail Viewport
		// Split 70% Detailed (Left) / 30% Env (Right) of AVAILABLE width
		detailWidth := int(float64(availableWidth) * 0.7)
		envWidth := availableWidth - detailWidth - 2 // minus split border/padding

		vpHeight := msg.Height - 9 // Optimized overhead deduction (Overhead ~9 lines to fit footer)
		if vpHeight < 0 {
			vpHeight = 0
		}

		m.viewport.Width = detailWidth - 2 // Increased right gap (-6)
		m.viewport.Height = vpHeight

		// Env viewport should be smaller than valid envWidth to account for container style
		// Container has Border(1) + Padding(1) = 2 chars overhead
		m.envViewport.Width = envWidth
		if m.envViewport.Width < 0 {
			m.envViewport.Width = 0
		}
		m.envViewport.Height = vpHeight

	case []model.Process:
		m.processes = msg
		m.filterProcesses()
		// Auto-select first and fetch tree?
		if len(m.filtered) > 0 {
			m.table.SetCursor(0)
			return m, m.fetchTree(m.filtered[0].PID)
		}

	case treeMsg:
		m.updateTreeViewport(model.Result(msg))

	case model.Result:
		m.selectedDetail = &msg
		m.updateDetailViewport()
		m.updateEnvViewport()

	case error:
		m.err = msg
		return m, nil
	}

	return m, nil
}

func (m *MainModel) updateEnvViewport() {
	if m.selectedDetail == nil {
		return
	}
	res := *m.selectedDetail
	var b strings.Builder

	// Adapted from internal/output/envonly.go
	if len(res.Process.Env) > 0 {
		for _, env := range res.Process.Env {
			fmt.Fprintf(&b, "%s\n", env)
		}
	} else {
		fmt.Fprintf(&b, "No environment variables found.\n")
	}

	content := b.String()
	// m.envViewport.Width is already adjusted for container padding/borders in Update
	if m.envViewport.Width > 0 {
		// Use strict wrapping to force break on long strings without spaces
		content = wrap.String(content, m.envViewport.Width)
	}
	m.envViewport.SetContent(content)
}

func (m *MainModel) filterProcesses() {
	filter := m.input.Value()
	if !m.flagExact {
		filter = strings.ToLower(filter)
	}

	var rows []table.Row

	m.filtered = nil
	for _, p := range m.processes {
		cmd := p.Command
		if !m.flagExact {
			cmd = strings.ToLower(cmd)
		}

		match := false
		if filter == "" {
			match = true
		} else if m.flagExact {
			match = p.Command == m.input.Value() // Exact case-sensitive match on name
		} else {
			// Search in Command, PID, User, or Cmdline
			match = strings.Contains(cmd, filter) ||
				strings.Contains(fmt.Sprintf("%d", p.PID), filter) ||
				strings.Contains(strings.ToLower(p.User), filter) ||
				strings.Contains(strings.ToLower(p.Cmdline), filter)
		}

		if match {
			m.filtered = append(m.filtered, p)
			startedStr := p.StartedAt.Format("Jan 02 15:04:05")
			if p.StartedAt.IsZero() {
				startedStr = ""
			}

			rows = append(rows, table.Row{
				fmt.Sprintf("%d", p.PID),
				p.Command, // Process Name
				// PPID removed
				p.User,
				startedStr,
				p.Cmdline, // Full Command
			})
		}
	}
	m.table.SetRows(rows)
}

func (m *MainModel) updateDetailViewport() {
	if m.selectedDetail == nil {
		return
	}
	res := *m.selectedDetail
	var b strings.Builder

	if m.flagWarnings {
		output.RenderWarnings(&b, res, true)
	} else if m.flagTree {
		output.PrintTree(&b, res.Ancestry, res.Children, true)
	} else {
		output.RenderStandard(&b, res, true, true)
	}

	content := b.String()
	if m.viewport.Width > 0 {
		// Use strict wrapping here too for consistency
		content = wrap.String(content, m.viewport.Width)
	}
	m.viewport.SetContent(content)
}

func (m *MainModel) updateTreeViewport(res model.Result) {
	if len(res.Ancestry) == 0 && res.Process.PID == 0 {
		m.treeViewport.SetContent("")
		return
	}
	var b strings.Builder
	// User requested "witr's tree flag output"
	// We use the same format
	output.PrintTree(&b, res.Ancestry, res.Children, true)

	content := b.String()
	// Wrap tree content if needed
	if m.treeViewport.Width > 0 {
		content = wrap.String(content, m.treeViewport.Width)
	}
	m.treeViewport.SetContent(content)
}

func (m MainModel) View() string {
	if m.quitting {
		return ""
	}
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	// Enforce consistent outer box size
	// We subtract 2 for the borders (1 top/bottom, 1 left/right is handled by style padding/border definitions if standard)
	// baseStyle has Border(Normal) -> 1 cell each side.
	// So inner content + border = full size.
	// We want the *Rendered* string to be m.width x m.height.
	// lipgloss.Width/Height on style sets the content width/height.
	// If we set style width/height, it includes padding but excludes border? No, lipgloss behavior varies.
	// Safest is to set Width/Height on the style to (m.width - 2) and (m.height - 2) if standard border is used.

	outerStyle := baseStyle.
		Width(m.width-2).
		Height(m.height-2).
		Padding(0, 1)

	if m.state == stateList {
		status := "Mode: Navigation (Press / to search)"
		if m.input.Focused() {
			status = "Mode: Searching (Press Esc/Enter to stop)"
		}

		// Main layout: Header + (Table | Tree) + Footer
		// Enforce strict height on the tree container to match table
		treeContainerStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(2).          // Increased padding
			Height(m.table.Height()) // Match calculated contentHeight

		treeHeader := "Ancestry Tree"
		selected := m.table.SelectedRow()
		if len(selected) > 0 {
			treeHeader = fmt.Sprintf("Ancestry Tree for PID %s", selected[0])
		}

		mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
			m.table.View(),
			treeContainerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					// Explicitly width-constrained header to ensure bottom border spans full pane
					tableHeaderStyle.Copy().Width(m.treeViewport.Width).Render(treeHeader),
					lipgloss.NewStyle().PaddingLeft(1).Render(m.treeViewport.View()),
				),
			),
		)

		return outerStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.MarginBottom(1).Render("witr dashboard"),
				lipgloss.NewStyle().MarginBottom(1).PaddingLeft(1).Render(fmt.Sprintf("%s", status)),
				lipgloss.NewStyle().MarginBottom(1).PaddingLeft(1).Render(m.input.View()),
				mainContent,
				// Footer at the bottom
				lipgloss.NewStyle().Height(1).Render(""), // Spacer if needed, or just rely on JoinVertical
				footerStyle.Width(m.width-4).Render(fmt.Sprintf("Total: %d | Esc/q: quit | enter: detail | Up/Down: Scroll", len(m.filtered))),
			),
		)
	}

	if m.state == stateDetail {
		if m.selectedDetail == nil {
			// consistent loading UI
			return outerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					lipgloss.JoinHorizontal(lipgloss.Center, titleStyle.Render("witr dashboard")),
					lipgloss.NewStyle().Height(1).Render(""),
					lipgloss.NewStyle().Width(m.width-4).Height(m.height-7).Render("Loading details..."),
					lipgloss.NewStyle().Height(1).Render(""),
					footerStyle.Width(m.width-4).Render("Esc/q: Back"),
				),
			)
		}

		availableWidth := m.width - 6
		if availableWidth < 0 {
			availableWidth = 0
		}
		detailWidth := int(float64(availableWidth) * 0.7)
		envWidth := availableWidth - detailWidth

		// Right Pane Style (Env) - Container
		// Match treeContainerStyle: Left border only
		envContainerStyle := lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(1).               // Reduced padding (was 2)
			Width(envWidth).              // Enforce width
			Height(m.viewport.Height + 2) // Account for Header (2 lines) + Viewport

		// Header Styles with Focus Logic
		detailHeader := tableHeaderStyle.Copy()
		envHeader := tableHeaderStyle.Copy()

		// Dim inactive headers
		if m.detailFocus == focusDetail {
			envHeader.Foreground(lipgloss.Color("243")).BorderForeground(lipgloss.Color("240"))
		} else {
			detailHeader.Foreground(lipgloss.Color("243")).BorderForeground(lipgloss.Color("240"))
		}

		// Render Left Pane (Details)
		// We need to apply the border style to the Viewport content
		// But viewport.View() just returns string.
		// We wrap viewport in the style.
		// Wait, baseStyle was applied to the Whole container.
		// The split layout needs separate containers if we want separate focus borders?
		// Currently: baseStyle wraps the whole JoinVertical(header, splitContent).

		// Let's modify to:
		// Header
		// SplitContent:
		//   Left: Info
		//   Right: Env
		// We want to highlight the PANE.

		// Adjusting layout:
		// Left Pane: viewport.View()
		// Right Pane: envContainer.Render(...)

		// To show focus, maybe change the Header text color or add a border around the specific pane?
		// Simple approach: Add a border around the focused pane if possible, or change the title color.

		// Let's change the "Environment" header color for right pane
		envHeaderStyle := lipgloss.NewStyle().Bold(true).PaddingBottom(1)
		if m.detailFocus == focusEnv {
			envHeaderStyle = envHeaderStyle.Foreground(lipgloss.Color("63"))
		}

		// Removed mainHeaderStyle and header variable

		// Construct Split Content
		splitContent := lipgloss.JoinHorizontal(lipgloss.Top,
			// Left Pane (Details)
			lipgloss.NewStyle().Width(detailWidth).Render(
				lipgloss.JoinVertical(lipgloss.Left,
					detailHeader.Width(m.viewport.Width).Render("Process Detail"),
					lipgloss.NewStyle().PaddingLeft(1).Render(m.viewport.View()),
				),
			),
			// Right Pane (Env)
			envContainerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					// Header width should match container inner width (envWidth - border(1) - padding(1) = envWidth - 2)
					// Actually envViewport.Width is already calculated as envWidth - 2
					envHeader.Width(m.envViewport.Width).Render("Environment Variables"),
					lipgloss.NewStyle().PaddingLeft(1).Render(m.envViewport.View()),
				),
			),
		)

		// PID Header Style
		pidStyle := lipgloss.NewStyle().
			Background(lipgloss.Color("2")).
			Foreground(lipgloss.Color("15")). // White text
			Padding(0, 1).
			Bold(true)

		headerComponents := []string{
			titleStyle.Render("witr dashboard"),
		}
		if m.selectedDetail != nil {
			headerComponents = append(headerComponents, pidStyle.Render(fmt.Sprintf("PID %d", m.selectedDetail.Process.PID)))
		}

		return outerStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				lipgloss.JoinHorizontal(lipgloss.Center, headerComponents...),
				lipgloss.NewStyle().Height(1).Render(""), // Blank line after header
				splitContent,
				// Footer
				lipgloss.NewStyle().Height(1).Render(""), // Small spacer before footer
				footerStyle.Width(m.width-4).Render("Esc/q: Back | Left/Right: Switch Pane | Up/Down: Scroll"),
			),
		)
	}

	return "Unknown state"
}
