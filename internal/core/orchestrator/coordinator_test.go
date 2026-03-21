package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewOrchestrator(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 100 * time.Millisecond,
		MaxConcurrent:    10,
		MaxRetries:       3,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)
	assert.NotNil(t, orch)
	assert.NotNil(t, orch.taskQueue)
	assert.NotNil(t, orch.scheduler)
	assert.NotNil(t, orch.stateManager)
	assert.NotNil(t, orch.recovery)
	assert.NotNil(t, orch.conflictMgr)
}

func TestOrchestrator_StartAndStop(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
		MaxConcurrent:    10,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = orch.Start(ctx)
	require.NoError(t, err)

	orch.mu.RLock()
	running := orch.running
	orch.mu.RUnlock()

	assert.True(t, running)

	err = orch.Stop(ctx)
	require.NoError(t, err)

	orch.mu.RLock()
	running = orch.running
	orch.mu.RUnlock()

	assert.False(t, running)
}

func TestOrchestrator_SubmitTask(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{
		Priority:  10,
		Prompt:    "Test task",
		AgentType: api.AgentTypeCoder,
	}

	err = orch.SubmitTask(context.Background(), task)
	require.NoError(t, err)

	assert.NotEmpty(t, task.ID)
	assert.Equal(t, api.TaskStatusQueued, task.Status)
	assert.False(t, task.CreatedAt.IsZero())
}

func TestOrchestrator_SubmitTaskWithDependencies(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task1 := &api.Task{
		Priority:  10,
		Prompt:    "First task",
		AgentType: api.AgentTypeCoder,
	}

	err = orch.SubmitTask(context.Background(), task1)
	require.NoError(t, err)

	task2 := &api.Task{
		Priority:     10,
		Prompt:       "Second task",
		AgentType:    api.AgentTypeCoder,
		Dependencies: []string{task1.ID},
	}

	err = orch.SubmitTask(context.Background(), task2)
	require.NoError(t, err)

	assert.Equal(t, api.TaskStatusQueued, task1.Status)
	assert.Equal(t, api.TaskStatusBlocked, task2.Status)
}

func TestOrchestrator_SubmitTaskWithParent(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task1 := &api.Task{
		Priority:  10,
		Prompt:    "First task",
		AgentType: api.AgentTypeCoder,
	}

	err = orch.SubmitTask(context.Background(), task1)
	require.NoError(t, err)

	task2 := &api.Task{
		ParentID:  task1.ID,
		Priority:  10,
		Prompt:    "Second task",
		AgentType: api.AgentTypeCoder,
	}

	err = orch.SubmitTask(context.Background(), task2)
	require.NoError(t, err)

	assert.Contains(t, task2.Dependencies, task1.ID)
}

func TestOrchestrator_SubmitTaskCircularDependency(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task1 := &api.Task{
		ID:           "task-1",
		Priority:     10,
		Prompt:       "First task",
		AgentType:    api.AgentTypeCoder,
		Dependencies: []string{"task-2"},
	}

	task2 := &api.Task{
		ID:           "task-2",
		Priority:     10,
		Prompt:       "Second task",
		AgentType:    api.AgentTypeCoder,
		Dependencies: []string{task1.ID},
	}

	err = orch.SubmitTask(context.Background(), task1)
	require.NoError(t, err)

	err = orch.SubmitTask(context.Background(), task2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "circular dependency")
}

func TestOrchestrator_GetTask(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{
		Priority:  10,
		Prompt:    "Test task",
		AgentType: api.AgentTypeCoder,
	}

	err = orch.SubmitTask(context.Background(), task)
	require.NoError(t, err)

	retrieved := orch.GetTask(task.ID)
	assert.NotNil(t, retrieved)
	assert.Equal(t, task.ID, retrieved.ID)
}

func TestOrchestrator_GetTaskStatus(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{
		Priority:  10,
		Prompt:    "Test task",
		AgentType: api.AgentTypeCoder,
	}

	err = orch.SubmitTask(context.Background(), task)
	require.NoError(t, err)

	status := orch.GetTaskStatus(task.ID)
	assert.Equal(t, api.TaskStatusQueued, status)
}

