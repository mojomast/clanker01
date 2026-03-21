package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestInitialModel(t *testing.T) {
	model := InitialModel()

	assert.Equal(t, ViewDashboard, model.view)
	assert.Equal(t, true, model.loading)
	assert.Equal(t, FocusMain, model.focus)
	assert.Equal(t, false, model.modal.active)
	assert.Equal(t, 0, model.width)
	assert.Equal(t, 0, model.height)
	assert.NotNil(t, model.spinner)
	assert.NotNil(t, model.theme)
	assert.NotNil(t, model.keymap)
	assert.NotNil(t, model.sidebar)
	assert.NotNil(t, model.header)
}

func TestModelInit(t *testing.T) {
	model := InitialModel()
	cmd := model.Init()

	assert.NotNil(t, cmd)
}

func TestModelUpdateWindowSize(t *testing.T) {
	model := InitialModel()

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	newModel, _ := model.Update(msg)

	assert.Equal(t, 100, newModel.(Model).width)
	assert.Equal(t, 50, newModel.(Model).height)
}

func TestModelUpdateTick(t *testing.T) {
	model := InitialModel()
	model.loading = false

	msg := TickMsg(time.Now())
	newModel, cmd := model.Update(msg)

	assert.False(t, newModel.(Model).loading)
	assert.NotNil(t, cmd)
}

func TestModelUpdateAgentList(t *testing.T) {
	model := InitialModel()
	model.loading = true

	agents := []Agent{
		{ID: "agent-1", Name: "Agent 1", Status: StatusRunning},
		{ID: "agent-2", Name: "Agent 2", Status: StatusIdle},
	}

	msg := AgentListMsg{Agents: agents, Err: nil}
	newModel, _ := model.Update(msg)

	assert.False(t, newModel.(Model).loading)
	assert.Equal(t, 2, len(newModel.(Model).agents))
	assert.Equal(t, "agent-1", newModel.(Model).agents[0].ID)
}

func TestModelUpdateTaskList(t *testing.T) {
	model := InitialModel()

	tasks := []Task{
		{ID: "task-1", Name: "Task 1", Status: TaskStatusRunning},
		{ID: "task-2", Name: "Task 2", Status: TaskStatusPending},
	}

	msg := TaskListMsg{Tasks: tasks, Err: nil}
	newModel, _ := model.Update(msg)

	assert.Equal(t, 2, len(newModel.(Model).tasks))
	assert.Equal(t, "task-1", newModel.(Model).tasks[0].ID)
}

func TestModelUpdateLogStream(t *testing.T) {
	model := InitialModel()

	entries := []LogEntry{
		{
			Timestamp: time.Now(),
			Level:     LevelInfo,
			AgentID:   "agent-1",
			Message:   "Test message",
		},
	}

	msg := LogStreamMsg{Entries: entries}
	newModel, _ := model.Update(msg)

	assert.Equal(t, 1, len(newModel.(Model).logEntries))
	assert.Equal(t, "Test message", newModel.(Model).logEntries[0].Message)
}

func TestModelUpdateLogCleared(t *testing.T) {
	model := InitialModel()
	model.logEntries = []LogEntry{
		{Timestamp: time.Now(), Level: LevelInfo, Message: "Old log"},
	}

	msg := LogClearedMsg{}
	newModel, _ := model.Update(msg)

	assert.Equal(t, 0, len(newModel.(Model).logEntries))
}

func TestModelUpdateLogStreamTruncation(t *testing.T) {
	model := InitialModel()

	entries := make([]LogEntry, maxLogEntries+100)
	for i := range entries {
		entries[i] = LogEntry{
			Timestamp: time.Now(),
			Level:     LevelInfo,
			Message:   "Test message",
		}
	}

	msg := LogStreamMsg{Entries: entries}
	newModel, _ := model.Update(msg)

	assert.Equal(t, maxLogEntries, len(newModel.(Model).logEntries))
}

func TestModelUpdateKeyMessages(t *testing.T) {
	model := InitialModel()
	model.loading = false
	model.width = 100
	model.height = 50

	keys := []string{"q", "?", "1", "2", "3", "4", "5"}
	for _, key := range keys {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{rune(key[0])}}
		newModel, _ := model.Update(msg)
		assert.NotNil(t, newModel)
	}
}

func TestModelViewLoading(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = true

	view := model.View()
	assert.Contains(t, view, "Loading...")
}

