package orchestrator

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewRecoveryManager(t *testing.T) {
	taskQueue := NewTaskQueue()
	scheduler := NewScheduler(nil, taskQueue, nil, nil)

	rm := NewRecoveryManager(nil, scheduler, nil)

	assert.NotNil(t, rm)
	assert.NotNil(t, rm.config)
	assert.Equal(t, 3, rm.config.MaxRetries)
	assert.Equal(t, 2.0, rm.config.RetryBackoff)
}

func TestRecoveryManager_WithCustomConfig(t *testing.T) {
	taskQueue := NewTaskQueue()
	scheduler := NewScheduler(nil, taskQueue, nil, nil)

	cfg := &RecoveryConfig{
		MaxRetries:   5,
		RetryDelay:   2 * time.Second,
		RetryBackoff: 3.0,
		MaxReassigns: 3,
	}

	rm := NewRecoveryManager(nil, scheduler, cfg)

	assert.Equal(t, 5, rm.config.MaxRetries)
	assert.Equal(t, 2*time.Second, rm.config.RetryDelay)
	assert.Equal(t, 3.0, rm.config.RetryBackoff)
	assert.Equal(t, 3, rm.config.MaxReassigns)
}

func TestIsTransientError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"timeout error", errors.New("request timeout"), true},
		{"connection refused", errors.New("connection refused"), true},
		{"temporary failure", errors.New("temporary failure"), true},
		{"rate limit", errors.New("rate limit exceeded"), true},
		{"service unavailable", errors.New("service unavailable"), true},
		{"other error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTransientError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsAgentError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"agent crashed", errors.New("agent crashed"), true},
		{"agent terminated", errors.New("agent terminated"), true},
		{"agent failed to initialize", errors.New("agent failed to initialize"), true},
		{"agent not responding", errors.New("agent not responding"), true},
		{"other error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isAgentError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsComplexityError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"too complex", errors.New("task too complex"), true},
		{"exceeded context", errors.New("exceeded context window"), true},
		{"task too large", errors.New("task too large"), true},
		{"needs decomposition", errors.New("task needs decomposition"), true},
		{"other error", errors.New("some other error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isComplexityError(tt.err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRecoveryManager_DetermineStrategy(t *testing.T) {
	taskQueue := NewTaskQueue()
	scheduler := NewScheduler(nil, taskQueue, nil, nil)
	rm := NewRecoveryManager(nil, scheduler, nil)

	strategy := rm.determineStrategy(errors.New("request timeout"))
	assert.Equal(t, RecoveryStrategyRetry, strategy)

	strategy = rm.determineStrategy(errors.New("agent crashed"))
	assert.Equal(t, RecoveryStrategyReassign, strategy)

	strategy = rm.determineStrategy(errors.New("task too complex"))
	assert.Equal(t, RecoveryStrategyDecompose, strategy)

	strategy = rm.determineStrategy(errors.New("some other error"))
	assert.Equal(t, RecoveryStrategyEscalate, strategy)
}

func TestRecoveryManager_RetryTask(t *testing.T) {
	taskQueue := NewTaskQueue()
	scheduler := NewScheduler(nil, taskQueue, nil, nil)

	orch := &Orchestrator{
		taskQueue:   taskQueue,
		scheduler:   scheduler,
		conflictMgr: &ConflictManager{},
	}

	rm := NewRecoveryManager(orch, scheduler, nil)

	task := &api.Task{
		ID:         "task-1",
		Priority:   10,
		Prompt:     "Test task",
		MaxRetries: 3,
		RetryCount: 0,
		Status:     api.TaskStatusFailed,
		Error:      errors.New("temporary failure"),
	}

	taskQueue.Enqueue(nil, task)

	err := rm.retryTask(context.Background(), task)
	assert.NoError(t, err)
	assert.Equal(t, 1, task.RetryCount)
	assert.Equal(t, api.TaskStatusQueued, task.Status)
}
