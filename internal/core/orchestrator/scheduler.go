package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type Scheduler struct {
	orchestrator *Orchestrator
	taskQueue    *TaskQueue
	config       *SchedulerConfig

	mu          sync.RWMutex
	assignments map[string]string // taskID -> agentID
	agentLoads  map[string]int    // agentID -> load
}

type SchedulerConfig struct {
	ScheduleInterval time.Duration
	MaxConcurrent    int
}

func NewScheduler(orch *Orchestrator, queue *TaskQueue, cfg *SchedulerConfig) *Scheduler {
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

func (s *Scheduler) executeTask(ctx context.Context, agent api.Agent, task *api.Task) {
	defer func() {
		s.mu.Lock()
		s.agentLoads[agent.ID()]--
		delete(s.assignments, task.ID)
		s.mu.Unlock()
	}()

	agentResult, err := agent.Execute(ctx, task)

	if err != nil {
		s.taskQueue.Fail(task.ID, err)
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

	s.taskQueue.Complete(task.ID, taskResult)
}

func (s *Scheduler) GetAssignment(taskID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	agentID, ok := s.assignments[taskID]
	return agentID, ok
}

func (s *Scheduler) GetAgentLoad(agentID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.agentLoads[agentID]
}

func (s *Scheduler) ReassignTask(ctx context.Context, taskID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.assignments[taskID]
	if !ok {
		return fmt.Errorf("task not assigned: %s", taskID)
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
