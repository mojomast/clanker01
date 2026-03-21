package task

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProgressTracker(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	assert.NotNil(t, tracker)
	assert.NotNil(t, tracker.tasks)
	assert.Equal(t, "plan-1", tracker.planID)
}

func TestProgressTracker_Initialize(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{
			ID:      "task-1",
			Name:    "Task 1",
			Status:  StatusPending,
			Timeout: time.Minute,
		},
		{
			ID:      "task-2",
			Name:    "Task 2",
			Status:  StatusPending,
			Timeout: time.Minute,
		},
	}

	tracker.Initialize(tasks)
	assert.Equal(t, 2, tracker.TaskCount())
}

func TestProgressTracker_Update(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	taskID := TaskID("task-1")
	status := StatusRunning
	percent := 50.0
	message := "Processing"

	err := tracker.Update(taskID, ProgressUpdate{
		Status:  &status,
		Percent: &percent,
		Message: &message,
	})
	require.NoError(t, err)

	progress, err := tracker.Get(taskID)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, progress.Status)
	assert.Equal(t, 50.0, progress.Percent)
	assert.Equal(t, "Processing", progress.Message)
}

func TestProgressTracker_Update_CreateIfNotExists(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	taskID := TaskID("task-1")
	status := StatusRunning

	err := tracker.Update(taskID, ProgressUpdate{
		Status: &status,
	})
	require.NoError(t, err)

	progress, err := tracker.Get(taskID)
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, progress.Status)
}

func TestProgressTracker_Get(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	taskID := TaskID("task-1")
	status := StatusCompleted

	tracker.Update(taskID, ProgressUpdate{Status: &status})

	progress, err := tracker.Get(taskID)
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, progress.Status)
}

func TestProgressTracker_Get_NotFound(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	progress, err := tracker.Get(TaskID("non-existent"))
	assert.Error(t, err)
	assert.Nil(t, progress)
	assert.True(t, IsTaskNotFound(err))
}

func TestProgressTracker_GetPlanProgress(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-3", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	tracker.MarkTaskStarted("task-1")
	tracker.MarkTaskCompleted("task-1")

	progress := tracker.GetPlanProgress()
	assert.Equal(t, "plan-1", progress.PlanID)
	assert.Equal(t, 3, progress.TotalTasks)
	assert.Equal(t, 1, progress.Completed)
	assert.Equal(t, 0, progress.Running)
	assert.Equal(t, 2, progress.Pending)
	assert.Equal(t, 0, progress.Failed)
	assert.Equal(t, 33.33333333333333, progress.Percent)
}

func TestProgressTracker_GetPlanProgress_WithETA(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-3", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	tracker.MarkTaskStarted("task-1")
	tracker.UpdatePercent("task-1", 50)
	tracker.MarkTaskCompleted("task-1")

	progress := tracker.GetPlanProgress()
	assert.Equal(t, 1, progress.Completed)
	assert.Greater(t, progress.ETA, time.Duration(0))
}

func TestProgressTracker_GetTasksByStatus(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-3", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	tracker.MarkTaskStarted("task-1")
	tracker.MarkTaskCompleted("task-2")

	pending := tracker.GetTasksByStatus(StatusPending)
	running := tracker.GetTasksByStatus(StatusRunning)
	completed := tracker.GetTasksByStatus(StatusCompleted)

	assert.Len(t, pending, 1)
	assert.Len(t, running, 1)
	assert.Len(t, completed, 1)
}

func TestProgressTracker_SetSubProgress(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	err := tracker.SetSubProgress("task-1", "sub1", 50.0)
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, 50.0, progress.SubProgress["sub1"])
	assert.Equal(t, 50.0, progress.Percent)
}

func TestProgressTracker_SetSubProgress_Multiple(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	tracker.SetSubProgress("task-1", "sub1", 25.0)
	tracker.SetSubProgress("task-1", "sub2", 75.0)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, 25.0, progress.SubProgress["sub1"])
	assert.Equal(t, 75.0, progress.SubProgress["sub2"])
	assert.Equal(t, 50.0, progress.Percent)
}

func TestProgressTracker_UpdateStatus(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	err := tracker.UpdateStatus("task-1", StatusRunning)
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, progress.Status)
	assert.False(t, progress.StartedAt.IsZero())
}

