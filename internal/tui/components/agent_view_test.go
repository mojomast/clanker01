package components

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/internal/tui"
)

func TestNewAgentViewModel(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)

	assert.NotNil(t, model)
	assert.NotNil(t, model.theme)
	assert.Empty(t, model.recentOutput)
	assert.Nil(t, model.agent)
}

func TestAgentViewModel_SetAgent(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)
	agent := &tui.Agent{
		ID:             "agent-1",
		Name:           "test-agent",
		Role:           tui.RoleCoder,
		Status:         tui.StatusRunning,
		Model:          "gpt-4",
		Temperature:    0.7,
		TasksCompleted: 10,
		Uptime:         time.Hour,
		CPUUsage:       50.0,
	}

	model.SetAgent(agent)

	assert.NotNil(t, model.agent)
	assert.Equal(t, "test-agent", model.agent.Name)
}

func TestAgentViewModel_SetOutput(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)
	output := []OutputLine{
		{
			Timestamp: time.Now(),
			Message:   "Test output",
			Level:     OutputInfo,
		},
	}

	model.SetOutput(output)

	assert.Len(t, model.recentOutput, 1)
	assert.Equal(t, "Test output", model.recentOutput[0].Message)
}

func TestAgentViewModel_AddOutput(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)

	for i := 0; i < 150; i++ {
		model.AddOutput(OutputLine{
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Output %d", i),
			Level:     OutputInfo,
		})
	}

	assert.Len(t, model.recentOutput, 100)
}

func TestAgentViewModel_Update_WindowSize(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)
	msg := tea.WindowSizeMsg{Width: 100, Height: 50}

	updatedModel, _ := model.Update(msg)

	assert.Equal(t, 100, updatedModel.width)
	assert.Equal(t, 50, updatedModel.height)
}

func TestAgentViewModel_Update_AgentUpdate(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)
	agent := tui.Agent{
		ID:             "agent-1",
		Name:           "updated-agent",
		Role:           tui.RoleCoder,
		Status:         tui.StatusRunning,
		Model:          "gpt-4",
		TasksCompleted: 10,
		Uptime:         time.Hour,
		CPUUsage:       50.0,
	}
	msg := tui.AgentUpdateMsg{Agent: agent}

	updatedModel, _ := model.Update(msg)

	assert.NotNil(t, updatedModel.agent)
	assert.Equal(t, "updated-agent", updatedModel.agent.Name)
}

func TestAgentViewModel_statusString(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)

	assert.Equal(t, "IDLE", model.statusString(tui.StatusIdle))
	assert.Equal(t, "RUNNING", model.statusString(tui.StatusRunning))
	assert.Equal(t, "PAUSED", model.statusString(tui.StatusPaused))
	assert.Equal(t, "ERROR", model.statusString(tui.StatusError))
	assert.Equal(t, "STOPPED", model.statusString(tui.StatusStopped))
}

func TestAgentViewModel_priorityString(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)

	assert.Equal(t, "low", model.priorityString(tui.PriorityLow))
	assert.Equal(t, "medium", model.priorityString(tui.PriorityMedium))
	assert.Equal(t, "high", model.priorityString(tui.PriorityHigh))
	assert.Equal(t, "critical", model.priorityString(tui.PriorityCritical))
}

func TestAvgDurationString(t *testing.T) {
	tests := []struct {
		name     string
		tasks    int
		uptime   time.Duration
		expected string
	}{
		{"Zero tasks", 0, time.Hour, "N/A"},
		{"Seconds", 10, 30 * time.Second, "3s"},
		{"Minutes", 10, 5 * time.Minute, "30s"},
		{"Hours", 2, 2 * time.Hour, "60.0m"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := avgDurationString(tt.tasks, tt.uptime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		tokens   int64
		expected string
	}{
		{100, "100"},
		{1000, "1.0K"},
		{1500, "1.5K"},
		{1000000, "1.0M"},
		{2500000, "2.5M"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatTokens(tt.tokens)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestAgentViewModel_View_NoAgent(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)
	model.width = 100
	model.height = 50

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "No agent selected")
}

func TestAgentViewModel_View_WithAgent(t *testing.T) {
	model := NewAgentViewModel(tui.DarkTheme)
	model.width = 100
	model.height = 50
	model.agent = &tui.Agent{
		ID:             "agent-1",
		Name:           "test-agent",
		Role:           tui.RoleCoder,
		Status:         tui.StatusRunning,
		Model:          "gpt-4",
		Temperature:    0.7,
		TasksCompleted: 10,
		Uptime:         time.Hour,
		CPUUsage:       50.0,
	}

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "AGENT: test-agent")
}
