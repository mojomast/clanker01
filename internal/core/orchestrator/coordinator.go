package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type Orchestrator struct {
	config       *OrchestratorConfig
	taskQueue    *TaskQueue
	scheduler    *Scheduler
	stateManager *StateManager
	recovery     *RecoveryManager
	conflictMgr  *ConflictManager
	agents       map[string]api.Agent

	mu      sync.RWMutex
	stopCh  chan struct{}
	running bool
}

type OrchestratorConfig struct {
	DataDir          string
	ScheduleInterval time.Duration
	MaxConcurrent    int
	MaxRetries       int
}

type ConflictManager struct {
	mu        sync.RWMutex
	conflicts map[string]*Conflict
}

type Conflict struct {
	ID               string
	Type             ConflictType
	Resource         string
	ConflictingTasks []string
	Resolution       *ConflictResolution
	DetectedAt       time.Time
}

type ConflictType string

const (
	ConflictTypeResource   ConflictType = "resource"
	ConflictTypeApproach   ConflictType = "approach"
	ConflictTypePriority   ConflictType = "priority"
	ConflictTypeDependency ConflictType = "dependency"
)

type ConflictResolution struct {
	Strategy string
	Action   string
	Resolved bool
}

func NewOrchestrator(cfg *OrchestratorConfig) (*Orchestrator, error) {
	taskQueue := NewTaskQueue()

	stateMgr, err := NewStateManager(cfg.DataDir)
	if err != nil {
		return nil, fmt.Errorf("create state manager: %w", err)
	}

	scheduler := NewScheduler(nil, taskQueue, &SchedulerConfig{
		ScheduleInterval: cfg.ScheduleInterval,
		MaxConcurrent:    cfg.MaxConcurrent,
	})

	recovery := NewRecoveryManager(nil, scheduler, &RecoveryConfig{
		MaxRetries: cfg.MaxRetries,
	})

	return &Orchestrator{
		config:       cfg,
		taskQueue:    taskQueue,
		scheduler:    scheduler,
		stateManager: stateMgr,
		recovery:     recovery,
		conflictMgr: &ConflictManager{
			conflicts: make(map[string]*Conflict),
		},
		agents: make(map[string]api.Agent),
		stopCh: make(chan struct{}),
	}, nil
}

func (o *Orchestrator) Start(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.running {
		return fmt.Errorf("orchestrator already running")
	}

	if _, err := o.stateManager.Load(); err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	o.scheduler.orchestrator = o
	o.recovery.orchestrator = o

	go o.scheduler.Start(ctx)

	o.running = true

	go o.monitorLoop(ctx)

	return nil
}

func (o *Orchestrator) Stop(ctx context.Context) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.running {
		return nil
	}

	close(o.stopCh)

	if err := o.stateManager.Save(); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	o.running = false

	return nil
}

func (o *Orchestrator) SubmitTask(ctx context.Context, task *api.Task) error {
	if task.ID == "" {
		task.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	if task.Status == "" {
		task.Status = api.TaskStatusPending
	}

	o.stateManager.mu.Lock()
	o.stateManager.currentState.Metrics.TasksSubmitted++
	o.stateManager.mu.Unlock()

	if task.ParentID != "" {
		task.Dependencies = append(task.Dependencies, task.ParentID)
	}

	// Pre-check for cycles by temporarily adding to the dependency graph.
	// Enqueue() also calls dependencies.Add(), so we remove first if a cycle
	// is detected, or let Enqueue handle the final addition.
	o.taskQueue.dependencies.Add(task)

	if o.taskQueue.dependencies.HasCycle() {
		if cycle, found := o.taskQueue.dependencies.DetectCycle(task.ID); found {
			o.taskQueue.dependencies.Remove(task.ID)

			o.conflictMgr.mu.Lock()
			conflict := &Conflict{
				ID:               fmt.Sprintf("conflict-%d", time.Now().UnixNano()),
				Type:             ConflictTypeDependency,
				ConflictingTasks: cycle,
				DetectedAt:       time.Now(),
			}
			o.conflictMgr.conflicts[conflict.ID] = conflict
			o.conflictMgr.mu.Unlock()

			return fmt.Errorf("circular dependency detected: %v", cycle)
		}
	}

	// Remove the temporary addition so Enqueue can add it cleanly (avoiding
	// double addition to the dependency graph's reverse-dependency map).
	o.taskQueue.dependencies.Remove(task.ID)

	return o.taskQueue.Enqueue(ctx, task)
}

func (o *Orchestrator) DecomposeTask(ctx context.Context, task *api.Task) ([]*api.Task, error) {
	var subtasks []*api.Task

	parts := splitTask(task)
	for i, part := range parts {
		subtask := &api.Task{
			ID:           fmt.Sprintf("%s-%d", task.ID, i),
			ParentID:     task.ID,
			Priority:     task.Priority,
			AgentType:    task.AgentType,
			Prompt:       part,
			Dependencies: []string{},
			Status:       api.TaskStatusPending,
			CreatedAt:    time.Now(),
			MaxRetries:   task.MaxRetries,
		}
		subtasks = append(subtasks, subtask)
	}

	return subtasks, nil
}

func (o *Orchestrator) RegisterAgent(agent api.Agent) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.agents[agent.ID()] = agent
}