func TestModelViewZeroWidth(t *testing.T) {
	model := InitialModel()
	model.width = 0

	view := model.View()
	assert.Contains(t, view, "Loading...")
}

func TestModelViewDashboard(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewDashboard

	view := model.View()
	assert.Contains(t, view, "DASHBOARD")
	assert.Contains(t, view, "SWARM")
}

func TestModelViewAgents(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewAgents

	view := model.View()
	assert.Contains(t, view, "AGENTS")
}

func TestModelViewTasks(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewTasks

	view := model.View()
	assert.Contains(t, view, "TASK QUEUE")
}

func TestModelViewLogs(t *testing.T) {
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
			Message:   "Test log entry",
		},
	}

	view := model.View()

	assert.Contains(t, view, "INFO")
	assert.Contains(t, view, "Test log entry")
}

func TestModelViewConfig(t *testing.T) {
	model := InitialModel()
	model.width = 100
	model.height = 50
	model.loading = false
	model.view = ViewConfig

	view := model.View()
	assert.Contains(t, view, "CONFIGURATION")
}

func TestModelViewModal(t *testing.T) {
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
}

func TestSidebarCreation(t *testing.T) {
	sidebar := NewSidebar()

	assert.Equal(t, 5, len(sidebar.items))
	assert.Equal(t, "Dashboard", sidebar.items[0].label)
	assert.Equal(t, "▸", sidebar.items[0].icon)
	assert.Equal(t, ViewDashboard, sidebar.items[0].viewID)
	assert.Equal(t, 0, sidebar.selected)
	assert.Equal(t, 20, sidebar.width)
}

func TestHeaderCreation(t *testing.T) {
	header := NewHeader()

	assert.Equal(t, "v1.0.0", header.version)
	assert.Equal(t, 0, header.agentCount)
	assert.Equal(t, 0, header.taskCount)
	assert.Equal(t, 0.0, header.cpuUsage)
	assert.Equal(t, uint64(0), header.memUsage)
	assert.False(t, header.time.IsZero())
}

func TestFormatMemory(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{500, "500B"},
		{1024, "1.0KB"},
		{1024 * 1024, "1.0MB"},
		{1024 * 1024 * 1024, "1.0GB"},
		{1536 * 1024 * 1024, "1.5GB"},
	}

	for _, tt := range tests {
		result := formatMemory(tt.bytes)
		assert.Contains(t, result, tt.expected)
	}
}

func TestModelTick(t *testing.T) {
	model := InitialModel()
	cmd := model.tick()

	assert.NotNil(t, cmd)
}

func TestAgentStatusValues(t *testing.T) {
	values := []AgentStatus{
		StatusIdle,
		StatusRunning,
		StatusPaused,
		StatusError,
		StatusStopped,
	}

	for _, status := range values {
		assert.True(t, status >= StatusIdle && status <= StatusStopped)
	}
}

func TestTaskStatusValues(t *testing.T) {
	values := []TaskStatus{
		TaskStatusPending,
		TaskStatusQueued,
		TaskStatusRunning,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCancelled,
	}

	for _, status := range values {
		assert.True(t, status >= TaskStatusPending && status <= TaskStatusCancelled)
	}
}

func TestLogLevelValues(t *testing.T) {
	values := []LogLevel{
		LevelDebug,
		LevelInfo,
		LevelWarn,
		LevelError,
	}

	for _, level := range values {
		assert.True(t, level >= LevelDebug && level <= LevelError)
	}
}

func TestPriorityValues(t *testing.T) {
	values := []Priority{
		PriorityLow,
		PriorityMedium,
		PriorityHigh,
		PriorityCritical,
	}

	for _, priority := range values {
		assert.True(t, priority >= PriorityLow && priority <= PriorityCritical)
	}
}

func TestViewIDValues(t *testing.T) {
	values := []ViewID{
		ViewDashboard,
		ViewAgents,
		ViewAgentDetail,
		ViewTasks,
		ViewTaskDetail,
		ViewLogs,
		ViewConfig,
	}

	for _, view := range values {
		assert.True(t, view >= ViewDashboard && view <= ViewConfig)
	}
}

func TestModalTypeValues(t *testing.T) {
	values := []ModalType{
		ModalNone,
		ModalAddAgent,
		ModalNewTask,
		ModalConfirm,
		ModalTextInput,
	}

	for _, modalType := range values {
		assert.True(t, modalType >= ModalNone && modalType <= ModalTextInput)
	}
}
