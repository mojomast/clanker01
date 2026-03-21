package components

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/internal/tui"
)

func TestNewLogsModel(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)

	assert.NotNil(t, model)
	assert.NotNil(t, model.theme)
	assert.Empty(t, model.entries)
	assert.Equal(t, 0, model.scrollOffset)
	assert.True(t, model.follow)
}

func TestLogsModel_SetEntries(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)
	entries := []tui.LogEntry{
		{
			Timestamp: time.Now(),
			Level:     tui.LevelInfo,
			AgentID:   "agent-1",
			Message:   "Test message",
		},
	}

	model.SetEntries(entries)

	assert.Len(t, model.entries, 1)
}

func TestLogsModel_AddEntry(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)

	for i := 0; i < 150; i++ {
		model.AddEntry(tui.LogEntry{
			Timestamp: time.Now(),
			Level:     tui.LevelInfo,
			AgentID:   "agent-1",
			Message:   fmt.Sprintf("Message %d", i),
		})
	}

	assert.Len(t, model.entries, 150)
}

func TestLogsModel_filteredEntries(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)
	now := time.Now()
	entries := []tui.LogEntry{
		{
			Timestamp: now,
			Level:     tui.LevelInfo,
			AgentID:   "agent-1",
			Message:   "info message",
		},
		{
			Timestamp: now,
			Level:     tui.LevelError,
			AgentID:   "agent-1",
			Message:   "error message",
		},
		{
			Timestamp: now,
			Level:     tui.LevelInfo,
			AgentID:   "agent-2",
			Message:   "another info",
		},
	}
	model.SetEntries(entries)

	filtered := model.filteredEntries()
	assert.Len(t, filtered, 3)

	model.filter.Level = tui.LevelError
	filtered = model.filteredEntries()
	assert.Len(t, filtered, 1)

	model.filter.Level = -1
	model.filter.AgentID = "agent-1"
	filtered = model.filteredEntries()
	assert.Len(t, filtered, 2)
}

func TestLogsModel_handleKey(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)
	model.width = 100
	model.height = 50

	entries := []tui.LogEntry{}
	for i := 0; i < 100; i++ {
		entries = append(entries, tui.LogEntry{
			Timestamp: time.Now(),
			Level:     tui.LevelInfo,
			AgentID:   "agent-1",
			Message:   fmt.Sprintf("Message %d", i),
		})
	}
	model.SetEntries(entries)

	t.Run("Scroll up", func(t *testing.T) {
		model.follow = false
		model.scrollOffset = 10
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyUp})
		assert.Equal(t, 9, updatedModel.scrollOffset)
		assert.False(t, updatedModel.follow)
	})

	t.Run("Scroll down", func(t *testing.T) {
		model.scrollOffset = 5
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, 6, updatedModel.scrollOffset)
	})

	t.Run("Toggle follow", func(t *testing.T) {
		model.follow = false
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
		assert.True(t, updatedModel.follow)
	})

	t.Run("Clear", func(t *testing.T) {
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
		assert.Len(t, updatedModel.entries, 0)
	})
}

func TestLogsModel_levelString(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)

	assert.Equal(t, "DEBUG", model.levelString(tui.LevelDebug))
	assert.Equal(t, "INFO", model.levelString(tui.LevelInfo))
	assert.Equal(t, "WARN", model.levelString(tui.LevelWarn))
	assert.Equal(t, "ERROR", model.levelString(tui.LevelError))
}

func TestLogsModel_maxVisibleLines(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)
	model.height = 50

	expected := model.height - 12
	assert.Equal(t, expected, model.maxVisibleLines())
}

func TestLogsModel_View(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)
	model.width = 100
	model.height = 50

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "LOGS STREAM")
	assert.Contains(t, view, "Level:")
	assert.Contains(t, view, "Agent:")
	assert.Contains(t, view, "Search:")
}

func TestLogsModel_View_WithEntries(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)
	model.width = 100
	model.height = 50
	now := time.Now()
	entries := []tui.LogEntry{
		{
			Timestamp: now,
			Level:     tui.LevelInfo,
			AgentID:   "agent-1",
			Message:   "Test message",
		},
		{
			Timestamp: now,
			Level:     tui.LevelError,
			AgentID:   "agent-2",
			Message:   "Error message",
		},
	}
	model.SetEntries(entries)

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Test message")
	assert.Contains(t, view, "Error message")
}

func TestLogsModel_renderStatistics(t *testing.T) {
	model := NewLogsModel(tui.DarkTheme)
	now := time.Now()
	entries := []tui.LogEntry{
		{Timestamp: now, Level: tui.LevelInfo, AgentID: "agent-1", Message: "info"},
		{Timestamp: now, Level: tui.LevelDebug, AgentID: "agent-1", Message: "debug"},
		{Timestamp: now, Level: tui.LevelWarn, AgentID: "agent-1", Message: "warn"},
		{Timestamp: now, Level: tui.LevelError, AgentID: "agent-1", Message: "error"},
	}
	model.SetEntries(entries)

	stats := model.renderStatistics()

	assert.Contains(t, stats, "Statistics:")
	assert.Contains(t, stats, "4 entries")
	assert.Contains(t, stats, "1 INFO")
	assert.Contains(t, stats, "1 DEBUG")
	assert.Contains(t, stats, "1 WARN")
	assert.Contains(t, stats, "1 ERROR")
}
