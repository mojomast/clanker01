package orchestrator

import (
	"context"
	"fmt"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type RecoveryStrategy string

const (
	RecoveryStrategyRetry     RecoveryStrategy = "retry"
	RecoveryStrategyReassign  RecoveryStrategy = "reassign"
	RecoveryStrategyDecompose RecoveryStrategy = "decompose"
	RecoveryStrategyEscalate  RecoveryStrategy = "escalate"
)

type RecoveryManager struct {
	orchestrator *Orchestrator
	scheduler    *Scheduler
	config       *RecoveryConfig
}

type RecoveryConfig struct {
	MaxRetries   int
	RetryDelay   time.Duration
	RetryBackoff float64
	MaxReassigns int
	Timeout      time.Duration
}

func NewRecoveryManager(orch *Orchestrator, scheduler *Scheduler, cfg *RecoveryConfig) *RecoveryManager {
	if cfg == nil {
		cfg = &RecoveryConfig{
			MaxRetries:   3,
			RetryDelay:   1 * time.Second,
			RetryBackoff: 2.0,
			MaxReassigns: 2,
			Timeout:      5 * time.Minute,
		}
	}

	return &RecoveryManager{
		orchestrator: orch,
		scheduler:    scheduler,
		config:       cfg,
	}
}

func (rm *RecoveryManager) HandleAgentFailure(ctx context.Context, agentID string, err error) error {
	task := rm.getAgentCurrentTask(agentID)
	if task == nil {
		return nil
	}

	strategy := rm.determineStrategy(err)

	switch strategy {
	case RecoveryStrategyRetry:
		return rm.retryTask(ctx, task)
	case RecoveryStrategyReassign:
		return rm.reassignTask(ctx, task)
	case RecoveryStrategyDecompose:
		return rm.decomposeTask(ctx, task)
	case RecoveryStrategyEscalate:
		return rm.escalate(ctx, task, err)
	}

	return nil
}

func (rm *RecoveryManager) HandleTaskFailure(ctx context.Context, taskID string, err error) error {
	task := rm.orchestrator.taskQueue.GetTask(taskID)
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	strategy := rm.determineStrategy(err)

	switch strategy {
	case RecoveryStrategyRetry:
		return rm.retryTask(ctx, task)
	case RecoveryStrategyReassign:
		return rm.reassignTask(ctx, task)
	case RecoveryStrategyDecompose:
		return rm.decomposeTask(ctx, task)
	case RecoveryStrategyEscalate:
		return rm.escalate(ctx, task, err)
	}

	return nil
}

func (rm *RecoveryManager) determineStrategy(err error) RecoveryStrategy {
	if isTransientError(err) {
		return RecoveryStrategyRetry
	}
	if isAgentError(err) {
		return RecoveryStrategyReassign
	}
	if isComplexityError(err) {
		return RecoveryStrategyDecompose
	}
	return RecoveryStrategyEscalate
}

func (rm *RecoveryManager) retryTask(ctx context.Context, task *api.Task) error {
	if task.RetryCount >= rm.config.MaxRetries {
		return fmt.Errorf("max retries exceeded for task %s", task.ID)
	}

	task.RetryCount++
	task.Status = api.TaskStatusQueued
	task.Error = nil

	delay := rm.config.RetryDelay * time.Duration(1/rm.config.RetryBackoff*float64(task.RetryCount))
	time.Sleep(delay)

	return rm.orchestrator.taskQueue.Enqueue(ctx, task)
}

func (rm *RecoveryManager) reassignTask(ctx context.Context, task *api.Task) error {
	if task.RetryCount >= rm.config.MaxReassigns {
		return fmt.Errorf("max reassigns exceeded for task %s", task.ID)
	}

	task.RetryCount++
	task.AssignedAgent = ""
	task.Status = api.TaskStatusQueued
	task.Error = nil

	return rm.scheduler.ReassignTask(ctx, task.ID)
}

func (rm *RecoveryManager) decomposeTask(ctx context.Context, task *api.Task) error {
	subtasks, err := rm.orchestrator.DecomposeTask(ctx, task)
	if err != nil {
		return fmt.Errorf("decompose task: %w", err)
	}

	for _, subtask := range subtasks {
		if err := rm.orchestrator.taskQueue.Enqueue(ctx, subtask); err != nil {
			return fmt.Errorf("enqueue subtask: %w", err)
		}
	}

	return nil
}

func (rm *RecoveryManager) escalate(ctx context.Context, task *api.Task, err error) error {
	return fmt.Errorf("task %s failed and requires manual intervention: %w", task.ID, err)
}

func (rm *RecoveryManager) getAgentCurrentTask(agentID string) *api.Task {
	taskID, ok := rm.scheduler.GetAssignment(agentID)
	if !ok {
		return nil
	}
	return rm.orchestrator.taskQueue.GetTask(taskID)
}

func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	transientErrors := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"rate limit",
		"service unavailable",
	}

	for _, pattern := range transientErrors {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isAgentError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	agentErrors := []string{
		"agent crashed",
		"agent terminated",
		"agent failed to initialize",
		"agent not responding",
	}

	for _, pattern := range agentErrors {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func isComplexityError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	complexityErrors := []string{
		"too complex",
		"exceeded context",
		"task too large",
		"needs decomposition",
	}

	for _, pattern := range complexityErrors {
		if contains(errStr, pattern) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
