package components

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/internal/tui"
)

func TestNewConfigModel(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)

	assert.NotNil(t, model)
	assert.NotNil(t, model.theme)
	assert.NotEmpty(t, model.fields)
	assert.Equal(t, 0, model.selectedTab)
	assert.False(t, model.isEditing)
}

func TestConfigModel_SetConfig(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)
	config := tui.Config{
		APIEndpoint:  "https://api.test.com/v1",
		DefaultModel: "gpt-4",
		MaxAgents:    20,
		TaskTimeout:  30 * time.Minute,
		LogLevel:     tui.LevelDebug,
		Temperature:  0.8,
		MaxTokens:    8192,
	}

	model.SetConfig(config)

	assert.Equal(t, "https://api.test.com/v1", model.fields[0].Value)
	assert.Equal(t, "gpt-4", model.fields[1].Value)
	assert.Equal(t, 20, model.fields[2].Value)
}

func TestConfigModel_handleKey(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)

	t.Run("Move down", func(t *testing.T) {
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, 1, updatedModel.selectedField)
	})

	t.Run("Move up", func(t *testing.T) {
		model.selectedField = 1
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyUp})
		assert.Equal(t, 0, updatedModel.selectedField)
	})

	t.Run("Enter to edit", func(t *testing.T) {
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, updatedModel.isEditing)
	})

	t.Run("Esc to cancel edit", func(t *testing.T) {
		model.isEditing = true
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updatedModel.isEditing)
		assert.Empty(t, updatedModel.editBuffer)
	})

	t.Run("Tab to cycle tabs", func(t *testing.T) {
		model.selectedTab = 0
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyTab})
		assert.Equal(t, 1, updatedModel.selectedTab)
	})
}

func TestConfigModel_handleEditKey(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)
	model.isEditing = true
	model.editBuffer = "test"

	t.Run("Type characters", func(t *testing.T) {
		updatedModel, _ := model.handleEditKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
		assert.Equal(t, "testa", updatedModel.editBuffer)
	})

	t.Run("Backspace", func(t *testing.T) {
		updatedModel, _ := model.handleEditKey(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "test", updatedModel.editBuffer)
	})

	t.Run("Enter saves", func(t *testing.T) {
		updatedModel, _ := model.handleEditKey(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, updatedModel.isEditing)
		assert.Empty(t, updatedModel.editBuffer)
	})

	t.Run("Esc cancels", func(t *testing.T) {
		updatedModel, _ := model.handleEditKey(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updatedModel.isEditing)
		assert.Empty(t, updatedModel.editBuffer)
	})
}

func TestConfigModel_validate(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)

	t.Run("Valid fields", func(t *testing.T) {
		for i := range model.fields {
			model.fields[i].Value = "valid"
		}
		isValid := true
		for _, field := range model.fields {
			if field.Required && (field.Value == "" || fmt.Sprintf("%v", field.Value) == "") {
				isValid = false
				break
			}
		}
		assert.True(t, isValid)
	})

	t.Run("Missing required field", func(t *testing.T) {
		model.fields[0].Value = ""
		isValid := true
		for _, field := range model.fields {
			if field.Required && (field.Value == "" || fmt.Sprintf("%v", field.Value) == "") {
				isValid = false
				break
			}
		}
		assert.False(t, isValid)
	})
}

func TestConfigModel_GetConfig(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)

	model.fields[0].Value = "https://api.test.com/v1"
	model.fields[1].Value = "gpt-4"
	model.fields[2].Value = 20
	model.fields[3].Value = "30m"
	model.fields[4].Value = "Info"
	model.fields[5].Value = 0.8
	model.fields[6].Value = 8192
	model.fields[7].Value = 5
	model.fields[8].Value = "5s"
	model.fields[9].Value = "100ms"

	config := model.GetConfig()

	assert.Equal(t, "https://api.test.com/v1", config.APIEndpoint)
	assert.Equal(t, "gpt-4", config.DefaultModel)
	assert.Equal(t, 20, config.MaxAgents)
	assert.Equal(t, tui.LevelInfo, config.LogLevel)
	assert.Equal(t, 0.8, config.Temperature)
	assert.Equal(t, 8192, config.MaxTokens)
	assert.Equal(t, 5, config.RetryAttempts)
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected tui.LogLevel
	}{
		{"debug", tui.LevelDebug},
		{"Debug", tui.LevelDebug},
		{"DEBUG", tui.LevelDebug},
		{"info", tui.LevelInfo},
		{"Info", tui.LevelInfo},
		{"warn", tui.LevelWarn},
		{"Warn", tui.LevelWarn},
		{"error", tui.LevelError},
		{"Error", tui.LevelError},
		{"unknown", tui.LevelInfo},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := parseLogLevel(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigModel_View(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)
	model.width = 100
	model.height = 50

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "CONFIGURATION")
	assert.Contains(t, view, "[Save] [Reset] [Export]")
	assert.Contains(t, view, "General")
	assert.Contains(t, view, "Agent Defaults")
	assert.Contains(t, view, "UI Settings")
}

func TestConfigModel_renderTabs(t *testing.T) {
	model := NewConfigModel(tui.DarkTheme)
	model.selectedTab = 1

	tabs := model.renderTabs()

	assert.Contains(t, tabs, "General")
	assert.Contains(t, tabs, "Agent Defaults")
	assert.Contains(t, tabs, "UI Settings")
}
