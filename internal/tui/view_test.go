package tui

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestViewLoadingState(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = true

	view := model.View()

	assert.Contains(t, view, "Loading...")
}

func TestViewZeroDimensions(t *testing.T) {
	model := InitialModel()
	model.width = 0
	model.height = 0

	view := model.View()

	assert.Contains(t, view, "Loading...")
}

func TestViewDashboard(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewDashboard

	view := model.View()

	assert.Contains(t, view, "SWARM")
	assert.Contains(t, view, "Dashboard View")
	assert.Contains(t, view, "v1.0.0")
	assert.Contains(t, view, "[a] Add Agent")
	assert.Contains(t, view, "[t] New Task")
	assert.Contains(t, view, "[l] Logs")
	assert.Contains(t, view, "[r] Refresh")
	assert.Contains(t, view, "[?] Help")
	assert.Contains(t, view, "[q] Quit")
}

func TestViewAgents(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewAgents

	view := model.View()

	assert.Contains(t, view, "Agents View")
	assert.Contains(t, view, "[Enter] Details")
	assert.Contains(t, view, "[a] Add")
	assert.Contains(t, view, "[d] Delete")
}

func TestViewAgentDetail(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewAgentDetail

	view := model.View()

	assert.Contains(t, view, "Agent Detail View")
}

func TestViewTasks(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewTasks

	view := model.View()

	assert.Contains(t, view, "Tasks View")
	assert.Contains(t, view, "[Enter] View")
	assert.Contains(t, view, "[d] Delete")
	assert.Contains(t, view, "[↑↓] Navigate")
	assert.Contains(t, view, "[r] Retry")
	assert.Contains(t, view, "[c] Cancel")
}

func TestViewTaskDetail(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewTaskDetail

	view := model.View()

	assert.Contains(t, view, "Task Detail View")
}

func TestViewLogsWithEntries(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewLogs
	model.logEntries = []LogEntry{
		{
			Timestamp: time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC),
			Level:     LevelInfo,
			AgentID:   "agent-1",
			Message:   "Test log message 1",
		},
		{
			Timestamp: time.Date(2024, 1, 15, 14, 31, 0, 0, time.UTC),
			Level:     LevelWarn,
			AgentID:   "agent-2",
			Message:   "Test log message 2",
		},
		{
			Timestamp: time.Date(2024, 1, 15, 14, 32, 0, 0, time.UTC),
			Level:     LevelError,
			AgentID:   "agent-3",
			Message:   "Test log message 3",
		},
	}

	view := model.View()

	assert.Contains(t, view, "INFO")
	assert.Contains(t, view, "WARN")
	assert.Contains(t, view, "ERROR")
	assert.Contains(t, view, "agent-1")
	assert.Contains(t, view, "agent-2")
	assert.Contains(t, view, "agent-3")
	assert.Contains(t, view, "Test log message 1")
	assert.Contains(t, view, "Test log message 2")
	assert.Contains(t, view, "Test log message 3")
	assert.Contains(t, view, "[f] Follow")
	assert.Contains(t, view, "[↑↓] Scroll")
	assert.Contains(t, view, "[/] Search")
	assert.Contains(t, view, "[c] Clear")
}

func TestViewLogsWithoutEntries(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewLogs
	model.logEntries = []LogEntry{}

	view := model.View()

	assert.Contains(t, view, "No log entries available.")
}

func TestViewConfig(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewConfig

	view := model.View()

	assert.Contains(t, view, "Config View")
	assert.Contains(t, view, "[Tab] Next Field")
	assert.Contains(t, view, "[Enter] Edit")
	assert.Contains(t, view, "[Esc] Cancel")
	assert.Contains(t, view, "[Ctrl+S] Save")
}

func TestViewSidebar(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false

	view := model.View()

	assert.Contains(t, view, "▸ Dashboard")
	assert.Contains(t, view, "▸ Agents")
	assert.Contains(t, view, "▸ Tasks")
	assert.Contains(t, view, "▸ Logs")
	assert.Contains(t, view, "▸ Config")
}

func TestViewSidebarSelected(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.sidebar.selected = 1

	view := model.View()

	assert.Contains(t, view, "▸ Dashboard")
	assert.Contains(t, view, "▸ Agents")
}

func TestViewHeader(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false

	view := model.View()

	assert.Contains(t, view, "SWARM")
	assert.Contains(t, view, "v1.0.0")
	assert.Contains(t, view, "Agents:")
	assert.Contains(t, view, "Tasks:")
	assert.Contains(t, view, "CPU:")
	assert.Contains(t, view, "MEM:")
}

func TestViewModalAddAgent(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.modal = Modal{
		active:    true,
		modalType: ModalAddAgent,
	}

	view := model.View()

	assert.Contains(t, view, "ADD NEW AGENT")
	assert.Contains(t, view, "Name:")
	assert.Contains(t, view, "Role:")
	assert.Contains(t, view, "Model:")
	assert.Contains(t, view, "Tools:")
	assert.Contains(t, view, "[Cancel]")
	assert.Contains(t, view, "[Add Agent]")
}

