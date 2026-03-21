package components

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/internal/tui"
)

func TestNewTaskQueueModel(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)

	assert.NotNil(t, model)
	assert.NotNil(t, model.theme)
	assert.Empty(t, model.tasks)
	assert.Equal(t, 0, model.cursor)
	assert.False(t, model.showDetails)
}

func TestTaskQueueModel_SetTasks(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)
	tasks := []tui.Task{
		{
			ID:        "task-1",
			Name:      "test-task",
			Type:      tui.TaskTypeCode,
			Status:    tui.TaskStatusPending,
			Priority:  tui.PriorityHigh,
			CreatedAt: time.Now(),
		},
	}

	model.SetTasks(tasks)

	assert.Len(t, model.tasks, 1)
	assert.Equal(t, "test-task", model.tasks[0].Name)
}

func TestTaskQueueModel_SetFilter(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)
	filter := TaskFilter{
		Status:   tui.TaskStatusRunning,
		Priority: tui.PriorityHigh,
	}

	model.SetFilter(filter)

	assert.Equal(t, tui.TaskStatusRunning, model.filter.Status)
	assert.Equal(t, tui.PriorityHigh, model.filter.Priority)
}

func TestTaskQueueModel_filteredTasks(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)
	now := time.Now()
	tasks := []tui.Task{
		{
			ID:        "task-1",
			Name:      "running-task",
			Type:      tui.TaskTypeCode,
			Status:    tui.TaskStatusRunning,
			Priority:  tui.PriorityHigh,
			CreatedAt: now,
		},
		{
			ID:        "task-2",
			Name:      "pending-task",
			Type:      tui.TaskTypeCode,
			Status:    tui.TaskStatusPending,
			Priority:  tui.PriorityLow,
			CreatedAt: now,
		},
	}
	model.SetTasks(tasks)

	filtered := model.filteredTasks()
	assert.Len(t, filtered, 2)

	model.filter.Status = tui.TaskStatusRunning
	filtered = model.filteredTasks()
	assert.Len(t, filtered, 1)
	assert.Equal(t, "running-task", filtered[0].Name)
}

func TestTaskQueueModel_handleKey(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)
	now := time.Now()
	tasks := []tui.Task{
		{
			ID:        "task-1",
			Name:      "task-1",
			Type:      tui.TaskTypeCode,
			Status:    tui.TaskStatusPending,
			Priority:  tui.PriorityMedium,
			CreatedAt: now,
		},
		{
			ID:        "task-2",
			Name:      "task-2",
			Type:      tui.TaskTypeCode,
			Status:    tui.TaskStatusPending,
			Priority:  tui.PriorityMedium,
			CreatedAt: now,
		},
		{
			ID:        "task-3",
			Name:      "task-3",
			Type:      tui.TaskTypeCode,
			Status:    tui.TaskStatusPending,
			Priority:  tui.PriorityMedium,
			CreatedAt: now,
		},
	}
	model.SetTasks(tasks)

	t.Run("Move down", func(t *testing.T) {
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyDown})
		assert.Equal(t, 1, updatedModel.cursor)
	})

	t.Run("Move up", func(t *testing.T) {
		model.cursor = 1
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyUp})
		assert.Equal(t, 0, updatedModel.cursor)
	})

	t.Run("Enter shows details", func(t *testing.T) {
		updatedModel, _ := model.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, updatedModel.showDetails)
		assert.NotNil(t, updatedModel.taskDetail)
	})

	t.Run("Esc closes details", func(t *testing.T) {
		model.showDetails = true
		updatedModel, _ := model.handleDetailKey(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updatedModel.showDetails)
		assert.Nil(t, updatedModel.taskDetail)
	})
}

func TestTaskQueueModel_statusString(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)

	assert.Equal(t, "pending", model.statusString(tui.TaskStatusPending))
	assert.Equal(t, "queued", model.statusString(tui.TaskStatusQueued))
	assert.Equal(t, "running", model.statusString(tui.TaskStatusRunning))
	assert.Equal(t, "done", model.statusString(tui.TaskStatusCompleted))
	assert.Equal(t, "failed", model.statusString(tui.TaskStatusFailed))
	assert.Equal(t, "cancelled", model.statusString(tui.TaskStatusCancelled))
}

func TestTaskQueueModel_priorityString(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)

	assert.Equal(t, "low", model.priorityString(tui.PriorityLow))
	assert.Equal(t, "medium", model.priorityString(tui.PriorityMedium))
	assert.Equal(t, "high", model.priorityString(tui.PriorityHigh))
	assert.Equal(t, "critical", model.priorityString(tui.PriorityCritical))
}

func TestFormatDurationShort(t *testing.T) {
	tests := []struct {
		input    time.Duration
		expected string
	}{
		{time.Second, "1s"},
		{30 * time.Second, "30s"},
		{time.Minute, "1m"},
		{2*time.Hour + 30*time.Minute, "2h"},
	}

	for _, tt := range tests {
		t.Run(tt.input.String(), func(t *testing.T) {
			result := formatDurationShort(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTaskQueueModel_View(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)
	model.width = 100
	model.height = 50

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "Filter:")
	assert.Contains(t, view, "ID")
	assert.Contains(t, view, "Name")
	assert.Contains(t, view, "Priority")
	assert.Contains(t, view, "Status")
}

func TestTaskQueueModel_View_Details(t *testing.T) {
	model := NewTaskQueueModel(tui.DarkTheme)
	model.width = 100
	model.height = 50
	now := time.Now()
	task := &tui.Task{
		ID:          "task-1",
		Name:        "test-task",
		Type:        tui.TaskTypeCode,
		Status:      tui.TaskStatusRunning,
		Priority:    tui.PriorityHigh,
		CreatedAt:   now,
		StartedAt:   &now,
		Progress:    50.0,
		Description: "Test description",
		Output:      "Test output\nLine 2\nLine 3",
	}
	model.taskDetail = task
	model.showDetails = true

	view := model.View()

	assert.NotEmpty(t, view)
	assert.Contains(t, view, "TASK: task-1")
	assert.Contains(t, view, "TASK DETAILS")
}
