package tui

import (
	"cmp"
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
			return err
		}
		return ports
	}
}

func (m MainModel) refreshContainers() tea.Cmd {
	return func() tea.Msg {
		return proc.ListAllContainers()
	}
}

func (m MainModel) refreshLocks() tea.Cmd {
	return func() tea.Msg {
		if m.showAllFiles {
			return mergeLocksAndOpenFiles(proc.ListLockedFiles(), proc.ListAllOpenFiles())
		}
		return proc.ListLockedFiles()
	}
}

func (m MainModel) refreshNetworks() tea.Cmd {
	return func() tea.Msg {
		return proc.ListNetworkSnapshot()
	}
}

// mergeLocksAndOpenFiles returns the union of locks and open files. When the
// same (pid, path) appears in both, the lock entry wins so the Type column
// surfaces the lock kind (POSIX/FLOCK) and Mode reflects lock access rather
// than the raw fd access mode.
func mergeLocksAndOpenFiles(locks, opens []*model.LockedFile) []*model.LockedFile {
	seen := make(map[string]struct{}, len(locks)+len(opens))
	key := func(l *model.LockedFile) string {
		return fmt.Sprintf("%d\x00%s", l.PID, l.Path)
	}

	merged := make([]*model.LockedFile, 0, len(locks)+len(opens))
	for _, l := range locks {
		merged = append(merged, l)
		seen[key(l)] = struct{}{}
	}
	for _, o := range opens {
		if _, dup := seen[key(o)]; dup {
			continue
		}
		merged = append(merged, o)
	}
	return merged
}

// fetchContainerDetail mirrors `witr -c <name> --verbose`: enrich the match,
// try the host-visible PID path through the normal pipeline, fall back to
// returning the bare ContainerMatch for the runtime-info render.
func (m MainModel) fetchContainerDetail(match *model.ContainerMatch) tea.Cmd {
	return func() tea.Msg {
		proc.EnrichContainer(match)
		pid := proc.ResolveContainerHostPID(match.Runtime, match.ID)
		if pid > 0 && proc.PIDBelongsToContainer(pid, match.ID) {
			res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
				PID:     pid,
				Verbose: true,
				Tree:    true,
			})
			if err == nil {
				res.Process.Container = output.FormatContainerLine(match)
				if len(res.Ancestry) > 0 {
					res.Ancestry[len(res.Ancestry)-1].Container = res.Process.Container
				}
				return res
			}
		}
		return match
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

type processSorter struct {
	procs []model.Process
	keys  []string // pre-computed lowercase keys (nil for non-string sorts)
	col   string
	desc  bool
}

func (s processSorter) Len() int { return len(s.procs) }

func (s processSorter) Swap(i, j int) {
	s.procs[i], s.procs[j] = s.procs[j], s.procs[i]
	if s.keys != nil {
		s.keys[i], s.keys[j] = s.keys[j], s.keys[i]
	}
}

func (s processSorter) Less(i, j int) bool {
	var n int
	switch s.col {
	case "pid":
		n = cmp.Compare(s.procs[i].PID, s.procs[j].PID)
	case "name", "user":
		n = cmp.Compare(s.keys[i], s.keys[j])
	case "cpu":
		n = cmp.Compare(s.procs[i].CPUPercent, s.procs[j].CPUPercent)
	case "time":
		n = cmp.Compare(s.procs[i].StartedAt.UnixNano(), s.procs[j].StartedAt.UnixNano())
	default: // "mem"
		n = cmp.Compare(s.procs[i].MemoryRSS, s.procs[j].MemoryRSS)
	}
	if n == 0 {
		n = cmp.Compare(s.procs[i].PID, s.procs[j].PID)
	}
	if s.desc {
		return n > 0
	}
	return n < 0
}

func (m *MainModel) sortProcesses() {
	s := processSorter{
		procs: m.processes,
		col:   m.sortCol,
		desc:  m.sortDesc,
	}

	if m.sortCol == "name" || m.sortCol == "user" {
		s.keys = make([]string, len(m.processes))
		for i := range m.processes {
			if m.sortCol == "name" {
				s.keys[i] = strings.ToLower(m.processes[i].Command)
			} else {
				s.keys[i] = strings.ToLower(m.processes[i].User)
			}
		}
	}

	sort.Stable(s)
}

func (m *MainModel) sortPorts() {
	sort.Slice(m.ports, func(i, j int) bool {
		less := lessPorts(m.ports[i], m.ports[j], m.sortPortCol)
		if m.sortPortDesc {
			return !less
		}
		return less
	})
}