func (o *Orchestrator) GetAvailableAgents() []api.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var available []api.Agent
	for _, agent := range o.agents {
		if agent.Status() == api.AgentStatusReady {
			available = append(available, agent)
		}
	}
	return available
}

func (o *Orchestrator) GetAvailableAgentsByType(agentType api.AgentType) []api.Agent {
	o.mu.RLock()
	defer o.mu.RUnlock()

	var available []api.Agent
	for _, agent := range o.agents {
		if agent.Type() == agentType && agent.Status() == api.AgentStatusReady {
			available = append(available, agent)
		}
	}
	return available
}

func (o *Orchestrator) GetTask(taskID string) *api.Task {
	return o.taskQueue.GetTask(taskID)
}

func (o *Orchestrator) GetTaskStatus(taskID string) api.TaskStatus {
	task := o.taskQueue.GetTask(taskID)
	if task == nil {
		return ""
	}
	return task.Status
}

func (o *Orchestrator) GetStats() *OrchestratorStats {
	pending, running, completed, failed := o.taskQueue.GetStats()

	state := o.stateManager.GetState()

	return &OrchestratorStats{
		PendingTasks:   pending,
		RunningTasks:   running,
		CompletedTasks: completed,
		FailedTasks:    failed,
		TasksSubmitted: state.Metrics.TasksSubmitted,
		TasksCompleted: state.Metrics.TasksCompleted,
		TasksFailed:    state.Metrics.TasksFailed,
		ActiveAgents:   len(state.Agents),
	}
}

func (o *Orchestrator) CreateCheckpoint(name string) (*Checkpoint, error) {
	o.syncState()
	return o.stateManager.CreateCheckpoint(name)
}

func (o *Orchestrator) RestoreCheckpoint(id string) error {
	_, err := o.stateManager.RestoreCheckpoint(id)
	return err
}

func (o *Orchestrator) ListCheckpoints() ([]*Checkpoint, error) {
	return o.stateManager.ListCheckpoints()
}

func (o *Orchestrator) monitorLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.syncState()
		}
	}
}

func (o *Orchestrator) syncState() {
	pending, running, completed, failed := o.taskQueue.GetStats()

	queueState := &QueueState{
		PendingCount:   pending,
		RunningCount:   running,
		CompletedCount: completed,
		FailedCount:    failed,
	}

	o.stateManager.UpdateQueueState(queueState)
}

type OrchestratorStats struct {
	PendingTasks   int
	RunningTasks   int
	CompletedTasks int
	FailedTasks    int
	TasksSubmitted int64
	TasksCompleted int64
	TasksFailed    int64
	ActiveAgents   int
}

func splitTask(task *api.Task) []string {
	parts := []string{task.Prompt}

	if len(parts[0]) > 500 {
		mid := len(parts[0]) / 2
		parts = []string{
			parts[0][:mid],
			parts[0][mid:],
		}
	}

	return parts
}
