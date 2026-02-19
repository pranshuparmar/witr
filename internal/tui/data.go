package tui

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wrap"
	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	"github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/pkg/model"
)

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
		return filteredProcs
	}
}

func (m MainModel) refreshPorts() tea.Cmd {
	return func() tea.Msg {
		ports, err := proc.ListOpenPorts()
		if err != nil {
			return nil
		}
		return ports
	}
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

func (m *MainModel) sortPorts() {
	sort.Slice(m.ports, func(i, j int) bool {
		var less bool
		switch m.sortPortCol {
		case "port":
			less = m.ports[i].Port < m.ports[j].Port
		case "proto":
			less = strings.ToLower(m.ports[i].Protocol) < strings.ToLower(m.ports[j].Protocol)
		case "addr":
			less = strings.ToLower(m.ports[i].Address) < strings.ToLower(m.ports[j].Address)
		case "state":
			less = strings.ToLower(m.ports[i].State) < strings.ToLower(m.ports[j].State)
		default:
			less = m.ports[i].Port < m.ports[j].Port
		}
		if m.sortPortDesc {
			return !less
		}
		return less
	})
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
		if exp >= 5 { //avoid index out of range
			break
		}
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
				p.User,
				p.Command,
				fmt.Sprintf("%.1f%%", p.CPUPercent),
				fmt.Sprintf("%s (%.1f%%)", formatBytes(p.MemoryRSS), p.MemoryPercent),
				startedStr,
				p.Cmdline,
			})
		}
	}
	m.table.SetRows(rows)
}

func (m *MainModel) getColumns() []table.Column {
	cols := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "User", Width: 12},
		{Title: "Name", Width: 20},
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
	addArrow(1, "user")
	addArrow(2, "name")
	addArrow(3, "cpu")
	addArrow(4, "mem")
	addArrow(5, "time")
	addArrow(6, "cmd")

	return cols
}

func (m *MainModel) getPortColumns() []table.Column {
	cols := []table.Column{
		{Title: "Port", Width: 6},
		{Title: "Protocol", Width: 10},
		{Title: "Address", Width: 30},
		{Title: "State", Width: 20},
	}

	addArrow := func(idx int, key string) {
		if m.sortPortCol == key {
			if m.sortPortDesc {
				cols[idx].Title += " ↓"
			} else {
				cols[idx].Title += " ↑"
			}
		}
	}

	addArrow(0, "port")
	addArrow(1, "proto")
	addArrow(2, "addr")
	addArrow(3, "state")

	return cols
}

func (m *MainModel) updatePortTable() {
	m.sortPorts()

	var rows []table.Row
	filter := strings.ToLower(m.portInput.Value())
	seen := make(map[string]bool)

	existingCols := m.portTable.Columns()
	newCols := m.getPortColumns()
	for i := range existingCols {
		if i < len(newCols) {
			newCols[i].Width = existingCols[i].Width
		}
	}
	m.portTable.SetColumns(newCols)

	procMap := make(map[int]model.Process)
	for _, p := range m.processes {
		procMap[p.PID] = p
	}

	for _, p := range m.ports {
		match := false
		if filter == "" {
			match = true
		} else {
			if strings.Contains(fmt.Sprintf("%d", p.Port), filter) ||
				strings.Contains(strings.ToLower(p.Protocol), filter) ||
				strings.Contains(strings.ToLower(p.Address), filter) ||
				strings.Contains(strings.ToLower(p.State), filter) {
				match = true
			}
		}

		if match {
			if !m.showAllPorts && p.State != "LISTEN" && p.State != "OPEN" {
				continue
			}

			key := fmt.Sprintf("%d|%s|%s|%s", p.Port, p.Protocol, p.Address, p.State)

			if !seen[key] {
				seen[key] = true
				rows = append(rows, table.Row{
					fmt.Sprintf("%d", p.Port),
					p.Protocol,
					p.Address,
					p.State,
				})
			}
		}
	}

	m.portTable.SetRows(rows)
	m.updatePortDetails()
}

func (m *MainModel) updatePortDetails() {
	selected := m.portTable.SelectedRow()
	if len(selected) < 4 {
		m.portDetailTable.SetRows(nil)
		return
	}

	portStr := selected[0]
	protocol := selected[1]
	address := selected[2]
	state := selected[3]

	port, _ := strconv.Atoi(portStr)

	var rows []table.Row
	seen := make(map[int]bool)

	procMap := make(map[int]model.Process)
	for _, p := range m.processes {
		procMap[p.PID] = p
	}

	for _, p := range m.ports {
		if p.Port == port && p.Protocol == protocol && p.Address == address && p.State == state {
			if !seen[p.PID] {
				seen[p.PID] = true
				if proc, ok := procMap[p.PID]; ok {
					cmd := proc.Cmdline
					cols := m.portDetailTable.Columns()
					if len(cols) > 3 {
						width := cols[3].Width
						if width > 3 && len(cmd) > width {
							cmd = cmd[:width-3] + "..."
						}
					}
					rows = append(rows, table.Row{
						fmt.Sprintf("%d", proc.PID),
						proc.User,
						proc.Command,
						cmd,
					})
				} else {
					rows = append(rows, table.Row{
						fmt.Sprintf("%d", p.PID),
						"???",
						"???",
						"???",
					})
				}
			}
		}
	}
	m.portDetailTable.SetRows(rows)
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
		dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))
		fmt.Fprintf(&b, "%s\n", dimStyle.Render("No environment variables found."))
	}

	content := b.String()
	if m.envViewport.Width > 0 {
		content = wrap.String(content, m.envViewport.Width)
	}
	m.envViewport.SetContent(content)
}

func (m *MainModel) updateTreeViewport(res model.Result) {
	if len(res.Ancestry) == 0 && res.Process.PID == 0 {
		m.treeViewport.SetContent("")
		return
	}
	var b strings.Builder

	treeLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("#af87ff")).Bold(true).Render("Ancestry Tree:")
	fmt.Fprintf(&b, "%s\n", treeLabel)

	ancestry := res.Ancestry
	if len(ancestry) == 0 {
		if res.Process.PID > 0 {
			ancestry = []model.Process{res.Process}
		} else {
			dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#767676"))
			fmt.Fprintf(&b, "  %s\n", dimStyle.Render("No ancestry found"))
		}
	}

	if len(ancestry) > 0 {
		output.PrintTree(&b, ancestry, res.Children, true)
	}

	if res.Process.Cmdline != "" {
		label := lipgloss.NewStyle().Foreground(lipgloss.Color("#af87ff")).Bold(true).Render("Command:")
		fmt.Fprintf(&b, "\n%s\n%s\n", label, res.Process.Cmdline)
	}

	content := b.String()
	if m.treeViewport.Width > 0 {
		content = wrap.String(content, m.treeViewport.Width)
	}
	m.treeViewport.SetContent(content)
}
