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
			BorderForeground(lipgloss.Color("#585858")) // Dark Gray

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")). // White
			Background(lipgloss.Color("#7D56F4")). // Purple
			Padding(0, 1)

	tableHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#5f5fd7")). // Purple/Blue
				Bold(true).
				Border(lipgloss.NormalBorder(), false, false, true, false).
				BorderForeground(lipgloss.Color("#585858")). // Dark Gray
				Padding(0, 1)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#5f5fd7")). // Purple/Blue
			Bold(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#767676")). // Dimmed Gray
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(lipgloss.Color("#585858")). // Dark Gray
			Padding(0, 1).
			Width(100)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffffff")). // White
			Background(lipgloss.Color("#22aa22")). // Green
			Padding(0, 1).
			Bold(true)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#ffffff")). // White
				Background(lipgloss.Color("#767676")). // Dimmed Gray
				Padding(0, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f5f")). // Soft red
			Bold(true)

	actionMenuStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffdf87")). // Amber
			Bold(true)

	confirmStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ffaf5f")). // Orange-amber
			Bold(true)
)

type tab int

const (
	tabProcesses tab = iota
	tabPorts
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
	state           modelState
	table           table.Model
	input           textinput.Model
	viewport        viewport.Model
	treeViewport    viewport.Model
	envViewport     viewport.Model
	processes       []model.Process
	filtered        []model.Process
	selectedDetail  *model.Result
	detailFocus     focusState
	listFocus       focusState
	activeTab       tab
	portTable       table.Model
	portDetailTable table.Model
	portInput       textinput.Model
	ports           []model.OpenPort
	statusMsg       string // transient status/error message shown in status line
	width           int
	height          int
	quitting        bool

	selectionID int

	sortCol      string
	sortDesc     bool
	sortPortCol  string
	sortPortDesc bool
	showAllPorts bool
	version      string

	// Mouse double-click tracking
	lastClickTime time.Time
	lastClickX    int
	lastClickY    int

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

	s := table.DefaultStyles()
	s.Header = tableHeaderStyle.BorderForeground(lipgloss.Color("#585858"))
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("#ffffaf")). // Light Yellow
		Background(lipgloss.Color("#5f00d7")). // Purple
		Bold(false)
	t.SetStyles(s)

	portColumns := []table.Column{
		{Title: "Port", Width: 6},
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
		{Title: "PID", Width: 8},
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
		state:           stateList,
		table:           t,
		portTable:       pt,
		portDetailTable: pdt,
		input:           ti,
		portInput:       pi,
		viewport:        vp,
		treeViewport:    tvp,
		envViewport:     evp,
		reniceInput:     ri,
		detailFocus:     focusDetail,
		listFocus:       focusMain,
		activeTab:       tabProcesses,
		sortCol:         "mem",
		sortDesc:        true,
		sortPortCol:     "port",
		sortPortDesc:    false,
		version:         version,
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
