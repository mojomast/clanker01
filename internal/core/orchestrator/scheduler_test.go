package orchestrator

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewScheduler(t *testing.T) {
	taskQueue := NewTaskQueue()
	cfg := &SchedulerConfig{
		ScheduleInterval: 100,
		MaxConcurrent:    5,
	}

	s := NewScheduler(nil, taskQueue, cfg, nil)

	assert.NotNil(t, s)
	assert.Equal(t, cfg.ScheduleInterval, s.config.ScheduleInterval)
	assert.Equal(t, cfg.MaxConcurrent, s.config.MaxConcurrent)
	assert.NotNil(t, s.assignments)
	assert.NotNil(t, s.agentLoads)
}

func TestScheduler_GetAssignment(t *testing.T) {
	taskQueue := NewTaskQueue()
	s := NewScheduler(nil, taskQueue, nil, nil)

	task := &api.Task{ID: "task-1", Priority: 10, Prompt: "Test task"}
	taskQueue.Enqueue(nil, task)

	s.mu.Lock()
	s.assignments["task-1"] = "agent-1"
	s.mu.Unlock()

	agentID, ok := s.GetAssignment("task-1")
	assert.True(t, ok)
	assert.Equal(t, "agent-1", agentID)
}

func TestScheduler_GetAssignment_NotFound(t *testing.T) {
	taskQueue := NewTaskQueue()
	s := NewScheduler(nil, taskQueue, nil, nil)

	_, ok := s.GetAssignment("nonexistent")
	assert.False(t, ok)
}

func TestScheduler_GetAgentLoad(t *testing.T) {
	taskQueue := NewTaskQueue()
	s := NewScheduler(nil, taskQueue, nil, nil)

	s.mu.Lock()
	s.agentLoads["agent-1"] = 3
	s.mu.Unlock()

	load := s.GetAgentLoad("agent-1")
	assert.Equal(t, 3, load)
}

func TestScheduler_GetAgentLoad_NotFound(t *testing.T) {
	taskQueue := NewTaskQueue()
	s := NewScheduler(nil, taskQueue, nil, nil)

	load := s.GetAgentLoad("nonexistent")
	assert.Equal(t, 0, load)
}

func TestScheduler_PickLeastLoaded(t *testing.T) {
	taskQueue := NewTaskQueue()
	s := NewScheduler(nil, taskQueue, nil, nil)

	agents := []api.Agent{
		&mockAgent{id: "agent-1"},
		&mockAgent{id: "agent-2"},
		&mockAgent{id: "agent-3"},
	}

	s.mu.Lock()
	s.agentLoads["agent-1"] = 5
	s.agentLoads["agent-2"] = 2
	s.agentLoads["agent-3"] = 4
	s.mu.Unlock()

	best := s.pickLeastLoaded(agents)
	assert.Equal(t, "agent-2", best.ID())
}

func TestScheduler_GetStats(t *testing.T) {
	taskQueue := NewTaskQueue()
	s := NewScheduler(nil, taskQueue, nil, nil)

	s.mu.Lock()
	s.assignments["task-1"] = "agent-1"
	s.assignments["task-2"] = "agent-1"
	s.assignments["task-3"] = "agent-2"
	s.agentLoads["agent-1"] = 2
	s.agentLoads["agent-2"] = 1
	s.mu.Unlock()

	assigned, totalLoad := s.GetStats()
	assert.Equal(t, 3, assigned)
	assert.Equal(t, 3, totalLoad)
}

func TestScheduler_ReassignTask_NotAssigned(t *testing.T) {
	taskQueue := NewTaskQueue()
	s := NewScheduler(nil, taskQueue, nil, nil)

	err := s.ReassignTask(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "task not assigned")
}

type mockAgent struct {
	id string
}

func (m *mockAgent) ID() string          { return m.id }
func (m *mockAgent) Type() api.AgentType { return api.AgentTypeCoder }
func (m *mockAgent) Name() string        { return "mock" }
func (m *mockAgent) Initialize(ctx context.Context, config *api.AgentConfig) error {
	return nil
}
func (m *mockAgent) Start(ctx context.Context) error  { return nil }
func (m *mockAgent) Stop(ctx context.Context) error   { return nil }
func (m *mockAgent) Pause(ctx context.Context) error  { return nil }
func (m *mockAgent) Resume(ctx context.Context) error { return nil }
func (m *mockAgent) Execute(ctx context.Context, task *api.Task) (*api.AgentResult, error) {
	return &api.AgentResult{TaskID: task.ID, Success: true}, nil
}
func (m *mockAgent) SendMessage(ctx context.Context, msg *api.AgentMessage) error { return nil }
func (m *mockAgent) ReceiveMessage() <-chan *api.AgentMessage                     { return nil }
func (m *mockAgent) Broadcast(ctx context.Context, msg *api.AgentMessage) error   { return nil }
func (m *mockAgent) Status() api.AgentStatus                                      { return api.AgentStatusReady }
func (m *mockAgent) CurrentTask() *api.Task                                       { return nil }
func (m *mockAgent) Metrics() *api.AgentMetrics                                   { return &api.AgentMetrics{} }
func (m *mockAgent) Health() *api.AgentHealth                                     { return &api.AgentHealth{} }