func TestProgressTracker_UpdatePercent(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	err := tracker.UpdatePercent("task-1", 75.0)
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, 75.0, progress.Percent)
}

func TestProgressTracker_UpdateMessage(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	message := "Processing data"
	err := tracker.UpdateMessage("task-1", message)
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, message, progress.Message)
}

func TestProgressTracker_UpdateError(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	errMsg := "Task failed"
	err := tracker.UpdateError("task-1", errMsg)
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, errMsg, progress.Error)
}

func TestProgressTracker_MarkTaskStarted(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	err := tracker.MarkTaskStarted("task-1")
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, StatusRunning, progress.Status)
	assert.False(t, progress.StartedAt.IsZero())
}

func TestProgressTracker_MarkTaskCompleted(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	err := tracker.MarkTaskCompleted("task-1")
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, StatusCompleted, progress.Status)
	assert.Equal(t, 100.0, progress.Percent)
}

func TestProgressTracker_MarkTaskFailed(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	errMsg := "Task failed"
	err := tracker.MarkTaskFailed("task-1", errMsg)
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, StatusFailed, progress.Status)
	assert.Equal(t, errMsg, progress.Error)
}

func TestProgressTracker_Reset(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)
	tracker.MarkTaskStarted("task-1")

	assert.Equal(t, 1, tracker.TaskCount())

	tracker.Reset()
	assert.Equal(t, 0, tracker.TaskCount())
}

func TestProgressTracker_GetAllProgress(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)
	tracker.MarkTaskStarted("task-1")

	all := tracker.GetAllProgress()
	assert.Len(t, all, 2)
	assert.Equal(t, StatusRunning, all["task-1"].Status)
	assert.Equal(t, StatusPending, all["task-2"].Status)
}

func TestProgressTracker_TaskCount(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	assert.Equal(t, 0, tracker.TaskCount())

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)
	assert.Equal(t, 2, tracker.TaskCount())
}

func TestProgressTracker_IsComplete(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)
	assert.False(t, tracker.IsComplete())

	tracker.MarkTaskCompleted("task-1")
	assert.False(t, tracker.IsComplete())

	tracker.MarkTaskCompleted("task-2")
	assert.True(t, tracker.IsComplete())
}

func TestProgressTracker_HasFailedTasks(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)
	assert.False(t, tracker.HasFailedTasks())

	tracker.MarkTaskCompleted("task-1")
	assert.False(t, tracker.HasFailedTasks())

	tracker.MarkTaskFailed("task-2", "error")
	assert.True(t, tracker.HasFailedTasks())
}

func TestProgressTracker_WithFailedAndCancelled(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-3", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	tracker.MarkTaskCompleted("task-1")
	tracker.UpdateStatus("task-2", StatusFailed)
	tracker.UpdateStatus("task-3", StatusCancelled)

	progress := tracker.GetPlanProgress()
	assert.Equal(t, 1, progress.Completed)
	assert.Equal(t, 2, progress.Failed)
}

func TestProgressTracker_WithBlockedTasks(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusBlocked, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	progress := tracker.GetPlanProgress()
	assert.Equal(t, 2, progress.Pending)
	assert.Equal(t, 0, progress.Failed)
}

func TestProgressTracker_UpdateWithProgress(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	subProgress := map[string]float64{
		"sub1": 30.0,
		"sub2": 70.0,
	}

	err := tracker.Update("task-1", ProgressUpdate{
		Progress: &TaskProgress{SubProgress: subProgress},
	})
	require.NoError(t, err)

	progress, err := tracker.Get("task-1")
	require.NoError(t, err)
	assert.Equal(t, subProgress, progress.SubProgress)
}

func TestProgressTracker_PercentCalculation(t *testing.T) {
	tracker := NewProgressTracker("plan-1")

	tasks := []*Task{
		{ID: "task-1", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-2", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-3", Status: StatusPending, Timeout: time.Minute},
		{ID: "task-4", Status: StatusPending, Timeout: time.Minute},
	}

	tracker.Initialize(tasks)

	tracker.MarkTaskCompleted("task-1")
	tracker.MarkTaskCompleted("task-2")

	progress := tracker.GetPlanProgress()
	assert.Equal(t, 50.0, progress.Percent)
}
