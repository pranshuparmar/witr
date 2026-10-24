package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(colorBorderDim)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorBrandFg).
			Background(colorBrandBg).
			Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(colorBorderDim).
				Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	placeholderStyle = lipgloss.NewStyle().Foreground(colorMuted).Faint(basicColors)

	// The layout reserves one line for the footer: text that doesn't fit is
	// cut rather than wrapped onto a second line.
	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Faint(basicColors).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(colorBorderDim).
			Padding(0, 1).
			Width(100).
			MaxHeight(2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(colorOnAccent).
			Background(colorGreenBg).
			Padding(0, 1).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(colorOnAccent).
				Background(colorIdleTabBg).
				Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorError).
			Bold(true)

	actionMenuStyle = lipgloss.NewStyle().
			Foreground(colorAmber).
			Bold(true)

	confirmStyle = lipgloss.NewStyle().
			Foreground(colorConfirm).
			Bold(true)

	// Matches the selected table row: it names the process an action targets.
	actionTargetStyle = lipgloss.NewStyle().
				Foreground(colorSelectFg).
				Background(colorSelectBg).
				Reverse(basicColors).
				Padding(0, 1)

	pidStyle = lipgloss.NewStyle().
			Background(colorGreenBg).
			Foreground(colorOnAccent).
			Padding(0, 1).
			Bold(true)

	spacerStyle    = lipgloss.NewStyle().Height(1)
	paddedStyle    = lipgloss.NewStyle().PaddingLeft(1)
	statusBarStyle = lipgloss.NewStyle().MarginBottom(1).PaddingLeft(1)

	paneDividerStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				PaddingLeft(2)

	detailDividerStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				PaddingLeft(1)

	envPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), false, false, false, true).
			PaddingLeft(1)

	// Cached table styles with customized selection colors
	cachedTableStyles = func() table.Styles {
		s := table.DefaultStyles()
		s.Selected = s.Selected.
			Foreground(colorSelectFg).
			Background(colorSelectBg).
			Reverse(basicColors).
			Bold(false)
		return s
	}()
)

type tab int