func TestViewModalNewTask(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.modal = Modal{
		active:    true,
		modalType: ModalNewTask,
	}

	view := model.View()

	assert.Contains(t, view, "NEW TASK")
	assert.Contains(t, view, "Task Type:")
	assert.Contains(t, view, "Priority:")
	assert.Contains(t, view, "Assign To:")
	assert.Contains(t, view, "Description:")
	assert.Contains(t, view, "[Cancel]")
	assert.Contains(t, view, "[Create Task]")
}

func TestViewModalConfirm(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.modal = Modal{
		active:    true,
		modalType: ModalConfirm,
	}

	view := model.View()

	assert.Contains(t, view, "CONFIRM")
	assert.Contains(t, view, "Are you sure you want to proceed?")
	assert.Contains(t, view, "[Cancel]")
	assert.Contains(t, view, "[Confirm]")
}

func TestViewModalTextInput(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.modal = Modal{
		active:    true,
		modalType: ModalTextInput,
	}

	view := model.View()

	assert.Contains(t, view, "INPUT")
	assert.Contains(t, view, "Enter your input below:")
	assert.Contains(t, view, "[Cancel]")
	assert.Contains(t, view, "[OK]")
}

func TestLevelString(t *testing.T) {
	model := InitialModel()

	assert.Equal(t, "DEBUG", model.levelString(LevelDebug))
	assert.Equal(t, "INFO", model.levelString(LevelInfo))
	assert.Equal(t, "WARN", model.levelString(LevelWarn))
	assert.Equal(t, "ERROR", model.levelString(LevelError))
	assert.Equal(t, "UNKNOWN", model.levelString(LogLevel(99)))
}

func TestFormatMemoryBytes(t *testing.T) {
	result := formatMemory(500)
	assert.Equal(t, "500B", result)
}

func TestFormatMemoryKilobytes(t *testing.T) {
	result := formatMemory(1536)
	assert.Contains(t, result, "KB")
}

func TestFormatMemoryMegabytes(t *testing.T) {
	result := formatMemory(1572864)
	assert.Contains(t, result, "MB")
}

func TestFormatMemoryGigabytes(t *testing.T) {
	result := formatMemory(1610612736)
	assert.Contains(t, result, "GB")
}

func TestViewWithAgents(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.agents = []Agent{
		{
			ID:     "agent-1",
			Name:   "Test Agent",
			Status: StatusRunning,
		},
	}

	view := model.View()

	assert.Contains(t, view, "SWARM")
	assert.Contains(t, view, "Dashboard View")
}

func TestViewWithTasks(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.tasks = []Task{
		{
			ID:     "task-1",
			Name:   "Test Task",
			Status: TaskStatusRunning,
		},
	}

	view := model.View()

	assert.Contains(t, view, "SWARM")
	assert.Contains(t, view, "Dashboard View")
}

func TestViewFooterDifferentViews(t *testing.T) {
	views := []struct {
		viewID   ViewID
		expected []string
	}{
		{ViewDashboard, []string{"[a] Add Agent", "[t] New Task"}},
		{ViewAgents, []string{"[Enter] Details", "[a] Add"}},
		{ViewTasks, []string{"[Enter] View", "[d] Delete"}},
		{ViewLogs, []string{"[f] Follow", "[↑↓] Scroll"}},
		{ViewConfig, []string{"[Tab] Next Field", "[Enter] Edit"}},
	}

	for _, tt := range views {
		model := InitialModel()
		model.width = 100
		model.height = 50
		model.loading = false
		model.view = tt.viewID

		view := model.View()

		for _, expected := range tt.expected {
			assert.Contains(t, view, expected)
		}
	}
}

func TestRenderFooter(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false

	footer := model.renderFooter()

	assert.NotEmpty(t, footer)
	assert.Contains(t, footer, "[a] Add Agent")
	assert.Contains(t, footer, "[t] New Task")
	assert.Contains(t, footer, "[?] Help")
	assert.Contains(t, footer, "[q] Quit")
}

func TestRenderHeader(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.header.agentCount = 5
	model.header.taskCount = 23
	model.header.cpuUsage = 45.0
	model.header.memUsage = 2100000000
	model.header.time = time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	header := model.renderHeader()

	assert.Contains(t, header, "SWARM v1.0.0")
	assert.Contains(t, header, "Agents: 5")
	assert.Contains(t, header, "Tasks: 23")
	assert.Contains(t, header, "CPU: 45%")
	assert.Contains(t, header, "MEM:")
	assert.Contains(t, header, "14:30")
}

func TestRenderSidebar(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false

	sidebar := model.renderSidebar()

	assert.Contains(t, sidebar, "▸ Dashboard")
	assert.Contains(t, sidebar, "▸ Agents")
	assert.Contains(t, sidebar, "▸ Tasks")
	assert.Contains(t, sidebar, "▸ Logs")
	assert.Contains(t, sidebar, "▸ Config")
}

func TestRenderMainDashboard(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewDashboard

	main := model.renderMain()

	assert.Contains(t, main, "Dashboard View")
}

func TestRenderMainAgents(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewAgents

	main := model.renderMain()

	assert.Contains(t, main, "Agents View")
}
