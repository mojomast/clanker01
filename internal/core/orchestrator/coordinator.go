package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/internal/core/task"
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
	planner      *task.Planner
	verifier     *task.Verifier

	mu      sync.RWMutex
	stopCh  chan struct{}
	running bool
}

type OrchestratorConfig struct {
	DataDir          string
	ScheduleInterval time.Duration
	MaxConcurrent    int
	MaxRetries       int

	// PlannerTemplates are task templates for the planner.
	PlannerTemplates map[string]task.TaskTemplate
	// PlannerConstraints control task planning limits.
	PlannerConstraints task.Constraints
	// LLMPlanner is an optional LLM-based planner. May be nil.
	LLMPlanner task.LLMPlanner
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

	// Create the task planner with configured templates, constraints, and LLM planner.
	planner := task.NewPlanner(cfg.PlannerTemplates, cfg.PlannerConstraints, cfg.LLMPlanner)

	// Create the task verifier with default settings (no custom checkers).
	verifier := task.NewVerifier(nil, nil)

	scheduler := NewScheduler(nil, taskQueue, &SchedulerConfig{
		ScheduleInterval: cfg.ScheduleInterval,
		MaxConcurrent:    cfg.MaxConcurrent,
	}, verifier)

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
		agents:   make(map[string]api.Agent),
		planner:  planner,
		verifier: verifier,
		stopCh:   make(chan struct{}),
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

func (o *Orchestrator) SubmitTask(ctx context.Context, t *api.Task) error {
	if t.ID == "" {
		t.ID = fmt.Sprintf("task-%d", time.Now().UnixNano())
	}

	if t.CreatedAt.IsZero() {
		t.CreatedAt = time.Now()
	}

	if t.Status == "" {
		t.Status = api.TaskStatusPending
	}

	o.stateManager.mu.Lock()
	o.stateManager.currentState.Metrics.TasksSubmitted++
	o.stateManager.mu.Unlock()

	if t.ParentID != "" {
		t.Dependencies = append(t.Dependencies, t.ParentID)
	}

	// If the task is complex, decompose it via the planner and submit subtasks.
	if o.shouldPlan(t) {
		subtasks, err := o.planTask(ctx, t)
		if err == nil && len(subtasks) > 1 {
			// Mark the parent as completed (it is now represented by subtasks).
			t.Status = api.TaskStatusCompleted
			for _, st := range subtasks {
				if err := o.SubmitTask(ctx, st); err != nil {
					return fmt.Errorf("submit planned subtask %s: %w", st.ID, err)
				}
			}
			return nil
		}
		// On planning error or single subtask, fall through to normal queueing.
	}

	// Pre-check for cycles by temporarily adding to the dependency graph.
	// Enqueue() also calls dependencies.Add(), so we remove first if a cycle
	// is detected, or let Enqueue handle the final addition.
	o.taskQueue.dependencies.Add(t)

	if o.taskQueue.dependencies.HasCycle() {
		if cycle, found := o.taskQueue.dependencies.DetectCycle(t.ID); found {
			o.taskQueue.dependencies.Remove(t.ID)

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
	o.taskQueue.dependencies.Remove(t.ID)

	return o.taskQueue.Enqueue(ctx, t)
}

// DecomposeTask breaks a task into subtasks. It tries the Planner first; if
// planning fails or returns a single task it falls back to the naive splitter.
func (o *Orchestrator) DecomposeTask(ctx context.Context, t *api.Task) ([]*api.Task, error) {
	// Try the planner first.
	if o.planner != nil {
		objective := task.ObjectiveFromAPITask(t)
		plan, err := o.planner.Plan(ctx, objective)
		if err == nil && plan != nil && len(plan.Tasks) > 0 {
			subtasks := task.TasksToAPITasks(plan.Tasks, t.ID, t.AgentType)
			if len(subtasks) > 1 {
				return subtasks, nil
			}
		}
		// On error or single-task plan, fall through to naive decomposition.
	}

	return naiveDecomposeTask(t), nil
}

// shouldPlan returns true if a task should be decomposed via the Planner.
func (o *Orchestrator) shouldPlan(t *api.Task) bool {
	if o.planner == nil {
		return false
	}
	// Explicitly marked complex.
	if t.IsComplex {
		return true
	}
	// Multiple requirements suggest a multi-step task.
	if len(t.Requirements) > 1 {
		return true
	}
	// Very long prompts are likely complex.
	if len(t.Prompt) > 500 {
		return true
	}
	return false
}

// planTask uses the Planner to decompose a complex api.Task into subtasks.
func (o *Orchestrator) planTask(ctx context.Context, t *api.Task) ([]*api.Task, error) {
	objective := task.ObjectiveFromAPITask(t)
	plan, err := o.planner.Plan(ctx, objective)
	if err != nil {
		return nil, fmt.Errorf("plan task %s: %w", t.ID, err)
	}
	if plan == nil || len(plan.Tasks) == 0 {
		return nil, fmt.Errorf("planner returned empty plan for task %s", t.ID)
	}
	return task.TasksToAPITasks(plan.Tasks, t.ID, t.AgentType), nil
}

// naiveDecomposeTask is the fallback decomposition that splits a task's prompt.
func naiveDecomposeTask(t *api.Task) []*api.Task {
	var subtasks []*api.Task

	parts := splitTask(t)
	for i, part := range parts {
		subtask := &api.Task{
			ID:           fmt.Sprintf("%s-%d", t.ID, i),
			ParentID:     t.ID,
			Priority:     t.Priority,
			AgentType:    t.AgentType,
			Prompt:       part,
			Dependencies: []string{},
			Status:       api.TaskStatusPending,
			CreatedAt:    time.Now(),
			MaxRetries:   t.MaxRetries,
		}
		subtasks = append(subtasks, subtask)
	}

	return subtasks
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
