package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewStateManager(t *testing.T) {
	tmpDir := t.TempDir()

	sm, err := NewStateManager(tmpDir)
	require.NoError(t, err)
	assert.NotNil(t, sm)
	assert.NotNil(t, sm.currentState)
	assert.Equal(t, tmpDir, sm.dataDir)
}

func TestStateManager_GetState(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	state := sm.GetState()
	assert.NotNil(t, state)
	assert.Equal(t, "1.0.0", state.Version)
	assert.NotNil(t, state.Agents)
	assert.NotNil(t, state.Tasks)
}

func TestStateManager_UpdateAgentState(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	agentState := &AgentState{
		ID:           "agent-1",
		Type:         api.AgentTypeCoder,
		Status:       api.AgentStatusReady,
		CurrentTask:  "task-1",
		LastActivity: time.Now(),
		Metrics:      api.AgentMetrics{},
	}

	sm.UpdateAgentState("agent-1", agentState)

	state := sm.GetState()
	assert.Equal(t, agentState, state.Agents["agent-1"])
}

func TestStateManager_UpdateTaskState(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	taskState := &TaskState{
		ID:            "task-1",
		Status:        api.TaskStatusRunning,
		AssignedAgent: "agent-1",
		Progress:      0.5,
		StartedAt:     time.Now(),
	}

	sm.UpdateTaskState("task-1", taskState)

	state := sm.GetState()
	assert.Equal(t, taskState, state.Tasks["task-1"])
}

func TestStateManager_UpdateQueueState(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	queueState := &QueueState{
		PendingCount:   5,
		RunningCount:   3,
		CompletedCount: 10,
		FailedCount:    2,
	}

	sm.UpdateQueueState(queueState)

	state := sm.GetState()
	assert.Equal(t, queueState, state.Queue)
}

func TestStateManager_UpdateMetrics(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	metrics := &OrchestratorMetrics{
		TasksSubmitted:  100,
		TasksCompleted:  90,
		TasksFailed:     5,
		TotalDuration:   time.Hour,
		AvgTaskDuration: time.Minute,
	}

	sm.UpdateMetrics(metrics)

	state := sm.GetState()
	assert.Equal(t, metrics, state.Metrics)
}

func TestStateManager_SaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	agentState := &AgentState{
		ID:           "agent-1",
		Type:         api.AgentTypeCoder,
		Status:       api.AgentStatusReady,
		LastActivity: time.Now(),
		Metrics:      api.AgentMetrics{},
	}

	sm.UpdateAgentState("agent-1", agentState)

	err := sm.Save()
	require.NoError(t, err)

	sm2, err := NewStateManager(tmpDir)
	require.NoError(t, err)

	loadedState, err := sm2.Load()
	require.NoError(t, err)

	assert.NotNil(t, loadedState)
	assert.Equal(t, "agent-1", loadedState.Agents["agent-1"].ID)
	assert.Equal(t, api.AgentTypeCoder, loadedState.Agents["agent-1"].Type)
}

func TestStateManager_Load_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	loadedState, err := sm.Load()
	require.NoError(t, err)
	assert.NotNil(t, loadedState)
}

func TestStateManager_CreateCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	agentState := &AgentState{
		ID:   "agent-1",
		Type: api.AgentTypeCoder,
	}

	sm.UpdateAgentState("agent-1", agentState)

	checkpoint, err := sm.CreateCheckpoint("test-checkpoint")
	require.NoError(t, err)
	assert.NotNil(t, checkpoint)
	assert.NotEmpty(t, checkpoint.ID)
	assert.Equal(t, "test-checkpoint", checkpoint.Name)
	assert.NotNil(t, checkpoint.State)
}

func TestStateManager_RestoreCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	agentState := &AgentState{
		ID:   "agent-1",
		Type: api.AgentTypeCoder,
	}

	sm.UpdateAgentState("agent-1", agentState)

	checkpoint, err := sm.CreateCheckpoint("test-checkpoint")
	require.NoError(t, err)

	sm2, err := NewStateManager(tmpDir)
	require.NoError(t, err)

	restoredState, err := sm2.RestoreCheckpoint(checkpoint.ID)
	require.NoError(t, err)

	assert.NotNil(t, restoredState)
	assert.Equal(t, "agent-1", restoredState.Agents["agent-1"].ID)
}

func TestStateManager_ListCheckpoints(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	_, err := sm.CreateCheckpoint("checkpoint-1")
	require.NoError(t, err)

	_, err = sm.CreateCheckpoint("checkpoint-2")
	require.NoError(t, err)

	checkpoints, err := sm.ListCheckpoints()
	require.NoError(t, err)
	assert.Len(t, checkpoints, 2)
}

func TestStateManager_DeleteCheckpoint(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	checkpoint, err := sm.CreateCheckpoint("test-checkpoint")
	require.NoError(t, err)

	err = sm.DeleteCheckpoint(checkpoint.ID)
	require.NoError(t, err)

	checkpoints, err := sm.ListCheckpoints()
	require.NoError(t, err)
	assert.Len(t, checkpoints, 0)
}

func TestStateManager_Clear(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	agentState := &AgentState{ID: "agent-1", Type: api.AgentTypeCoder}
	sm.UpdateAgentState("agent-1", agentState)

	_, err := sm.CreateCheckpoint("test-checkpoint")
	require.NoError(t, err)

	err = sm.Clear()
	require.NoError(t, err)

	state := sm.GetState()
	assert.Empty(t, state.Agents)
	assert.Empty(t, state.Tasks)
}

func TestStateManager_StateFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	agentState := &AgentState{ID: "agent-1", Type: api.AgentTypeCoder}
	sm.UpdateAgentState("agent-1", agentState)

	err := sm.Save()
	require.NoError(t, err)

	statePath := filepath.Join(tmpDir, "state.json")
	info, err := os.Stat(statePath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestStateManager_CheckpointDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	sm, _ := NewStateManager(tmpDir)

	_, err := sm.CreateCheckpoint("test-checkpoint")
	require.NoError(t, err)

	checkpointsDir := filepath.Join(tmpDir, "checkpoints")
	info, err := os.Stat(checkpointsDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
}
