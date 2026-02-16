package tui

import (
	"fmt"
	"os"
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
			BorderForeground(lipgloss.Color("240")) // Dark Gray

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")). // White
			Background(lipgloss.Color("#7D56F4")). // Purple
			Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("62")). // Purple/Blue
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("240")). // Dark Gray
				Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("62")). // Purple/Blue
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")). // Dimmed Gray
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("240")). // Dark Gray
			Padding(0, 1).
			Width(100)
)

type modelState int

const (
	stateList modelState = iota
	stateDetail
)

type focusState int

const (
	focusDetail focusState = iota
	focusEnv
	focusMain
	focusSide
)

type MainModel struct {
	state          modelState
	table          table.Model
	input          textinput.Model
	viewport       viewport.Model
	treeViewport   viewport.Model
	envViewport    viewport.Model
	processes      []model.Process
	filtered       []model.Process
	selectedDetail *model.Result
	detailFocus    focusState
	listFocus      focusState
	err            error
	width          int
	height         int
	quitting       bool

	selectionID int

	sortCol  string // "pid", "name", "user", "cpu", "mem", "time"
	sortDesc bool

	version string
}

func InitialModel(version string) MainModel {
	columns := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "User", Width: 12},
		{Title: "CPU%", Width: 6},
		{Title: "Mem", Width: 16},
		{Title: "Started", Width: 19},
		{Title: "Command", Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := table.DefaultStyles()
	s.Header = tableHeaderStyle.BorderForeground(lipgloss.Color("240")) // Dark Gray
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")). // Light Yellow
		Background(lipgloss.Color("56")).  // Purple
		Bold(false)
	t.SetStyles(s)

	ti := textinput.New()
	ti.Placeholder = "Search PID, Name, User, Command..."
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle
	ti.Blur()

	vp := viewport.New(0, 0)
	vp.YPosition = 0

	tvp := viewport.New(0, 0)
	tvp.YPosition = 0

	evp := viewport.New(0, 0)
	evp.YPosition = 0

	return MainModel{
		state:        stateList,
		table:        t,
		input:        ti,
		viewport:     vp,
		treeViewport: tvp,
		envViewport:  evp,
		detailFocus:  focusDetail,
		listFocus:    focusMain,
		sortCol:      "mem",
		sortDesc:     true,
		version:      version,
	}
}

func Start(version string) error {
	p := tea.NewProgram(InitialModel(version), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running tui: %w", err)
	}
	return nil
}

func (m MainModel) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		m.refreshProcesses(),
		waitTick(),
	)
}

func (m MainModel) refreshProcesses() tea.Cmd {
	return func() tea.Msg {
		procs, err := proc.ListProcesses()
		if err != nil {
			return err
		}
		selfPID := os.Getpid()
		filteredProcs := make([]model.Process, 0, len(procs))
		for _, p := range procs {
			if p.PID == selfPID {
				continue
			}
			if p.PPID == selfPID && (p.Command == "ps" || strings.HasPrefix(p.Command, "ps ")) {
				continue
			}
			filteredProcs = append(filteredProcs, p)
		}
		procs = filteredProcs

		return procs
	}
}

type treeMsg model.Result

type debounceMsg struct {
	id  int
	pid int
}

type tickMsg time.Time