// lessPorts orders two ports by the given column (ascending). Unknown columns
// fall back to port number.
func lessPorts(a, b model.OpenPort, col string) bool {
	switch col {
	case "proto":
		return strings.ToLower(a.Protocol) < strings.ToLower(b.Protocol)
	case "addr":
		return strings.ToLower(a.Address) < strings.ToLower(b.Address)
	case "state":
		return strings.ToLower(a.State) < strings.ToLower(b.State)
	default: // "port" and anything unspecified
		return a.Port < b.Port
	}
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

// processMatches reports whether p matches the already-lowercased filter on any
// of its command, PID, user, or full command line.
func processMatches(p model.Process, filter string) bool {
	return strings.Contains(strings.ToLower(p.Command), filter) ||
		strings.Contains(strconv.Itoa(p.PID), filter) ||
		strings.Contains(strings.ToLower(p.User), filter) ||
		strings.Contains(strings.ToLower(p.Cmdline), filter)
}

func (m *MainModel) filterProcesses() {
	filter := strings.ToLower(m.input.Value())
	var rows []table.Row

	cols := m.table.Columns()
	colWidth := func(i int) int {
		if i < len(cols) {
			return cols[i].Width
		}
		return 20
	}
	nameWidth := colWidth(2)
	cmdlineWidth := colWidth(6)

	m.filtered = nil
	for _, p := range m.processes {
		match := filter == "" || processMatches(p, filter)

		if match {
			m.filtered = append(m.filtered, p)
			startedStr := p.StartedAt.Format("Jan 02 15:04:05")
			if p.StartedAt.IsZero() {
				startedStr = ""
			}

			row := table.Row{
				fmt.Sprintf("%8d", p.PID),
				output.SanitizeTerminalLine(p.User),
				truncateMiddle(output.SanitizeTerminalLine(p.Command), nameWidth),
				fmt.Sprintf("%6s", fmt.Sprintf("%.1f%%", p.CPUPercent)),
				fmt.Sprintf("%16s", fmt.Sprintf("%s (%.1f%%)", formatBytes(p.MemoryRSS), p.MemoryPercent)),
				startedStr,
			}
			if m.showCmdCol {
				row = append(row, truncateMiddle(output.SanitizeTerminalLine(p.Cmdline), cmdlineWidth))
			}
			rows = append(rows, row)
		}
	}
	m.table.SetRows(rows)
}

func centerHeader(title string, width int) string {
	w := lipgloss.Width(title)
	if w >= width {
		return title
	}
	pad := width - w
	left := pad / 2
	return strings.Repeat(" ", left) + title
}

func (m *MainModel) getColumns() []table.Column {
	cols := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "User", Width: 12},
		{Title: "Name", Width: 20},
		{Title: "CPU%", Width: 6},
		{Title: "Mem", Width: 16},
		{Title: "Started", Width: 19},
	}
	if m.showCmdCol {
		cols = append(cols, table.Column{Title: "Command", Width: 50})
	}

	addArrow := func(idx int, key string) {
		if idx < len(cols) && m.sortCol == key {
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

	// Center-align numeric column headers
	for _, idx := range []int{0, 3, 4} {
		if idx < len(cols) {
			cols[idx].Title = centerHeader(cols[idx].Title, cols[idx].Width)
		}
	}

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

	// Center-align the Port column header
	cols[0].Title = centerHeader(cols[0].Title, cols[0].Width)

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
					fmt.Sprintf("%6d", p.Port),
					p.Protocol,
					p.Address,
					p.State,
				})
			}
		}
	}

	m.portTable.SetRows(rows)
	m.updatePortDetailsWithMap(procMap)
}

func (m *MainModel) updatePortDetails() {
	procMap := make(map[int]model.Process, len(m.processes))
	for _, p := range m.processes {
		procMap[p.PID] = p
	}
	m.updatePortDetailsWithMap(procMap)
}

func (m *MainModel) updatePortDetailsWithMap(procMap map[int]model.Process) {
	selected := m.portTable.SelectedRow()
	if len(selected) < 4 {
		m.portDetailTable.SetRows(nil)
		return
	}

	portStr := strings.TrimSpace(selected[0])
	protocol := selected[1]
	address := selected[2]
	state := selected[3]

	port, _ := strconv.Atoi(portStr)

	var rows []table.Row
	seen := make(map[int]bool)

	for _, p := range m.ports {
		if p.Port == port && p.Protocol == protocol && p.Address == address && p.State == state {
			if !seen[p.PID] {
				seen[p.PID] = true
				if proc, ok := procMap[p.PID]; ok {
					cmd := output.SanitizeTerminalLine(proc.Cmdline)
					cols := m.portDetailTable.Columns()
					if len(cols) > 3 {
						width := cols[3].Width
						if width > 3 && len(cmd) > width {
							cmd = cmd[:width-3] + "..."
						}
					}
					rows = append(rows, table.Row{
						fmt.Sprintf("%8d", proc.PID),
						output.SanitizeTerminalLine(proc.User),
						output.SanitizeTerminalLine(proc.Command),
						cmd,
					})
				} else {
					rows = append(rows, table.Row{
						fmt.Sprintf("%8d", p.PID),
						"???",
						"???",
						"???",
					})
				}
			}
		}
	}
	m.portDetailTable.SetRows(rows)
	if len(rows) > 0 {
		m.portDetailTable.SetCursor(0)
	}
}