func TestOrchestrator_GetStats(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task1 := &api.Task{Priority: 10, Prompt: "Task 1"}
	task2 := &api.Task{Priority: 5, Prompt: "Task 2"}

	orch.SubmitTask(context.Background(), task1)
	orch.SubmitTask(context.Background(), task2)

	stats := orch.GetStats()
	assert.NotNil(t, stats)
	assert.Equal(t, 2, stats.PendingTasks)
	assert.Equal(t, 0, stats.RunningTasks)
	assert.Equal(t, 0, stats.CompletedTasks)
	assert.Equal(t, 0, stats.FailedTasks)
	assert.Equal(t, int64(2), stats.TasksSubmitted)
}

func TestOrchestrator_CreateCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	checkpoint, err := orch.CreateCheckpoint("test-checkpoint")
	require.NoError(t, err)
	assert.NotNil(t, checkpoint)
	assert.Equal(t, "test-checkpoint", checkpoint.Name)
}

func TestOrchestrator_ListCheckpoints(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	_, err = orch.CreateCheckpoint("checkpoint-1")
	require.NoError(t, err)

	_, err = orch.CreateCheckpoint("checkpoint-2")
	require.NoError(t, err)

	checkpoints, err := orch.ListCheckpoints()
	require.NoError(t, err)
	assert.Len(t, checkpoints, 2)
}

func TestOrchestrator_RestoreCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{Priority: 10, Prompt: "Test task"}
	orch.SubmitTask(context.Background(), task)

	checkpoint, err := orch.CreateCheckpoint("test-checkpoint")
	require.NoError(t, err)

	orch2, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	err = orch2.RestoreCheckpoint(checkpoint.ID)
	require.NoError(t, err)

	stats := orch2.GetStats()
	assert.Equal(t, int64(1), stats.TasksSubmitted)
}

func TestOrchestrator_DecomposeTask(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{
		ID:        "task-1",
		Priority:  10,
		Prompt:    "Test task",
		AgentType: api.AgentTypeCoder,
	}

	subtasks, err := orch.DecomposeTask(context.Background(), task)
	require.NoError(t, err)
	assert.NotEmpty(t, subtasks)
	assert.Equal(t, task.ID, subtasks[0].ParentID)
}

func TestOrchestrator_SplitTask(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{
		ID:        "task-1",
		Priority:  10,
		Prompt:    "This is a very long task that should be split into multiple parts",
		AgentType: api.AgentTypeCoder,
	}

	subtasks, err := orch.DecomposeTask(context.Background(), task)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(subtasks), 1)
}

func TestOrchestrator_SyncState(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{Priority: 10, Prompt: "Test task"}
	orch.SubmitTask(context.Background(), task)

	orch.syncState()

	state := orch.stateManager.GetState()
	assert.NotNil(t, state.Queue)
	assert.Greater(t, state.Queue.PendingCount, 0)
}

func TestConflictManager(t *testing.T) {
	cm := &ConflictManager{
		conflicts: make(map[string]*Conflict),
	}

	conflict := &Conflict{
		ID:               "conflict-1",
		Type:             ConflictTypeDependency,
		Resource:         "task-1",
		ConflictingTasks: []string{"task-1", "task-2"},
		DetectedAt:       time.Now(),
	}

	cm.mu.Lock()
	cm.conflicts[conflict.ID] = conflict
	cm.mu.Unlock()

	cm.mu.RLock()
	retrieved := cm.conflicts["conflict-1"]
	cm.mu.RUnlock()

	assert.NotNil(t, retrieved)
	assert.Equal(t, "conflict-1", retrieved.ID)
}

func TestOrchestratorStop_SavesState(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &OrchestratorConfig{
		DataDir:          tmpDir,
		ScheduleInterval: 10 * time.Millisecond,
	}

	orch, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	task := &api.Task{Priority: 10, Prompt: "Test task"}
	orch.SubmitTask(context.Background(), task)

	ctx := context.Background()
	orch.Start(ctx)

	err = orch.Stop(ctx)
	require.NoError(t, err)

	orch2, err := NewOrchestrator(cfg)
	require.NoError(t, err)

	state, err := orch2.stateManager.Load()
	require.NoError(t, err)
	assert.NotNil(t, state)
	assert.Equal(t, int64(1), state.Metrics.TasksSubmitted)
}