const (
	tabProcesses tab = iota
	tabPorts
	tabContainers
	tabLocks
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

type actionKind int

const (
	actionNone   actionKind = iota
	actionKill              // SIGKILL
	actionTerm              // SIGTERM
	actionPause             // SIGSTOP
	actionResume            // SIGCONT
	actionRenice            // setpriority
)

type MainModel struct {
	state              modelState
	table              table.Model
	input              textInput
	viewport           viewport.Model
	treeViewport       viewport.Model
	envViewport        viewport.Model
	processes          []model.Process
	filtered           []model.Process
	selectedDetail     *model.Result
	detailFocus        focusState
	listFocus          focusState
	activeTab          tab
	portTable          table.Model
	portDetailTable    table.Model
	portInput          textInput
	ports              []model.OpenPort
	containerTable     table.Model
	containerInput     textInput
	containers         []*model.ContainerMatch
	filteredContainers []*model.ContainerMatch
	selectedContainer  *model.ContainerMatch
	lockTable          table.Model
	lockInput          textInput
	locks              []*model.LockedFile
	filteredLocks      []*model.LockedFile
	statusMsg          string // transient status/error message shown in status line
	width              int
	height             int
	quitting           bool

	selectionID int

	sortCol           string
	sortDesc          bool
	sortPortCol       string
	sortPortDesc      bool
	sortContainerCol  string
	sortContainerDesc bool
	sortLockCol       string
	sortLockDesc      bool
	showAllPorts      bool
	showAllFiles      bool
	showCmdCol        bool
	version           string

	// Mouse double-click tracking
	lastClickTime time.Time
	lastClickX    int
	lastClickY    int

	// Adaptive auto-refresh: refreshEvery is the current cadence (it adapts to
	// how long refreshes take); lastRefresh gates the next one; refreshStartedAt
	// marks an in-flight refresh so its duration can be measured and two don't
	// overlap.
	refreshEvery     time.Duration
	lastRefresh      time.Time
	refreshStartedAt time.Time
	slowStreak       int
	fastStreak       int

	// Ancestry navigation in the side panel. treeRows maps each tree line
	// below the label to its treePIDs index, or -1 for a line that isn't a
	// process.
	treePIDs      []int
	treeRows      []int
	treeCursor    int
	treeResult    *model.Result
	treeAncestry  []model.Process
	treeTargetPID int

	// Process action state. actionTarget is captured when the menu opens, so
	// the action hits that process even if the list re-sorts underneath.
	actionMenuOpen bool
	pendingAction  actionKind
	actionTarget   *model.Process
	reniceInput    textInput

	// PID to select once the first process list arrives
	initialPID int

	// exactName and exactPort make filters seeded from CLI targets match
	// exactly (names with -x, ports always) until the user edits them.
	exactName bool
	exactPort bool

	// Replay mode state
	snapshots     []*model.Snapshot
	snapshotIndex int
	isReplay      bool
}

func InitialModel(version string) MainModel {
	columns := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "User", Width: 12},
		{Title: "Name", Width: 20},
		{Title: "Avg CPU", Width: 9},
		{Title: "Mem", Width: 16},
		{Title: "Started", Width: 19},
		{Title: "Command", Width: 50},
	}

	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(20),
	)

	s := cachedTableStyles
	s.Header = tableHeaderStyle.BorderForeground(colorBorderDim)
	t.SetStyles(s)

	portColumns := []table.Column{
		{Title: centerHeader("Port", 6), Width: 6},
		{Title: "Protocol", Width: 10},
		{Title: "Address", Width: 30},
		{Title: "State", Width: 20},
	}
	pt := table.New(
		table.WithColumns(portColumns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	pt.SetStyles(s)

	pdCols := []table.Column{
		{Title: centerHeader("PID", 8), Width: 8},
		{Title: "User", Width: 12},
		{Title: "Name", Width: 15},
		{Title: "Command", Width: 20},
	}
	pdt := table.New(
		table.WithColumns(pdCols),
		table.WithFocused(false),
		table.WithHeight(20),
	)
	pdt.SetStyles(s)

	containerColumns := []table.Column{
		{Title: "ID", Width: 14},
		{Title: "Name", Width: 22},
		{Title: "Runtime", Width: 10},
		{Title: "Image", Width: 28},
		{Title: "Status", Width: 22},
		{Title: "Ports", Width: 24},
		{Title: "Command", Width: 28},
	}
	ct := table.New(
		table.WithColumns(containerColumns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	ct.SetStyles(s)

	lockColumns := []table.Column{
		{Title: centerHeader("PID", 8), Width: 8},
		{Title: "Process", Width: 18},
		{Title: "Type", Width: 8},
		{Title: "Mode", Width: 8},
		{Title: "Path", Width: 50},
	}
	lt := table.New(
		table.WithColumns(lockColumns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	lt.SetStyles(s)

	li := newTextInput("Search PID, Process, Type, Mode, Path...", 156, 50)
	ci := newTextInput("Search ID, Name, Runtime, Image, Status, Ports, Command...", 156, 50)
	ti := newTextInput("Search PID, Name, User, Command...", 156, 50)
	pi := newTextInput("Search Port, Protocol, Address, State...", 156, 50)

	vp := viewport.New(0, 0)
	vp.YPosition = 0

	tvp := viewport.New(0, 0)
	tvp.YPosition = 0

	evp := viewport.New(0, 0)
	evp.YPosition = 0

	// The renice prompt follows the confirmation text, so its "> " is plain.
	ri := newTextInput("−20…19", 4, 8)
	ri.PromptStyle = lipgloss.NewStyle()

	return MainModel{
		state:             stateList,
		table:             t,
		portTable:         pt,
		portDetailTable:   pdt,
		containerTable:    ct,
		containerInput:    ci,
		lockTable:         lt,
		lockInput:         li,
		input:             ti,
		portInput:         pi,
		viewport:          vp,
		treeViewport:      tvp,
		envViewport:       evp,
		reniceInput:       ri,
		detailFocus:       focusDetail,
		listFocus:         focusMain,
		activeTab:         tabProcesses,
		sortCol:           "mem",
		sortDesc:          true,
		sortPortCol:       "port",
		sortPortDesc:      false,
		sortContainerCol:  "name",
		sortContainerDesc: false,
		sortLockCol:       "pid",
		sortLockDesc:      false,
		version:           version,
		refreshEvery:      refreshInterval,
	}
}

func Start(version string, targets []model.Target, exact bool) error {
	lipgloss.SetColorProfile(lipglossProfile(colorProfile))

	p := tea.NewProgram(InitialModel(version).withTargets(targets, exact), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running tui: %w", err)
	}
	return nil
}

// StartReplay launches the interactive TUI browsing recorded snapshots.
func StartReplay(version string, snapshots []*model.Snapshot, targets []model.Target, exact bool) error {
	lipgloss.SetColorProfile(lipglossProfile(colorProfile))

	m := InitialModel(version)
	m.snapshots = snapshots
	m.isReplay = true
	if len(snapshots) > 0 {
		m.snapshotIndex = len(snapshots) - 1
	}
	p := tea.NewProgram(m.withTargets(targets, exact), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("error running tui: %w", err)
	}
	return nil
}

// withTargets seeds the initial tab, filter and selection from the CLI
// targets so `witr -i` opens where a non-interactive run would have looked.
// The TUI shows one target of each type; any others are listed in the status
// line rather than dropped silently.
func (m MainModel) withTargets(targets []model.Target, exact bool) MainModel {
	var skipped []string
	for _, t := range targets {
		used := false
		switch t.Type {
		case model.TargetPID:
			if pid, err := strconv.Atoi(t.Value); err == nil && m.initialPID == 0 {
				m.initialPID = pid
				used = true
			}
		case model.TargetName:
			if m.input.Value() == "" {
				m.input.SetValue(t.Value)
				m.exactName = exact
				used = true
			}
		case model.TargetPort:
			if m.portInput.Value() == "" {
				m.portInput.SetValue(t.Value)
				m.exactPort = true
				m.activeTab = tabPorts
				used = true
			}
		case model.TargetContainer:
			if m.containerInput.Value() == "" {
				m.containerInput.SetValue(t.Value)
				m.activeTab = tabContainers
				used = true
			}
		case model.TargetFile:
			if locksTabEnabled && m.lockInput.Value() == "" {
				m.lockInput.SetValue(t.Value)
				m.activeTab = tabLocks
				used = true
			}
		}
		if !used {
			skipped = append(skipped, string(t.Type)+" "+t.Value)
		}
	}
	if len(skipped) > 0 {
		m.statusMsg = "Interactive mode shows one target of each type; not shown: " + strings.Join(skipped, ", ")
	}
	return m
}

func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.refreshProcesses(),
		tea.EnableMouseCellMotion,
	}
	if !m.isReplay {
		cmds = append(cmds, waitTick())
	}
	switch m.activeTab {
	case tabPorts:
		cmds = append(cmds, m.refreshPorts())
	case tabContainers:
		cmds = append(cmds, m.refreshContainers())
	case tabLocks:
		cmds = append(cmds, m.refreshLocks())
	}
	return tea.Batch(cmds...)
}