func (m *MainModel) updateDetailViewport() {
	var b strings.Builder
	switch {
	case m.selectedDetail != nil:
		// Process detail shares space with the env pane on the right.
		if m.width > 6 {
			availableWidth := m.width - 6
			detailViewWidth := int(float64(availableWidth) * detailPaneRatio)
			m.viewport.Width = detailViewWidth - 4
			if m.viewport.Width < 1 {
				m.viewport.Width = 1
			}
		}
		output.RenderStandard(&b, *m.selectedDetail, true, true)
	case m.selectedContainer != nil:
		// Container detail occupies the full width — no env pane to share with.
		if w := m.width - 6; w > 0 {
			m.viewport.Width = w
		}
		label := "container " + m.selectedContainer.Name
		output.RenderContainerFallback(&b, label, m.selectedContainer, true, true)
	default:
		return
	}

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
			fmt.Fprintf(&b, "%s\n", output.SanitizeTerminalLine(env))
		}
	} else {
		dimStyle := lipgloss.NewStyle().Foreground(colorMuted)
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
		m.treePIDs = nil
		m.treeCursor = 0
		m.treeResult = nil
		m.treeAncestry = nil
		return
	}

	ancestry := res.Ancestry
	if len(ancestry) == 0 {
		if res.Process.PID > 0 {
			ancestry = []model.Process{res.Process}
		}
	}

	isRefresh := m.treeTargetPID == res.Process.PID

	resCopy := res
	m.treeResult = &resCopy
	m.treeAncestry = ancestry
	m.treeTargetPID = res.Process.PID

	// Build the flat PID list: ancestry then children (max 10)
	var newPIDs []int
	for _, p := range ancestry {
		newPIDs = append(newPIDs, p.PID)
	}
	childLimit := 10
	if len(res.Children) < childLimit {
		childLimit = len(res.Children)
	}
	for i := 0; i < childLimit; i++ {
		newPIDs = append(newPIDs, res.Children[i].PID)
	}

	// Preserve cursor position only on periodic refresh of the same process
	restored := false
	if isRefresh {
		oldCursor := m.treeCursor
		oldPID := 0
		if oldCursor >= 0 && oldCursor < len(m.treePIDs) {
			oldPID = m.treePIDs[oldCursor]
		}
		if oldPID > 0 {
			for i, pid := range newPIDs {
				if pid == oldPID {
					m.treeCursor = i
					restored = true
					break
				}
			}
		}
	}
	m.treePIDs = newPIDs

	if !restored {
		if len(ancestry) > 0 {
			m.treeCursor = len(ancestry) - 1
		} else {
			m.treeCursor = 0
		}
	}

	m.renderTreeContent(res, ancestry)
}

// rerenderTree re-renders the tree viewport with the current cursor position.
func (m *MainModel) rerenderTree() {
	if m.treeResult == nil {
		return
	}
	m.renderTreeContent(*m.treeResult, m.treeAncestry)
}

