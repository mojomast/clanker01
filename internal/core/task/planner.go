package task

import (
	"context"
	"fmt"
	"time"
)

type Planner struct {
	templates   map[string]TaskTemplate
	constraints Constraints
	graph       *DependencyGraph
	decomposer  *Decomposer
	llmPlanner  LLMPlanner
}

type Objective struct {
	ID          string         `json:"id"`
	Description string         `json:"description"`
	Goals       []Goal         `json:"goals"`
	Constraints Constraints    `json:"constraints"`
	Context     map[string]any `json:"context"`
}

type Goal struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	SuccessCriteria string `json:"success_criteria"`
}

type Constraints struct {
	MaxParallelism int           `json:"max_parallelism"`
	MaxDuration    time.Duration `json:"max_duration"`
	ResourceLimit  ResourceLimit `json:"resource_limits"`
	AgentPool      []AgentID     `json:"agent_pool"`
}

type Plan struct {
	ID           string        `json:"id"`
	ObjectiveID  string        `json:"objective_id"`
	Tasks        []*Task       `json:"tasks"`
	CreatedAt    time.Time     `json:"created_at"`
	EstimatedDur time.Duration `json:"estimated_duration"`
}

type LLMPlanner interface {
	GeneratePlan(ctx context.Context, objective Objective) ([]*Task, error)
}

func NewPlanner(templates map[string]TaskTemplate, constraints Constraints, llmPlanner LLMPlanner) *Planner {
	return &Planner{
		templates:   templates,
		constraints: constraints,
		graph:       NewDependencyGraph(),
		decomposer:  NewDecomposer(DefaultRules, 5),
		llmPlanner:  llmPlanner,
	}
}

func (p *Planner) Plan(ctx context.Context, obj Objective) (*Plan, error) {
	tasks := make([]*Task, 0)

	if p.llmPlanner != nil {
		llmTasks, err := p.llmPlanner.GeneratePlan(ctx, obj)
		if err != nil {
			return nil, fmt.Errorf("LLM planning failed: %w", err)
		}
		tasks = append(tasks, llmTasks...)
	} else {
		for _, goal := range obj.Goals {
			decomposed, err := p.decomposeGoal(ctx, goal, obj.Context)
			if err != nil {
				return nil, fmt.Errorf("decompose goal %s: %w", goal.ID, err)
			}
			tasks = append(tasks, decomposed...)
		}
	}

	for _, task := range tasks {
		if task.CreatedAt.IsZero() {
			task.CreatedAt = time.Now()
		}
		task.UpdatedAt = time.Now()

		decomposedTasks, err := p.decomposer.Decompose(ctx, task)
		if err != nil {
			return nil, fmt.Errorf("decompose task %s: %w", task.ID, err)
		}
		if len(decomposedTasks) > 1 {
			tasks = replaceTask(tasks, task, decomposedTasks)
		}
	}

	if err := p.resolveDependencies(tasks); err != nil {
		return nil, fmt.Errorf("resolve dependencies: %w", err)
	}

	estimatedDur := p.estimateDuration(tasks)

	return &Plan{
		ID:           fmt.Sprintf("plan-%s-%d", obj.ID, time.Now().Unix()),
		ObjectiveID:  obj.ID,
		Tasks:        tasks,
		CreatedAt:    time.Now(),
		EstimatedDur: estimatedDur,
	}, nil
}

func (p *Planner) decomposeGoal(ctx context.Context, goal Goal, context map[string]any) ([]*Task, error) {
	task := &Task{
		ID:          TaskID(fmt.Sprintf("task-%s", goal.ID)),
		Name:        goal.Description,
		Description: goal.Description,
		Status:      StatusPending,
		Priority:    PriorityNormal,
		Kind:        KindDecision,
		Input: map[string]any{
			"goal_id":          goal.ID,
			"success_criteria": goal.SuccessCriteria,
			"context":          context,
		},
		Timeout:    5 * time.Minute,
		MaxRetries: 1,
	}

	return []*Task{task}, nil
}

func (p *Planner) resolveDependencies(tasks []*Task) error {
	graph := NewDependencyGraph()
	for _, task := range tasks {
		if err := graph.AddTask(task); err != nil {
			return fmt.Errorf("add task %s to graph: %w", task.ID, err)
		}
	}

	if err := graph.AssignLevels(); err != nil {
		return err
	}

	criticalPath, _ := graph.CriticalPath()
	if len(criticalPath) > 0 {
		for _, taskID := range criticalPath {
			if node := graph.GetNode(taskID); node != nil {
				node.Critical = true
			}
		}
	}

	p.graph = graph
	return nil
}

func (p *Planner) estimateDuration(tasks []*Task) time.Duration {
	var total time.Duration
	for _, task := range tasks {
		if task.Timeout > 0 {
			total += task.Timeout
		} else {
			total += 5 * time.Minute
		}
	}
	return total
}

func replaceTask(tasks []*Task, old *Task, newTasks []*Task) []*Task {
	result := make([]*Task, 0, len(tasks)-1+len(newTasks))
	for _, t := range tasks {
		if t.ID == old.ID {
			result = append(result, newTasks...)
		} else {
			result = append(result, t)
		}
	}
	return result
}
