package tui

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
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

	footerStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(colorBorderDim).
			Padding(0, 1).
			Width(100)

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
	tabNetwork
)

// networkScope filters Host / Docker rows on the Network tab.
type networkScope int

const (
	networkScopeAll networkScope = iota
	networkScopeHost
	networkScopeDocker
)

// networkEntry is one row in the combined Host + Docker network table.
type networkEntry struct {
	IsHost bool
	Host   model.HostInterface
	Net    model.DockerNetwork
}

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
	input              textinput.Model
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
	portInput          textinput.Model
	ports              []model.OpenPort
	containerTable     table.Model
	containerInput     textinput.Model
	containers         []*model.ContainerMatch
	filteredContainers []*model.ContainerMatch
	selectedContainer  *model.ContainerMatch
	selectedNetwork    *model.DockerNetwork
	lockTable          table.Model
	lockInput          textinput.Model
	locks              []*model.LockedFile
	filteredLocks      []*model.LockedFile
	networkTable       table.Model
	networkDetailTable table.Model
	networkInput       textinput.Model
	networks           []model.DockerNetwork
	hostIfaces         []model.HostInterface
	networkEntries     []networkEntry // combined Host + Docker rows
	filteredEntries    []networkEntry
	vethByBridge       map[string][]string
	networkErr         string
	networkHostSrc     string // "ipconfig", "ip addr", …
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
	sortNetworkCol    string
	sortNetworkDesc   bool
	showAllPorts      bool
	showAllFiles      bool
	// networkScope filters the combined Network table: all | host | docker.
	networkScope networkScope
	showCmdCol   bool
	version      string

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

	// Ancestry navigation in the side panel
	treePIDs      []int
	treeCursor    int
	treeResult    *model.Result
	treeAncestry  []model.Process
	treeTargetPID int

	// Process action state
	actionMenuOpen bool
	pendingAction  actionKind
	reniceInput    textinput.Model
}

func InitialModel(version string) MainModel {
	columns := []table.Column{
		{Title: "PID", Width: 8},
		{Title: "User", Width: 12},
		{Title: "Name", Width: 20},
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

	networkColumns := []table.Column{
		{Title: "Source", Width: 8},
		{Title: "Name", Width: 20},
		{Title: "Kind", Width: 12},
		{Title: "State", Width: 10},
		{Title: "Address", Width: 22},
		{Title: "Extra", Width: 18},
	}
	nt := table.New(
		table.WithColumns(networkColumns),
		table.WithFocused(true),
		table.WithHeight(20),
	)
	nt.SetStyles(s)

	ndCols := []table.Column{
		{Title: "Status", Width: 8},
		{Title: "Name", Width: 18},
		{Title: "IP", Width: 20},
		{Title: "MAC", Width: 18},
	}
	ndt := table.New(
		table.WithColumns(ndCols),
		table.WithFocused(false),
		table.WithHeight(20),
	)
	ndt.SetStyles(s)

	li := textinput.New()
	li.Placeholder = "Search PID, Process, Type, Mode, Path..."
	li.CharLimit = 156
	li.Width = 50
	li.Prompt = "> "
	li.PromptStyle = promptStyle
	li.Blur()

	ni := textinput.New()
	ni.Placeholder = "Search Name, Driver, Subnet, Gateway, Bridge..."
	ni.CharLimit = 156
	ni.Width = 50
	ni.Prompt = "> "
	ni.PromptStyle = promptStyle
	ni.Blur()

	ci := textinput.New()
	ci.Placeholder = "Search ID, Name, Runtime, Image, Status, Ports, Command..."
	ci.CharLimit = 156
	ci.Width = 50
	ci.Prompt = "> "
	ci.PromptStyle = promptStyle
	ci.Blur()

	ti := textinput.New()
	ti.Placeholder = "Search PID, Name, User, Command..."
	ti.CharLimit = 156
	ti.Width = 50
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle
	ti.Blur()

	pi := textinput.New()
	pi.Placeholder = "Search Port, Protocol, Address, State..."
	pi.CharLimit = 156
	pi.Width = 50
	pi.Prompt = "> "
	pi.PromptStyle = promptStyle
	pi.Blur()

	vp := viewport.New(0, 0)
	vp.YPosition = 0

	tvp := viewport.New(0, 0)
	tvp.YPosition = 0

	evp := viewport.New(0, 0)
	evp.YPosition = 0

	ri := textinput.New()
	ri.Placeholder = "−20…19"
	ri.CharLimit = 4
	ri.Width = 8
	ri.Blur()

	return MainModel{
		state:              stateList,
		table:              t,
		portTable:          pt,
		portDetailTable:    pdt,
		containerTable:     ct,
		containerInput:     ci,
		lockTable:          lt,
		lockInput:          li,
		networkTable:       nt,
		networkDetailTable: ndt,
		networkInput:       ni,
		input:              ti,
		portInput:          pi,
		viewport:           vp,
		treeViewport:       tvp,
		envViewport:        evp,
		reniceInput:        ri,
		detailFocus:        focusDetail,
		listFocus:          focusMain,
		activeTab:          tabProcesses,
		sortCol:            "mem",
		sortDesc:           true,
		sortPortCol:        "port",
		sortPortDesc:       false,
		sortContainerCol:   "name",
		sortContainerDesc:  false,
		sortLockCol:        "pid",
		sortLockDesc:       false,
		sortNetworkCol:     "name",
		sortNetworkDesc:    false,
		// Combined Host + Docker by default.
		networkScope: networkScopeAll,
		version:      version,
		refreshEvery: refreshInterval,
	}
}

func Start(version string) error {
	if os.Getenv("COLORTERM") == "" {
		os.Setenv("COLORTERM", "truecolor") //nolint:errcheck
	}

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
		tea.EnableMouseCellMotion,
	)
}