func (m *MainModel) renderTreeContent(res model.Result, ancestry []model.Process) {
	var b strings.Builder

	magenta := lipgloss.NewStyle().Foreground(colorTreeConn)
	green := lipgloss.NewStyle().Foreground(colorTreeTarget)
	highlight := lipgloss.NewStyle().
		Background(colorSelectBg).
		Foreground(colorSelectFg)
	dim := lipgloss.NewStyle().Foreground(colorMuted)
	sectionLabel := lipgloss.NewStyle().Foreground(colorSectionLabel).Bold(true)

	fmt.Fprintf(&b, "%s\n", sectionLabel.Render("Ancestry Tree:"))

	if len(ancestry) == 0 {
		fmt.Fprintf(&b, "  %s\n", dim.Render("No ancestry found"))
	}

	idx := 0
	for i, proc := range ancestry {
		indent := strings.Repeat("  ", i)
		if i > 0 {
			fmt.Fprintf(&b, "%s%s ", indent, magenta.Render("└─"))
		}

		label := fmt.Sprintf("%s (pid %d)", output.SanitizeTerminalLine(output.ChainName(proc)), proc.PID)
		if idx == m.treeCursor {
			label = highlight.Render(label)
		} else if i == len(ancestry)-1 {
			label = green.Render(label)
		}
		fmt.Fprintf(&b, "%s\n", label)
		idx++
	}

	children := res.Children
	limit := 10
	count := len(children)
	if count > 0 {
		baseIndent := strings.Repeat("  ", len(ancestry))
		for i, child := range children {
			if i >= limit {
				remaining := count - limit
				fmt.Fprintf(&b, "%s%s ... and %d more\n", baseIndent, magenta.Render("└─"), remaining)
				break
			}
			connector := "├─"
			isLast := (i == count-1) || (i == limit-1 && count <= limit)
			if isLast {
				connector = "└─"
			}

			label := fmt.Sprintf("%s (pid %d)", output.SanitizeTerminalLine(output.ChainName(child)), child.PID)
			if idx == m.treeCursor {
				label = highlight.Render(label)
			}
			fmt.Fprintf(&b, "%s%s %s\n", baseIndent, magenta.Render(connector), label)
			idx++
		}
	}

	if res.Process.Cmdline != "" {
		fmt.Fprintf(&b, "\n%s\n%s\n", sectionLabel.Render("Command:"), output.SanitizeTerminalLine(res.Process.Cmdline))
	}

	content := b.String()
	if m.treeViewport.Width > 0 {
		content = wrap.String(content, m.treeViewport.Width)
	}
	m.treeViewport.SetContent(content)
}

