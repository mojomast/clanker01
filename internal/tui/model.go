package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ViewID int

const (
	ViewDashboard ViewID = iota
	ViewAgents
	ViewAgentDetail
	ViewTasks
	ViewTaskDetail
	ViewLogs
	ViewConfig
)

type FocusArea int

const (
	FocusSidebar FocusArea = iota
	FocusMain
	FocusModal
)

type Model struct {
	view         ViewID
	previousView ViewID
	width        int
	height       int

	sidebar Sidebar
	header  Header

	agents     []Agent
	tasks      []Task
	logEntries []LogEntry

	selectedAgent int
	selectedTask  int

	spinner    spinner.Model
	loading    bool
	lastUpdate time.Time

	focus  FocusArea
	theme  Theme
	keymap KeyMap

	modal Modal
}

type Sidebar struct {
	items    []SidebarItem
	selected int
	width    int
}

type SidebarItem struct {
	label  string
	icon   string
	viewID ViewID
	badge  int
}

type Header struct {
	version    string
	agentCount int
	taskCount  int
	cpuUsage   float64
	memUsage   uint64
	time       time.Time
}

type Modal struct {
	active    bool
	modalType ModalType
	formData  map[string]interface{}
	result    chan ModalResult
}

type ModalType int

const (
	ModalNone ModalType = iota
	ModalAddAgent
	ModalNewTask
	ModalConfirm
	ModalTextInput
)

type ModalResult struct {
	Confirmed bool
	Data      map[string]interface{}
}

type Agent struct {
	ID          string
	Name        string
	Role        AgentRole
	Status      AgentStatus
	Model       string
	Temperature float64
	MaxTokens   int
	Tools       []string

	CurrentTask    *Task
	TasksCompleted int
	TasksFailed    int
	Uptime         time.Duration
	CPUUsage       float64
	MemoryUsage    uint64
	TokensUsed     int64
	LastActivity   time.Time
}

type AgentRole string

const (
	RoleOrchestrator AgentRole = "orchestrator"
	RoleResearcher   AgentRole = "researcher"
	RoleCoder        AgentRole = "coder"
	RoleReviewer     AgentRole = "reviewer"
	RoleTester       AgentRole = "tester"
)

type AgentStatus int

const (
	StatusIdle AgentStatus = iota
	StatusRunning
	StatusPaused
	StatusError
	StatusStopped
)

type Task struct {
	ID          string
	Name        string
	Description string
	Type        TaskType
	Priority    Priority
	Status      TaskStatus

	AgentID     string
	CreatedAt   time.Time
	StartedAt   *time.Time
	CompletedAt *time.Time

	Dependencies []string
	Progress     float64

	Output string
	Error  string
}

type TaskType string

const (
	TaskTypeResearch TaskType = "research"
	TaskTypeCode     TaskType = "code"
	TaskTypeTest     TaskType = "test"
	TaskTypeReview   TaskType = "review"
	TaskTypeDocument TaskType = "document"
)

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

type TaskStatus int

const (
	TaskStatusPending TaskStatus = iota
	TaskStatusQueued
	TaskStatusRunning
	TaskStatusCompleted
	TaskStatusFailed
	TaskStatusCancelled
)

type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	AgentID   string
	Message   string
	Fields    map[string]interface{}
}

type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Config struct {
	APIEndpoint   string
	DefaultModel  string
	MaxAgents     int
	TaskTimeout   time.Duration
	LogLevel      LogLevel
	Temperature   float64
	MaxTokens     int
	RetryAttempts int
	RetryDelay    time.Duration
	RefreshRate   time.Duration
}

func InitialModel() Model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return Model{
		view:    ViewDashboard,
		loading: true,
		spinner: s,
		theme:   DarkTheme,
		keymap:  DefaultKeyMap(),
		sidebar: NewSidebar(),
		header:  NewHeader(),
		focus:   FocusMain,
		modal: Modal{
			active: false,
		},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
	)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		if m.modal.active {
			return m.handleModalKey(msg)
		}
		return m.handleKey(msg)

	case TickMsg:
		m.lastUpdate = time.Now()
		cmds = append(cmds, m.tick())

	case AgentListMsg:
		m.agents = msg.Agents
		m.loading = false

	case TaskListMsg:
		m.tasks = msg.Tasks

	case LogStreamMsg:
		m.logEntries = append(m.logEntries, msg.Entries...)
		if len(m.logEntries) > maxLogEntries {
			m.logEntries = m.logEntries[len(m.logEntries)-maxLogEntries:]
		}

	case LogClearedMsg:
		m.logEntries = []LogEntry{}

	case ErrorMsg:
	}

	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) tick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m Model) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	return m, nil
}

func NewSidebar() Sidebar {
	return Sidebar{
		items: []SidebarItem{
			{label: "Dashboard", icon: "▸", viewID: ViewDashboard},
			{label: "Agents", icon: "▸", viewID: ViewAgents},
			{label: "Tasks", icon: "▸", viewID: ViewTasks},
			{label: "Logs", icon: "▸", viewID: ViewLogs},
			{label: "Config", icon: "▸", viewID: ViewConfig},
		},
		selected: 0,
		width:    20,
	}
}

func NewHeader() Header {
	return Header{
		version:    "v1.0.0",
		agentCount: 0,
		taskCount:  0,
		cpuUsage:   0,
		memUsage:   0,
		time:       time.Now(),
	}
}

const maxLogEntries = 1000
