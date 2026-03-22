package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/internal/core/task"
	"github.com/swarm-ai/swarm/pkg/api"
)

type Scheduler struct {
	orchestrator *Orchestrator
	taskQueue    *TaskQueue
	config       *SchedulerConfig
	verifier     *task.Verifier

	mu          sync.RWMutex
	assignments map[string]string // taskID -> agentID
	agentLoads  map[string]int    // agentID -> load
}

type SchedulerConfig struct {
	ScheduleInterval time.Duration
	MaxConcurrent    int
}

func NewScheduler(orch *Orchestrator, queue *TaskQueue, cfg *SchedulerConfig, verifier *task.Verifier) *Scheduler {
	if cfg == nil {
		cfg = &SchedulerConfig{
			ScheduleInterval: 100 * time.Millisecond,
			MaxConcurrent:    10,
		}
	}

	return &Scheduler{
		orchestrator: orch,
		taskQueue:    queue,
		config:       cfg,
		verifier:     verifier,
		assignments:  make(map[string]string),
		agentLoads:   make(map[string]int),
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(s.config.ScheduleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.schedule(ctx)
		}
	}
}

func (s *Scheduler) schedule(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	agents := s.orchestrator.GetAvailableAgents()
	for _, agent := range agents {
		if s.agentLoads[agent.ID()] >= s.config.MaxConcurrent {
			continue
		}

		task, err := s.taskQueue.Dequeue(ctx, agent.Type())
		if err != nil {
			continue
		}

		s.assignTask(ctx, agent, task)
	}
}

func (s *Scheduler) assignTask(ctx context.Context, agent api.Agent, task *api.Task) {
	task.AssignedAgent = agent.ID()
	s.assignments[task.ID] = agent.ID()
	s.agentLoads[agent.ID()]++

	go s.executeTask(ctx, agent, task)
}

func (s *Scheduler) executeTask(ctx context.Context, agent api.Agent, apiTask *api.Task) {
	defer func() {
		s.mu.Lock()
		s.agentLoads[agent.ID()]--
		delete(s.assignments, apiTask.ID)
		s.mu.Unlock()
	}()

	agentResult, err := agent.Execute(ctx, apiTask)

	if err != nil {
		s.taskQueue.Fail(apiTask.ID, err)
		return
	}

	taskResult := &api.TaskResult{
		TaskID:      agentResult.TaskID,
		Success:     agentResult.Success,
		Output:      agentResult.Output,
		Artifacts:   agentResult.Artifacts,
		Error:       agentResult.Error,
		Metrics:     agentResult.Metrics,
		CompletedAt: agentResult.CompletedAt,
	}

	// Verify the result if a verifier is configured and the task has verification spec.
	if verifyErr := s.verifyTaskResult(ctx, apiTask, taskResult); verifyErr != nil {
		// Verification failed — if retries remain, re-enqueue; otherwise fail.
		if apiTask.RetryCount < apiTask.MaxRetries {
			apiTask.RetryCount++
			apiTask.Status = api.TaskStatusQueued
			if enqErr := s.taskQueue.Enqueue(ctx, apiTask); enqErr != nil {
				// Re-enqueue failed; mark the task as failed to avoid silent loss.
				s.taskQueue.Fail(apiTask.ID, fmt.Errorf("verification failed and re-enqueue failed: %v; original: %v", enqErr, verifyErr))
			}
			return
		}
		s.taskQueue.Fail(apiTask.ID, verifyErr)
		return
	}

	s.taskQueue.Complete(apiTask.ID, taskResult)
}

// verifyTaskResult runs the Verifier on a completed task. Returns nil if
// verification passes, is not configured, or the task has no verification spec.
func (s *Scheduler) verifyTaskResult(ctx context.Context, apiTask *api.Task, result *api.TaskResult) error {
	if s.verifier == nil {
		return nil
	}

	// Only verify tasks that carry a verification spec.
	if apiTask.Verification == nil || len(apiTask.Verification) == 0 {
		return nil
	}

	// Mark the task as verifying while we check.
	apiTask.Status = api.TaskStatusVerifying

	// Convert the api.Task to an internal task.Task so the verifier can operate.
	internalTask := task.APITaskToTask(apiTask)

	// Populate internal task output from the result.
	if result != nil && result.Output != nil {
		if outputMap, ok := result.Output.(map[string]any); ok {
			internalTask.Output = outputMap
		} else {
			internalTask.Output = map[string]any{"result": result.Output}
		}
	}

	vr := s.verifier.Verify(ctx, internalTask)
	if !vr.Valid {
		msgs := make([]string, 0, len(vr.Failures))
		for _, f := range vr.Failures {
			msgs = append(msgs, f.Message)
		}
		return fmt.Errorf("verification failed: %v", msgs)
	}

	return nil
}

func (s *Scheduler) GetAssignment(taskID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentID, ok := s.assignments[taskID]
	return agentID, ok
}

// GetTaskForAgent performs a reverse lookup on the assignments map, returning
// the taskID currently assigned to the given agentID. This is O(n) but the
// assignments map is small (bounded by MaxConcurrent * number of agents).
func (s *Scheduler) GetTaskForAgent(agentID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for taskID, assignedAgent := range s.assignments {
		if assignedAgent == agentID {
			return taskID, true
		}
	}
	return "", false
}

func (s *Scheduler) GetAgentLoad(agentID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.agentLoads[agentID]
}

func (s *Scheduler) ReassignTask(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldAgentID, ok := s.assignments[taskID]
	if !ok {
		return fmt.Errorf("task not assigned: %s", taskID)
	}

	// Decrement the old agent's load before removing the assignment.
	if oldAgentID != "" {
		s.agentLoads[oldAgentID]--
	}
	delete(s.assignments, taskID)

	task := s.taskQueue.GetTask(taskID)
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	agents := s.orchestrator.GetAvailableAgentsByType(task.AgentType)
	if len(agents) == 0 {
		return fmt.Errorf("no available agents for task: %s", taskID)
	}

	bestAgent := s.pickLeastLoaded(agents)
	task.AssignedAgent = ""
	task.Status = api.TaskStatusQueued

	s.assignTask(ctx, bestAgent, task)
	// Note: assignTask already increments agentLoads, so no second increment here.

	return nil
}

func (s *Scheduler) pickLeastLoaded(agents []api.Agent) api.Agent {
	bestAgent := agents[0]
	minLoad := s.agentLoads[bestAgent.ID()]

	for _, agent := range agents {
		if load := s.agentLoads[agent.ID()]; load < minLoad {
			minLoad = load
			bestAgent = agent
		}
	}

	return bestAgent
}

func (s *Scheduler) GetStats() (assigned int, totalLoad int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, load := range s.agentLoads {
		totalLoad += load
	}

	return len(s.assignments), totalLoad
}