func (m *MainModel) sortContainers() {
	col := m.sortContainerCol
	desc := m.sortContainerDesc
	sort.SliceStable(m.containers, func(i, j int) bool {
		a, b := m.containers[i], m.containers[j]
		var less bool
		switch col {
		case "id":
			less = a.ID < b.ID
		case "runtime":
			less = strings.ToLower(a.Runtime) < strings.ToLower(b.Runtime)
		case "image":
			less = strings.ToLower(a.Image) < strings.ToLower(b.Image)
		case "status":
			less = strings.ToLower(a.Status) < strings.ToLower(b.Status)
		default: // "name"
			less = strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		if desc {
			return !less
		}
		return less
	})
}

func (m *MainModel) sortLocks() {
	col := m.sortLockCol
	desc := m.sortLockDesc
	sort.SliceStable(m.locks, func(i, j int) bool {
		a, b := m.locks[i], m.locks[j]
		var less bool
		switch col {
		case "process":
			less = strings.ToLower(a.Process) < strings.ToLower(b.Process)
		case "type":
			less = a.Type < b.Type
		case "mode":
			less = a.Mode < b.Mode
		case "path":
			less = a.Path < b.Path
		default: // "pid"
			less = a.PID < b.PID
		}
		if desc {
			return !less
		}
		return less
	})
}

// getContainerColumns returns container columns with sort-arrow suffixes.
func (m *MainModel) getContainerColumns() []table.Column {
	cols := []table.Column{
		{Title: "ID", Width: 14},
		{Title: "Name", Width: 22},
		{Title: "Runtime", Width: 10},
		{Title: "Image", Width: 28},
		{Title: "Status", Width: 22},
		{Title: "Ports", Width: 24},
		{Title: "Command", Width: 28},
	}
	arrow := " ↑"
	if m.sortContainerDesc {
		arrow = " ↓"
	}
	switch m.sortContainerCol {
	case "id":
		cols[0].Title += arrow
	case "name":
		cols[1].Title += arrow
	case "runtime":
		cols[2].Title += arrow
	case "image":
		cols[3].Title += arrow
	case "status":
		cols[4].Title += arrow
	}
	return cols
}

// getLockColumns returns lock columns with sort-arrow suffixes.
func (m *MainModel) getLockColumns() []table.Column {
	cols := []table.Column{
		{Title: centerHeader("PID", 8), Width: 8},
		{Title: "Process", Width: 18},
		{Title: "Type", Width: 8},
		{Title: "Mode", Width: 8},
		{Title: "Path", Width: 50},
	}
	arrow := " ↑"
	if m.sortLockDesc {
		arrow = " ↓"
	}
	switch m.sortLockCol {
	case "pid":
		cols[0].Title = centerHeader("PID"+arrow, 8)
	case "process":
		cols[1].Title += arrow
	case "type":
		cols[2].Title += arrow
	case "mode":
		cols[3].Title += arrow
	case "path":
		cols[4].Title += arrow
	}
	return cols
}

func (m *MainModel) updateContainerTable() {
	m.sortContainers()
	filter := strings.ToLower(strings.TrimSpace(m.containerInput.Value()))

	// Preserve any width changes the WindowSizeMsg handler made (the trailing
	// Command column flexes) before re-applying header arrows.
	existing := m.containerTable.Columns()
	newCols := m.getContainerColumns()
	for i := range existing {
		if i < len(newCols) {
			newCols[i].Width = existing[i].Width
		}
	}
	m.containerTable.SetColumns(newCols)

	cols := m.containerTable.Columns()
	w := func(i int) int {
		if i < len(cols) {
			return cols[i].Width
		}
		return 20
	}

	rows := make([]table.Row, 0, len(m.containers))
	filtered := make([]*model.ContainerMatch, 0, len(m.containers))
	for _, c := range m.containers {
		if filter != "" {
			haystack := strings.ToLower(c.ID + " " + c.Name + " " + c.Runtime + " " + c.Image + " " + c.Status + " " + c.Ports + " " + c.Command)
			if !strings.Contains(haystack, filter) {
				continue
			}
		}
		rows = append(rows, table.Row{
			output.SanitizeTerminalLine(output.ShortContainerID(c.ID)),
			truncate(output.SanitizeTerminalLine(c.Name), w(1)),
			output.SanitizeTerminalLine(c.Runtime),
			truncateMiddle(output.SanitizeTerminalLine(c.Image), w(3)),
			truncate(output.SanitizeTerminalLine(c.Status), w(4)),
			truncate(output.SanitizeTerminalLine(c.Ports), w(5)),
			truncateMiddle(output.SanitizeTerminalLine(c.Command), w(6)),
		})
		filtered = append(filtered, c)
	}
	m.containerTable.SetRows(rows)
	m.filteredContainers = filtered
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}

// truncateMiddle preserves the head and tail of s and replaces the middle
// with an ellipsis when the string is wider than n. Better than head-only
// truncation for paths and command lines where the unique part is usually
// at the end (e.g. .../Google Chrome Helper).
func truncateMiddle(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	// Below ~8 chars there's no room to show meaningful head+tail; fall
	// back to head truncation rather than emit something like "/…r".
	if n < 8 {
		return string(runes[:n-1]) + "…"
	}
	keep := n - 1 // reserve one cell for the ellipsis
	head := keep / 2
	tail := keep - head
	return string(runes[:head]) + "…" + string(runes[len(runes)-tail:])
}

func (m *MainModel) updateLockTable() {
	m.sortLocks()
	filter := strings.ToLower(strings.TrimSpace(m.lockInput.Value()))

	existing := m.lockTable.Columns()
	newCols := m.getLockColumns()
	for i := range existing {
		if i < len(newCols) {
			newCols[i].Width = existing[i].Width
		}
	}
	m.lockTable.SetColumns(newCols)

	cols := m.lockTable.Columns()
	w := func(i int) int {
		if i < len(cols) {
			return cols[i].Width
		}
		return 20
	}

	// In "all open files" mode with no search, cap rows to keep the UI snappy.
	// Typing into the search box lifts the cap so users can drill into the full set.
	rowLimit := 0
	if m.showAllFiles && filter == "" {
		rowLimit = openFilesDisplayCap
	}

	rows := make([]table.Row, 0, len(m.locks))
	filtered := make([]*model.LockedFile, 0, len(m.locks))
	for _, l := range m.locks {
		if filter != "" {
			haystack := strings.ToLower(fmt.Sprintf("%d %s %s %s %s", l.PID, l.Process, l.Type, l.Mode, l.Path))
			if !strings.Contains(haystack, filter) {
				continue
			}
		}
		if rowLimit > 0 && len(rows) >= rowLimit {
			filtered = append(filtered, l)
			continue
		}
		rows = append(rows, table.Row{
			fmt.Sprintf("%8d", l.PID),
			truncate(output.SanitizeTerminalLine(l.Process), w(1)),
			truncate(output.SanitizeTerminalLine(l.Type), w(2)),
			truncate(output.SanitizeTerminalLine(l.Mode), w(3)),
			truncateMiddle(output.SanitizeTerminalLine(l.Path), w(4)),
		})
		filtered = append(filtered, l)
	}
	m.lockTable.SetRows(rows)
	m.filteredLocks = filtered
}

const openFilesDisplayCap = 100

func (m *MainModel) getNetworkColumns() []table.Column {
	cols := []table.Column{
		{Title: "Source", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "Kind", Width: 12},
		{Title: "State", Width: 10},
		{Title: "Address", Width: 22},
		{Title: "Extra", Width: 18},
	}
	addArrow := func(idx int, col string) {
		if m.sortNetworkCol == col {
			arrow := " ↑"
			if m.sortNetworkDesc {
				arrow = " ↓"
			}
			cols[idx].Title += arrow
		}
	}
	addArrow(0, "source")
	addArrow(1, "name")
	addArrow(2, "kind")
	addArrow(3, "state")
	addArrow(4, "address")
	addArrow(5, "extra")
	return cols
}

func (m *MainModel) rebuildNetworkEntries() {
	entries := make([]networkEntry, 0, len(m.hostIfaces)+len(m.networks))
	for _, h := range m.hostIfaces {
		entries = append(entries, networkEntry{IsHost: true, Host: h})
	}
	for _, n := range m.networks {
		entries = append(entries, networkEntry{IsHost: false, Net: n})
	}
	m.networkEntries = entries
}

func (m *MainModel) sortNetworkEntries(entries []networkEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		key := func(e networkEntry) (source, name, kind, state, address, extra string) {
			if e.IsHost {
				ips := strings.Join(e.Host.IPv4, ", ")
				gw := strings.Join(e.Host.Gateway, ", ")
				return "Host", e.Host.Name, e.Host.Type, e.Host.State, ips, gw
			}
			return "Docker", e.Net.Name, e.Net.Driver, e.Net.Scope, e.Net.Subnet, e.Net.Gateway
		}
		as, an, ak, ast, aa, ae := key(a)
		bs, bn, bk, bst, ba, be := key(b)
		var n int
		switch m.sortNetworkCol {
		case "source":
			n = cmp.Compare(as, bs)
		case "kind":
			n = cmp.Compare(strings.ToLower(ak), strings.ToLower(bk))
		case "state":
			n = cmp.Compare(strings.ToLower(ast), strings.ToLower(bst))
		case "address":
			n = cmp.Compare(aa, ba)
		case "extra":
			n = cmp.Compare(ae, be)
		default: // name
			n = cmp.Compare(strings.ToLower(an), strings.ToLower(bn))
		}
		if n == 0 {
			// Host rows before Docker when equal; then name.
			if a.IsHost != b.IsHost {
				if a.IsHost {
					return !m.sortNetworkDesc
				}
				return m.sortNetworkDesc
			}
			n = cmp.Compare(strings.ToLower(an), strings.ToLower(bn))
		}
		if m.sortNetworkDesc {
			return n > 0
		}
		return n < 0
	})
}

