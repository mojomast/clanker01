package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewTaskQueue(t *testing.T) {
	q := NewTaskQueue()
	assert.NotNil(t, q)
	assert.NotNil(t, q.pending)
	assert.NotNil(t, q.dependencies)
}

func TestTaskQueue_Enqueue(t *testing.T) {
	q := NewTaskQueue()
	task := &api.Task{
		ID:       "task-1",
		Priority: 10,
		Prompt:   "Test task",
	}

	err := q.Enqueue(nil, task)
	assert.NoError(t, err)
	assert.Equal(t, api.TaskStatusQueued, task.Status)
	assert.Equal(t, 1, q.pending.Len())
}

func TestTaskQueue_EnqueueWithDependencies(t *testing.T) {
	q := NewTaskQueue()

	task1 := &api.Task{
		ID:       "task-1",
		Priority: 10,
		Prompt:   "First task",
	}
	task2 := &api.Task{
		ID:           "task-2",
		Priority:     10,
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}

	err := q.Enqueue(nil, task1)
	assert.NoError(t, err)

	err = q.Enqueue(nil, task2)
	assert.NoError(t, err)
	assert.Equal(t, api.TaskStatusQueued, task1.Status)
	assert.Equal(t, api.TaskStatusBlocked, task2.Status)
}

func TestTaskQueue_MatchesAgent(t *testing.T) {
	q := NewTaskQueue()

	task := &api.Task{
		ID:        "task-1",
		Priority:  10,
		Prompt:    "Test task",
		AgentType: api.AgentTypeCoder,
	}

	assert.True(t, q.matchesAgent(task, api.AgentTypeCoder))
	assert.False(t, q.matchesAgent(task, api.AgentTypeTester))

	task.AgentType = api.AgentType("")
	assert.True(t, q.matchesAgent(task, api.AgentTypeCoder))
	assert.True(t, q.matchesAgent(task, api.AgentTypeTester))
	assert.True(t, q.matchesAgent(task, api.AgentType("")))
}
