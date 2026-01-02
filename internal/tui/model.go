//go:build linux || darwin

package tui

import (
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/SanCognition/witr/internal/batch"
	"github.com/SanCognition/witr/pkg/model"
)

// SortField represents the field to sort by
type SortField int

const (
	SortByCPU SortField = iota
	SortByMem
	SortByAge
	SortByPID
)

func (s SortField) String() string {
	switch s {
	case SortByCPU:
		return "cpu"
	case SortByMem:
		return "mem"
	case SortByAge:
		return "age"
	case SortByPID:
		return "pid"
	default:
		return "cpu"
	}
}

// ParseSortField parses a sort field from string
func ParseSortField(s string) SortField {
	switch strings.ToLower(s) {
	case "mem", "memory":
		return SortByMem
	case "age":
		return SortByAge
	case "pid":
		return SortByPID
	default:
		return SortByCPU
	}
}

// ProcessDetails holds extended info for the details panel
type ProcessDetails struct {
	PID      int
	Ancestry []model.Process
	Ports    []int
	Err      error
}

// Model is the main TUI state
type Model struct {
	pattern       string
	processes     []batch.ProcessSummary
	filtered      []batch.ProcessSummary // Filtered view
	selected      map[int]bool           // Multi-selected PIDs
	cursorIndex   int
	details       *ProcessDetails
	detailsPID    int // PID we're fetching details for
	width, height int
	paused        bool
	refreshing    bool
	filterMode    bool
	filterText    string
	sortField     SortField
	keys          KeyMap
	lastRefresh   time.Time
	killResults   *killResultMsg
	err           error
}

// New creates a new TUI model
func New(pattern string, sortBy string) Model {
	return Model{
		pattern:   pattern,
		processes: []batch.ProcessSummary{},
		filtered:  []batch.ProcessSummary{},
		selected:  make(map[int]bool),
		sortField: ParseSortField(sortBy),
		keys:      DefaultKeyMap(),
	}
}

// Init initializes the model and starts the first refresh
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		refreshCmd(m.pattern),
		tickCmd(),
	)
}

// tickCmd returns a command that ticks after 2 seconds
func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// refreshCmd fetches process data asynchronously
func refreshCmd(pattern string) tea.Cmd {
	return func() tea.Msg {
		pids, err := batch.DiscoverPIDs(pattern)
		if err != nil {
			return processesMsg{err: err}
		}

		if len(pids) == 0 {
			return processesMsg{processes: []batch.ProcessSummary{}}
		}

		// Collect all results from async analysis
		results := batch.AnalyzeAsync(pids, 10)
		var processes []batch.ProcessSummary
		for summary := range results {
			if summary.Error == nil {
				processes = append(processes, summary)
			}
		}

		return processesMsg{processes: processes}
	}
}

// fetchDetailsCmd fetches detailed info for a process
func fetchDetailsCmd(pid int) tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement port detection
		// For now, just return ancestry
		return detailsMsg{
			pid:   pid,
			ports: []int{},
		}
	}
}

// sortProcesses sorts processes based on the current sort field
func (m *Model) sortProcesses() {
	sort.Slice(m.filtered, func(i, j int) bool {
		switch m.sortField {
		case SortByCPU:
			return m.filtered[i].CPU > m.filtered[j].CPU
		case SortByMem:
			return m.filtered[i].MemoryMB > m.filtered[j].MemoryMB
		case SortByAge:
			return m.filtered[i].StartedAt.Before(m.filtered[j].StartedAt)
		case SortByPID:
			return m.filtered[i].PID < m.filtered[j].PID
		default:
			return m.filtered[i].CPU > m.filtered[j].CPU
		}
	})
}

// applyFilter filters processes based on filter text
func (m *Model) applyFilter() {
	if m.filterText == "" {
		m.filtered = make([]batch.ProcessSummary, len(m.processes))
		copy(m.filtered, m.processes)
	} else {
		m.filtered = make([]batch.ProcessSummary, 0)
		search := strings.ToLower(m.filterText)
		for _, p := range m.processes {
			if strings.Contains(strings.ToLower(p.Command), search) ||
				strings.Contains(strings.ToLower(p.Script), search) ||
				strings.Contains(strings.ToLower(p.WorkDir), search) ||
				strings.Contains(strings.ToLower(p.Source), search) {
				m.filtered = append(m.filtered, p)
			}
		}
	}
	m.sortProcesses()

	// Ensure cursor is in bounds
	if m.cursorIndex >= len(m.filtered) {
		m.cursorIndex = max(0, len(m.filtered)-1)
	}
}

// currentProcess returns the process at the cursor
func (m *Model) currentProcess() *batch.ProcessSummary {
	if m.cursorIndex >= 0 && m.cursorIndex < len(m.filtered) {
		return &m.filtered[m.cursorIndex]
	}
	return nil
}

// selectedCount returns the number of selected processes
func (m *Model) selectedCount() int {
	count := 0
	for _, p := range m.filtered {
		if m.selected[p.PID] {
			count++
		}
	}
	return count
}