func (m *MainModel) updateNetworkTable() {
	m.rebuildNetworkEntries()
	m.sortNetworkEntries(m.networkEntries)

	filter := strings.ToLower(strings.TrimSpace(m.networkInput.Value()))
	existing := m.networkTable.Columns()
	newCols := m.getNetworkColumns()
	for i := range existing {
		if i < len(newCols) && existing[i].Width > 0 {
			newCols[i].Width = existing[i].Width
		}
	}
	m.networkTable.SetColumns(newCols)

	cols := m.networkTable.Columns()
	w := func(i int) int {
		if i < len(cols) {
			return cols[i].Width
		}
		return 16
	}

	rows := make([]table.Row, 0, len(m.networkEntries))
	filtered := make([]networkEntry, 0, len(m.networkEntries))
	for _, e := range m.networkEntries {
		if m.networkScope == networkScopeHost && !e.IsHost {
			continue
		}
		if m.networkScope == networkScopeDocker && e.IsHost {
			continue
		}
		var source, name, kind, state, address, extra string
		if e.IsHost {
			source = "Host"
			name = e.Host.Name
			kind = e.Host.Type
			state = e.Host.State
			address = strings.Join(e.Host.IPv4, ", ")
			extra = strings.Join(e.Host.Gateway, ", ")
			if extra == "" {
				extra = e.Host.MAC
			}
		} else {
			source = "Docker"
			name = e.Net.Name
			kind = e.Net.Driver
			state = e.Net.Scope
			if state == "" {
				state = "—"
			}
			address = e.Net.Subnet
			extra = e.Net.Gateway
			if extra == "N/A" {
				extra = e.Net.BridgeInterface
			}
		}
		if filter != "" {
			hay := strings.ToLower(fmt.Sprintf("%s %s %s %s %s %s %s",
				source, name, kind, state, address, extra, e.Host.Description))
			if !e.IsHost {
				hay = strings.ToLower(fmt.Sprintf("%s %s %s %s %s %s %s",
					source, name, kind, state, address, extra, e.Net.BridgeInterface))
			}
			if !strings.Contains(hay, filter) {
				continue
			}
		}
		filtered = append(filtered, e)
		rows = append(rows, table.Row{
			truncate(source, w(0)),
			truncate(output.SanitizeTerminalLine(name), w(1)),
			truncate(output.SanitizeTerminalLine(kind), w(2)),
			truncate(output.SanitizeTerminalLine(state), w(3)),
			truncate(output.SanitizeTerminalLine(address), w(4)),
			truncate(output.SanitizeTerminalLine(extra), w(5)),
		})
	}
	m.filteredEntries = filtered
	m.networkTable.SetRows(rows)
	m.updateNetworkDetails()
}