func waitTick() tea.Cmd {
	return tea.Tick(10*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m MainModel) fetchTree(p model.Process) tea.Cmd {
	return func() tea.Msg {
		res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
			PID:     p.PID,
			Verbose: false,
			Tree:    true,
		})
		if err != nil {
			return treeMsg(model.Result{
				Process: p,
			})
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
	case tickMsg:
		if m.state == stateList && !m.quitting && !m.input.Focused() {
			cmd = m.refreshProcesses()
		}
		return m, tea.Batch(cmd, waitTick())

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}

		if m.state == stateList {
			if m.input.Focused() {
				if msg.String() == "enter" || msg.String() == "esc" {
					m.input.Blur()
					return m, nil
				}
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
						treeCmd = tea.Tick(500*time.Millisecond, func(_ time.Time) tea.Msg {
							return debounceMsg{id: id, pid: pid}
						})
					}
				} else {
					m.treeViewport.SetContent("")
				}
				return m, tea.Batch(inputCmd, treeCmd)
			}

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
							m.viewport.GotoTop()
							m.envViewport.GotoTop()
							return m, m.fetchProcessDetail(pid)
						}
					}
				}

			// Focus Switching
			case "tab", "right", "left", "l", "h":
				if m.input.Focused() {
					break
				}
				if msg.String() == "tab" || msg.String() == "right" || msg.String() == "l" {
					if m.listFocus == focusMain {
						m.listFocus = focusSide
					} else {
						m.listFocus = focusMain
					}
				} else if msg.String() == "shift+tab" || msg.String() == "left" || msg.String() == "h" {
					if m.listFocus == focusSide {
						m.listFocus = focusMain
					} else {
						m.listFocus = focusSide
					}
				}
				return m, nil

			// Sorting Keys
			case "c", "p", "n", "m", "t", "u":
				newCol := ""
				switch msg.String() {
				case "c":
					newCol = "cpu"
				case "p":
					newCol = "pid"
				case "n":
					newCol = "name"
				case "m":
					newCol = "mem"
				case "t":
					newCol = "time"
				case "u":
					newCol = "user"
				}

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
				return m, nil
			}

			// Table navigation or Tree scrolling
			var cmd tea.Cmd
			if m.listFocus == focusMain {
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
							debounceCmd := tea.Tick(200*time.Millisecond, func(_ time.Time) tea.Msg {
								return debounceMsg{id: id, pid: p.PID}
							})
							return m, tea.Batch(cmd, debounceCmd)
						}
					}
				}
				return m, cmd
			} else {
				m.treeViewport, cmd = m.treeViewport.Update(msg)
				return m, cmd
			}

		} else if m.state == stateDetail {
			switch msg.String() {
			case "esc", "q", "backspace":
				m.state = stateList
				m.selectedDetail = nil
				m.detailFocus = focusDetail
				return m, m.refreshProcesses()
			case "left", "h":
				m.detailFocus = focusDetail
				return m, nil
			case "right", "l":
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

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		availableWidth := msg.Width - 6
		if availableWidth < 0 {
			availableWidth = 0
		}

		listWidth := int(float64(availableWidth) * 0.7)
		treeWidth := availableWidth - listWidth - 6

		contentHeight := msg.Height - 11
		if contentHeight < 5 {
			contentHeight = 5
		}

		// Fixed widths: PID(8) + Name(20) + User(12) + CPU(6) + Mem(16) + Started(19) = 81
		fixedColumnsWidth := 81
		cmdWidth := listWidth - fixedColumnsWidth - 12
		if cmdWidth < 10 {
			cmdWidth = 10
		}

		columns := m.getColumns()
		columns[6].Width = cmdWidth

		m.table.SetColumns(columns)

		m.table.SetWidth(listWidth)
		m.table.SetHeight(contentHeight)

		treeVpHeight := contentHeight - 2
		if treeVpHeight < 0 {
			treeVpHeight = 0
		}

		m.treeViewport.Width = treeWidth
		m.treeViewport.Height = treeVpHeight

		detailWidth := int(float64(availableWidth) * 0.7)
		envWidth := availableWidth - detailWidth - 2

		vpHeight := msg.Height - 9
		if vpHeight < 0 {
			vpHeight = 0
		}

		m.viewport.Width = detailWidth - 2
		m.viewport.Height = vpHeight

		m.envViewport.Width = envWidth
		if m.envViewport.Width < 0 {
			m.envViewport.Width = 0
		}
		m.envViewport.Height = vpHeight

	case []model.Process:
		// Capture current selection before update
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

		// Update cursor
		if len(m.filtered) > 0 {
			if !found {
				newIdx = 0
			}
			m.table.SetCursor(newIdx)

			m.selectionID++
			id := m.selectionID
			p := m.filtered[newIdx]
			return m, tea.Tick(200*time.Millisecond, func(_ time.Time) tea.Msg {
				return debounceMsg{id: id, pid: p.PID}
			})
		}

	case debounceMsg:
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

	case treeMsg:
		selected := m.table.SelectedRow()
		if len(selected) > 0 {
			var currentPID int
			fmt.Sscanf(selected[0], "%d", &currentPID)
			if model.Result(msg).Process.PID == currentPID {
				m.updateTreeViewport(model.Result(msg))
			}
		}

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

	if len(res.Process.Env) > 0 {
		for _, env := range res.Process.Env {
			fmt.Fprintf(&b, "%s\n", env)
		}
	} else {
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // Dimmed Gray
		fmt.Fprintf(&b, "%s\n", dimStyle.Render("No environment variables found."))
	}

	content := b.String()
	if m.envViewport.Width > 0 {
		content = wrap.String(content, m.envViewport.Width)
	}
	m.envViewport.SetContent(content)
}

func (m *MainModel) sortProcesses() {
	sort.Slice(m.processes, func(i, j int) bool {
		var less bool
		switch m.sortCol {
		case "pid":
			less = m.processes[i].PID < m.processes[j].PID
		case "name":
			less = strings.ToLower(m.processes[i].Command) < strings.ToLower(m.processes[j].Command)
		case "user":
			less = strings.ToLower(m.processes[i].User) < strings.ToLower(m.processes[j].User)
		case "cpu":
			less = m.processes[i].CPUPercent < m.processes[j].CPUPercent
		case "mem":
			less = m.processes[i].MemoryRSS < m.processes[j].MemoryRSS
		case "time":
			less = m.processes[i].StartedAt.Before(m.processes[j].StartedAt)
		default:
			less = m.processes[i].MemoryRSS < m.processes[j].MemoryRSS
		}
		if m.sortDesc {
			return !less
		}
		return less
	})
}

func (m *MainModel) getColumns() []table.Column {
	cols := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "User", Width: 12},
		{Title: "CPU%", Width: 6},
		{Title: "Mem", Width: 16},
		{Title: "Started", Width: 19},
		{Title: "Command", Width: 50},
	}

	addArrow := func(idx int, key string) {
		if m.sortCol == key {
			if m.sortDesc {
				cols[idx].Title += " ↓"
			} else {
				cols[idx].Title += " ↑"
			}
		}
	}

	addArrow(0, "pid")
	addArrow(1, "name")
	addArrow(2, "user")
	addArrow(3, "cpu")
	addArrow(4, "mem")
	addArrow(5, "time")

	return cols
}

