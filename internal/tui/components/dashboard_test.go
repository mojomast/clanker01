package components

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/internal/tui"
)

func TestNewDashboardModel(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)

	assert.NotNil(t, model)
	assert.NotNil(t, model.theme)
	assert.NotNil(t, model.agentTable)
	assert.Empty(t, model.agents)
	assert.Empty(t, model.tasks)
	assert.Empty(t, model.activity)
}

func TestDashboardModel_SetAgents(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)
	agents := []tui.Agent{
		{
			ID:             "agent-1",
			Name:           "test-agent",
			Role:           tui.RoleCoder,
			Status:         tui.StatusRunning,
			Model:          "gpt-4",
			Temperature:    0.7,
			TasksCompleted: 10,
			Uptime:         time.Hour,
			CPUUsage:       50.0,
		},
	}

	model.SetAgents(agents)

	assert.Len(t, model.agents, 1)
	assert.Equal(t, "test-agent", model.agents[0].Name)
}

func TestDashboardModel_SetTasks(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)
	tasks := []tui.Task{
		{
			ID:        "task-1",
			Name:      "test-task",
			Type:      tui.TaskTypeCode,
			Status:    tui.TaskStatusRunning,
			Priority:  tui.PriorityHigh,
			CreatedAt: time.Now(),
		},
	}

	model.SetTasks(tasks)

	assert.Len(t, model.tasks, 1)
	assert.Equal(t, "test-task", model.tasks[0].Name)
}

func TestDashboardModel_SetActivity(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)
	activity := []ActivityEntry{
		{
			Timestamp: time.Now(),
			AgentID:   "agent-1",
			Message:   "Test message",
			Type:      ActivityTaskCreated,
		},
	}

	model.SetActivity(activity)

	assert.Len(t, model.activity, 1)
	assert.Equal(t, "Test message", model.activity[0].Message)
}

func TestDashboardModel_AddActivity(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)

	for i := 0; i < 60; i++ {
		model.AddActivity(ActivityEntry{
			Timestamp: time.Now(),
			AgentID:   "agent-1",
			Message:   fmt.Sprintf("Message %d", i),
			Type:      ActivityTaskCreated,
		})
	}

	assert.Len(t, model.activity, 50)
	assert.Equal(t, "Message 59", model.activity[0].Message)
}

func TestDashboardModel_Update_WindowSize(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	updatedModel, _ := model.Update(msg)

	assert.Equal(t, 100, updatedModel.width)
	assert.Equal(t, 50, updatedModel.height)
}

func TestDashboardModel_Update_AgentList(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)
	agents := []tui.Agent{
		{
			ID:             "agent-1",
			Name:           "test-agent",
			Role:           tui.RoleCoder,
			Status:         tui.StatusRunning,
			Model:          "gpt-4",
			TasksCompleted: 10,
			Uptime:         time.Hour,
			CPUUsage:       50.0,
		},
	}
	msg := tui.AgentListMsg{Agents: agents}

	updatedModel, _ := model.Update(msg)

	assert.Len(t, updatedModel.agents, 1)
}

func TestDashboardModel_statusString(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)

	assert.Equal(t, "idle", model.statusString(tui.StatusIdle))
	assert.Equal(t, "running", model.statusString(tui.StatusRunning))
	assert.Equal(t, "paused", model.statusString(tui.StatusPaused))
	assert.Equal(t, "error", model.statusString(tui.StatusError))
	assert.Equal(t, "stopped", model.statusString(tui.StatusStopped))
}

func TestDashboardModel_View(t *testing.T) {
	model := NewDashboardModel(tui.DarkTheme)
	model.width = 100
	model.height = 50

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "AGENT OVERVIEW")
	assert.Contains(t, view, "TASK QUEUE")
	assert.Contains(t, view, "RECENT ACTIVITY")
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{time.Second, "1s"},
		{30 * time.Second, "30s"},
		{time.Minute, "1m 0s"},
		{2*time.Hour + 30*time.Minute, "2h 30m"},
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := formatDuration(tt.input)
			assert.Contains(t, result, tt.expected)
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input    string
		maxLen   int
		expected string
	}{
		{"short", 10, "short"},
		{"this is a very long string", 10, "this is..."},
		{"exact", 5, "exact"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			assert.Equal(t, tt.expected, result)
		})
	}
}