func (m *MainModel) updateNetworkDetails() {
	idx := m.networkTable.Cursor()
	if idx < 0 || idx >= len(m.filteredEntries) {
		m.networkDetailTable.SetRows(nil)
		return
	}
	e := m.filteredEntries[idx]
	if e.IsHost {
		m.updateHostIfaceDetails(e.Host)
		return
	}
	m.ensureNetworkEndpointColumns()
	net := e.Net
	cols := m.networkDetailTable.Columns()
	w := func(i int) int {
		if i < len(cols) {
			return cols[i].Width
		}
		return 16
	}
	rows := make([]table.Row, 0, len(net.Containers)+4)
	// Network metadata first, then endpoints.
	m.ensureNetworkDetailKVColumns()
	// Prefer endpoint table when containers exist; always show KV meta in side panel.
	// Use KV layout for a consistent host/docker experience.
	kv := [][2]string{
		{"Source", "Docker"},
		{"Name", net.Name},
		{"ID", net.ID},
		{"Driver", net.Driver},
		{"Scope", net.Scope},
		{"Bridge", net.BridgeInterface},
		{"Subnet", net.Subnet},
		{"Gateway", net.Gateway},
		{"Containers", fmt.Sprintf("%d", len(net.Containers))},
	}
	cols = m.networkDetailTable.Columns()
	w0, w1 := 12, 36
	if len(cols) > 0 {
		w0 = cols[0].Width
	}
	if len(cols) > 1 {
		w1 = cols[1].Width
	}
	for _, pair := range kv {
		val := pair[1]
		if val == "" {
			val = "—"
		}
		rows = append(rows, table.Row{
			truncate(output.SanitizeTerminalLine(pair[0]), w0),
			truncate(output.SanitizeTerminalLine(val), w1),
		})
	}
	for _, c := range net.Containers {
		st := c.Status
		if st == "" {
			st = "?"
		}
		label := fmt.Sprintf("→ %s %s", st, c.Name)
		rows = append(rows, table.Row{
			truncate(output.SanitizeTerminalLine(label), w0),
			truncate(output.SanitizeTerminalLine(c.IP), w1),
		})
	}
	// silence unused
	_ = w
	m.networkDetailTable.SetRows(rows)
}

func (m *MainModel) ensureNetworkEndpointColumns() {
	cols := m.networkDetailTable.Columns()
	want := []table.Column{
		{Title: "Status", Width: 8},
		{Title: "Name", Width: 18},
		{Title: "IP", Width: 20},
		{Title: "MAC", Width: 18},
	}
	if len(cols) == 4 && cols[0].Title == "Status" {
		return
	}
	// Preserve width budget from current table when possible.
	if len(cols) == 4 {
		for i := range want {
			if cols[i].Width > 0 {
				want[i].Width = cols[i].Width
			}
		}
	}
	m.networkDetailTable.SetColumns(want)
}

func (m *MainModel) ensureNetworkDetailKVColumns() {
	want := []table.Column{
		{Title: "Field", Width: 10},
		{Title: "Value", Width: 36},
	}
	cols := m.networkDetailTable.Columns()
	if len(cols) == 2 && cols[0].Title == "Field" {
		return
	}
	if len(cols) >= 2 {
		// Stretch value column from previous total width.
		total := 0
		for _, c := range cols {
			total += c.Width
		}
		if total > 14 {
			want[1].Width = total - want[0].Width
		}
	}
	m.networkDetailTable.SetColumns(want)
}