func formatBytes(bytes uint64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := uint64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (m *MainModel) filterProcesses() {
	filter := strings.ToLower(m.input.Value())
	var rows []table.Row

	m.filtered = nil
	for _, p := range m.processes {
		cmd := strings.ToLower(p.Command)

		match := false
		if filter == "" {
			match = true
		} else {
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
				p.Command,
				p.User,
				fmt.Sprintf("%.1f%%", p.CPUPercent),
				fmt.Sprintf("%s (%.1f%%)", formatBytes(p.MemoryRSS), p.MemoryPercent),
				startedStr,
				p.Cmdline,
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

	output.RenderStandard(&b, res, true, true)

	content := b.String()
	if m.viewport.Width > 0 {
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

	treeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true).Render("Ancestry Tree:") // Lavender
	fmt.Fprintf(&b, "%s\n", treeLabel)

	ancestry := res.Ancestry
	if len(ancestry) == 0 {
		if res.Process.PID > 0 {
			ancestry = []model.Process{res.Process}
		} else {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243")) // Dimmed Gray
			fmt.Fprintf(&b, "  %s\n", dimStyle.Render("No ancestry found"))
		}
	}

	if len(ancestry) > 0 {
		output.PrintTree(&b, ancestry, res.Children, true)
	}

	if res.Process.Cmdline != "" {
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("141")).Bold(true).Render("Command:") // Lavender
		fmt.Fprintf(&b, "\n%s\n%s\n", label, res.Process.Cmdline)
	}

	content := b.String()
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

	outerStyle := baseStyle.
		Width(m.width-2).
		Height(m.height-2).
		Padding(0, 1)

	if m.state == stateList {
		status := "Mode: Navigation (Press / to search)"
		if m.input.Focused() {
			status = "Mode: Searching (Press Esc/Enter to stop)"
		}

		activeBorderColor := lipgloss.Color("62") // Purple/Blue
		dimBorderColor := lipgloss.Color("240")   // Dark Gray

		treeBorderColor := dimBorderColor
		treeHeaderColor := dimBorderColor

		if m.listFocus == focusSide {
			treeBorderColor = activeBorderColor
			treeHeaderColor = activeBorderColor
		} else {
			treeHeaderColor = lipgloss.Color("250") // Light Gray
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

		// Reconstruct table styles to update focus colors
		s := table.DefaultStyles()
		if m.listFocus == focusMain {
			s.Header = tableHeaderStyle.BorderForeground(activeBorderColor)
		} else {
			s.Header = tableHeaderStyle.BorderForeground(dimBorderColor)
		}
		s.Selected = s.Selected.
			Foreground(lipgloss.Color("229")). // Light Yellow
			Background(lipgloss.Color("56")).  // Purple
			Bold(false)
		m.table.SetStyles(s)

		mainContent := lipgloss.JoinHorizontal(lipgloss.Top,
			m.table.View(),
			treeContainerStyle.Render(
				lipgloss.JoinVertical(lipgloss.Left,
					treeHeaderStyle.Render(treeHeader),
					lipgloss.NewStyle().PaddingLeft(1).Render(m.treeViewport.View()),
				),
			),
		)

		helpText := fmt.Sprintf("Total: %d | Enter: Detail | Sort: p/n/u/c/m/t | Esc/q: Quit | Tab: Focus | Up/Down: Scroll", len(m.filtered))
		footerContent := helpText
		if m.version != "" {
			gap := m.width - 6 - lipgloss.Width(helpText) - lipgloss.Width(m.version)
			if gap > 0 {
				footerContent = helpText + strings.Repeat(" ", gap) + m.version
			}
		}

		return outerStyle.Render(
			lipgloss.JoinVertical(lipgloss.Left,
				titleStyle.MarginBottom(1).Render("witr"),
				lipgloss.NewStyle().MarginBottom(1).PaddingLeft(1).Render(fmt.Sprintf("%s", status)),
				lipgloss.NewStyle().MarginBottom(1).PaddingLeft(1).Render(m.input.View()),
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

		activeBorderColor := lipgloss.Color("62") // Purple
		dimColor := lipgloss.Color("250")         // Lighter Gray
		dimBorderColor := lipgloss.Color("240")   // Dark Gray

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
			Background(lipgloss.Color("2")).   // Green
			Foreground(lipgloss.Color("231")). // White
			Padding(0, 1).
			Bold(true)

		headerComponents := []string{
			titleStyle.Render("witr"),
		}
		if m.selectedDetail != nil {
			headerComponents = append(headerComponents, pidStyle.Render(fmt.Sprintf("PID %d", m.selectedDetail.Process.PID)))
		}

		helpText := "Esc/q: Back | Tab: Focus | Up/Down: Scroll"
		footerContent := helpText
		if m.version != "" {
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