func (m *MainModel) updateHostIfaceDetails(iface model.HostInterface) {
	m.ensureNetworkDetailKVColumns()
	cols := m.networkDetailTable.Columns()
	w0, w1 := 12, 36
	if len(cols) > 0 {
		w0 = cols[0].Width
	}
	if len(cols) > 1 {
		w1 = cols[1].Width
	}
	mtu := "n/a"
	if iface.MTU > 0 {
		mtu = fmt.Sprintf("%d", iface.MTU)
	}
	kv := [][2]string{
		{"Source", "Host"},
		{"Name", iface.Name},
		{"Type", iface.Type},
		{"State", iface.State},
		{"Description", iface.Description},
		{"MTU", mtu},
		{"MAC", iface.MAC},
		{"DHCP", iface.DHCP},
		{"IPv4", strings.Join(iface.IPv4, ", ")},
		{"IPv6", strings.Join(iface.IPv6, ", ")},
		{"Gateway", strings.Join(iface.Gateway, ", ")},
		{"DNS", strings.Join(iface.DNS, ", ")},
		{"Index", fmt.Sprintf("%d", iface.IfIndex)},
	}
	rows := make([]table.Row, 0, len(kv))
	for _, pair := range kv {
		val := pair[1]
		if val == "" {
			val = "—"
		}
		rows = append(rows, table.Row{
			truncate(output.SanitizeTerminalLine(pair[0]), w0),
			truncate(output.SanitizeTerminalLine(val), w1),
		})
	}
	m.networkDetailTable.SetRows(rows)
}

func (m *MainModel) selectedNetworkEntry() (networkEntry, bool) {
	idx := m.networkTable.Cursor()
	if idx < 0 || idx >= len(m.filteredEntries) {
		return networkEntry{}, false
	}
	return m.filteredEntries[idx], true
}

func (m *MainModel) networkDetailContent() string {
	e, ok := m.selectedNetworkEntry()
	if !ok {
		return "No network selected."
	}
	label := lipgloss.NewStyle().Foreground(colorSectionLabel).Bold(true)
	muted := lipgloss.NewStyle().Foreground(colorMuted)
	var b strings.Builder
	if e.IsHost {
		h := e.Host
		fmt.Fprintf(&b, "%s\n", label.Render("Host interface"))
		fmt.Fprintf(&b, "  Name        : %s\n", h.Name)
		fmt.Fprintf(&b, "  Type        : %s\n", h.Type)
		fmt.Fprintf(&b, "  State       : %s\n", h.State)
		if h.Description != "" {
			fmt.Fprintf(&b, "  Description : %s\n", h.Description)
		}
		fmt.Fprintf(&b, "  MAC         : %s\n", h.MAC)
		if h.MTU > 0 {
			fmt.Fprintf(&b, "  MTU         : %d\n", h.MTU)
		}
		if h.DHCP != "" {
			fmt.Fprintf(&b, "  DHCP        : %s\n", h.DHCP)
		}
		fmt.Fprintf(&b, "  IPv4        : %s\n", orDash(strings.Join(h.IPv4, ", ")))
		fmt.Fprintf(&b, "  IPv6        : %s\n", orDash(strings.Join(h.IPv6, ", ")))
		fmt.Fprintf(&b, "  Gateway     : %s\n", orDash(strings.Join(h.Gateway, ", ")))
		fmt.Fprintf(&b, "  DNS         : %s\n", orDash(strings.Join(h.DNS, ", ")))
		if m.networkHostSrc != "" {
			fmt.Fprintf(&b, "\n%s\n  %s\n", label.Render("Source tool"), muted.Render(m.networkHostSrc))
		}
		return b.String()
	}
	net := e.Net
	fmt.Fprintf(&b, "%s\n", label.Render("Docker network"))
	fmt.Fprintf(&b, "  Name     : %s\n", net.Name)
	fmt.Fprintf(&b, "  ID       : %s\n", net.ID)
	fmt.Fprintf(&b, "  Driver   : %s\n", net.Driver)
	fmt.Fprintf(&b, "  Scope    : %s\n", net.Scope)
	fmt.Fprintf(&b, "  Bridge   : %s\n", net.BridgeInterface)
	fmt.Fprintf(&b, "  Subnet   : %s\n", net.Subnet)
	fmt.Fprintf(&b, "  Gateway  : %s\n", net.Gateway)
	fmt.Fprintf(&b, "\n%s\n", label.Render("Attached containers"))
	if len(net.Containers) == 0 {
		fmt.Fprintf(&b, "  %s\n", muted.Render("None"))
	} else {
		treeStyle := lipgloss.NewStyle().Foreground(colorTreeConn)
		for i, c := range net.Containers {
			prefix := "├──"
			if i == len(net.Containers)-1 {
				prefix = "└──"
			}
			st := c.Status
			if st == "" {
				st = "?"
			}
			fmt.Fprintf(&b, "  %s %s %s  %s\n",
				treeStyle.Render(prefix), st, c.Name, c.IP)
		}
	}
	if m.vethByBridge != nil {
		if veths := m.vethByBridge[net.BridgeInterface]; len(veths) > 0 {
			fmt.Fprintf(&b, "\n%s\n", label.Render("VETHs on bridge"))
			fmt.Fprintf(&b, "  %s\n", strings.Join(veths, ", "))
		}
	}
	return b.String()
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
