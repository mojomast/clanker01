package task

import (
	"context"
	"fmt"
	"sort"
	"time"
)

type Decomposer struct {
	rules      []DecompositionRule
	complexity ComplexityAnalyzer
	maxDepth   int
}

func NewDecomposer(rules []DecompositionRule, maxDepth int) *Decomposer {
	return &Decomposer{
		rules:      rules,
		complexity: &DefaultComplexityAnalyzer{},
		maxDepth:   maxDepth,
	}
}

func (d *Decomposer) Decompose(ctx context.Context, task *Task) ([]*Task, error) {
	if d.maxDepth <= 0 {
		return []*Task{task}, nil
	}

	applicable := d.findApplicableRules(task)
	if len(applicable) == 0 {
		return []*Task{task}, nil
	}

	rule := applicable[0]
	subtasks, err := d.applyRule(ctx, task, rule)
	if err != nil {
		return nil, fmt.Errorf("apply rule %s: %w", rule.ID, err)
	}

	result := make([]*Task, 0, len(subtasks))
	for _, st := range subtasks {
		childDepth := d.maxDepth - 1
		subDecomposer := &Decomposer{
			rules:      d.rules,
			complexity: d.complexity,
			maxDepth:   childDepth,
		}
		decomposed, err := subDecomposer.Decompose(ctx, st)
		if err != nil {
			return nil, fmt.Errorf("recursive decomposition: %w", err)
		}
		result = append(result, decomposed...)
	}

	return result, nil
}

func (d *Decomposer) findApplicableRules(task *Task) []DecompositionRule {
	var matched []DecompositionRule
	for _, rule := range d.rules {
		if d.matchesCondition(task, rule.Condition) {
			matched = append(matched, rule)
		}
	}
	sort.Slice(matched, func(i, j int) bool {
		return matched[i].Priority > matched[j].Priority
	})
	return matched
}

func (d *Decomposer) matchesCondition(task *Task, cond Condition) bool {
	if cond.TaskKind != "" && task.Kind != cond.TaskKind {
		return false
	}

	complexity := d.complexity.Analyze(task)
	if cond.MinComplexity > 0 && complexity < cond.MinComplexity {
		return false
	}

	if cond.HasPattern != "" && task.Input != nil {
		if pattern, ok := task.Input["pattern"].(string); !ok || pattern != cond.HasPattern {
			return false
		}
	}

	if cond.InputSize > 0 && task.Input != nil {
		if items, ok := task.Input["items"].([]any); ok && len(items) < cond.InputSize {
			return false
		}
	}

	return true
}

func (d *Decomposer) applyRule(ctx context.Context, task *Task, rule DecompositionRule) ([]*Task, error) {
	switch rule.Action.Strategy {
	case StrategyParallel:
		return d.parallelDecompose(task, rule.Action.ChunkSize)
	case StrategySequential:
		return d.sequentialDecompose(task)
	case StrategyPipeline:
		return d.pipelineDecompose(task)
	case StrategyMapReduce:
		return d.mapReduceDecompose(task, rule.Action.ChunkSize)
	case StrategyDivide:
		return d.divideConquerDecompose(task)
	default:
		return []*Task{task}, nil
	}
}

func (d *Decomposer) parallelDecompose(task *Task, chunkSize int) ([]*Task, error) {
	if task.Input == nil {
		return []*Task{task}, nil
	}

	input, ok := task.Input["items"].([]any)
	if !ok || len(input) == 0 {
		return []*Task{task}, nil
	}

	chunks := chunkSlice(input, chunkSize)
	tasks := make([]*Task, len(chunks))

	for i, chunk := range chunks {
		tasks[i] = &Task{
			ID:           TaskID(fmt.Sprintf("%s-chunk-%d", task.ID, i)),
			Name:         fmt.Sprintf("%s (chunk %d)", task.Name, i),
			Description:  task.Description,
			Status:       StatusPending,
			Priority:     task.Priority,
			Kind:         task.Kind,
			Input:        map[string]any{"items": chunk, "chunk_index": i},
			Dependencies: task.Dependencies,
			Timeout:      task.Timeout,
			MaxRetries:   task.MaxRetries,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	return tasks, nil
}

func (d *Decomposer) sequentialDecompose(task *Task) ([]*Task, error) {
	if task.Input == nil {
		return []*Task{task}, nil
	}

	steps, ok := task.Input["steps"].([]any)
	if !ok || len(steps) == 0 {
		return []*Task{task}, nil
	}

	tasks := make([]*Task, len(steps))
	var prevID TaskID

	for i, step := range steps {
		stepMap, ok := step.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("step %d is not a map", i)
		}

		deps := []TaskID{}
		if prevID != "" {
			deps = []TaskID{prevID}
		}

		description, _ := stepMap["description"].(string)
		input, _ := stepMap["input"].(map[string]any)

		tasks[i] = &Task{
			ID:           TaskID(fmt.Sprintf("%s-step-%d", task.ID, i)),
			Name:         fmt.Sprintf("%s (step %d)", task.Name, i+1),
			Description:  description,
			Status:       StatusPending,
			Priority:     task.Priority,
			Kind:         task.Kind,
			Input:        input,
			Dependencies: deps,
			Timeout:      task.Timeout,
			MaxRetries:   task.MaxRetries,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		prevID = tasks[i].ID
	}

	return tasks, nil
}

func (d *Decomposer) pipelineDecompose(task *Task) ([]*Task, error) {
	if task.Input == nil {
		return []*Task{task}, nil
	}

	stages, ok := task.Input["stages"].([]any)
	if !ok || len(stages) == 0 {
		return []*Task{task}, nil
	}

	tasks := make([]*Task, len(stages))
	var prevID TaskID

	for i, stage := range stages {
		stageMap, ok := stage.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("stage %d is not a map", i)
		}

		deps := []TaskID{}
		if prevID != "" {
			deps = []TaskID{prevID}
		}

		name, _ := stageMap["name"].(string)
		input, _ := stageMap["input"].(map[string]any)
		kind, _ := stageMap["kind"].(TaskKind)

		if kind == "" {
			kind = task.Kind
		}

		tasks[i] = &Task{
			ID:           TaskID(fmt.Sprintf("%s-stage-%d", task.ID, i)),
			Name:         name,
			Description:  fmt.Sprintf("Pipeline stage %d: %s", i, name),
			Status:       StatusPending,
			Priority:     task.Priority,
			Kind:         kind,
			Input:        input,
			Dependencies: deps,
			Timeout:      task.Timeout,
			MaxRetries:   task.MaxRetries,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		prevID = tasks[i].ID
	}

	return tasks, nil
}

func (d *Decomposer) mapReduceDecompose(task *Task, chunkSize int) ([]*Task, error) {
	if task.Input == nil {
		return []*Task{task}, nil
	}

	input, ok := task.Input["items"].([]any)
	if !ok || len(input) == 0 {
		return []*Task{task}, nil
	}

	chunks := chunkSlice(input, chunkSize)
	mapTasks := make([]*Task, len(chunks))

	for i, chunk := range chunks {
		mapTasks[i] = &Task{
			ID:          TaskID(fmt.Sprintf("%s-map-%d", task.ID, i)),
			Name:        fmt.Sprintf("%s (map %d)", task.Name, i),
			Description: fmt.Sprintf("Map phase for chunk %d", i),
			Status:      StatusPending,
			Priority:    task.Priority,
			Kind:        KindCompute,
			Input: map[string]any{
				"items":       chunk,
				"mapper":      task.Input["mapper"],
				"chunk_index": i,
			},
			Dependencies: task.Dependencies,
			Timeout:      task.Timeout,
			MaxRetries:   task.MaxRetries,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	reduceTask := &Task{
		ID:          TaskID(fmt.Sprintf("%s-reduce", task.ID)),
		Name:        fmt.Sprintf("%s (reduce)", task.Name),
		Description: "Reduce phase: aggregate map results",
		Status:      StatusPending,
		Priority:    task.Priority,
		Kind:        KindAggregate,
		Input: map[string]any{
			"reducer":   task.Input["reducer"],
			"map_tasks": len(mapTasks),
		},
		Dependencies: getTaskIDs(mapTasks),
		Timeout:      task.Timeout,
		MaxRetries:   task.MaxRetries,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	allTasks := append(mapTasks, reduceTask)
	return allTasks, nil
}

func (d *Decomposer) divideConquerDecompose(task *Task) ([]*Task, error) {
	if task.Input == nil {
		return []*Task{task}, nil
	}

	input, ok := task.Input["items"].([]any)
	if !ok || len(input) <= 2 {
		return []*Task{task}, nil
	}

	mid := len(input) / 2
	left := input[:mid]
	right := input[mid:]

	leftTask := &Task{
		ID:           TaskID(fmt.Sprintf("%s-divide-left", task.ID)),
		Name:         fmt.Sprintf("%s (divide left)", task.Name),
		Description:  "Divide and conquer: left half",
		Status:       StatusPending,
		Priority:     task.Priority,
		Kind:         task.Kind,
		Input:        map[string]any{"items": left},
		Dependencies: task.Dependencies,
		Timeout:      task.Timeout,
		MaxRetries:   task.MaxRetries,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	rightTask := &Task{
		ID:           TaskID(fmt.Sprintf("%s-divide-right", task.ID)),
		Name:         fmt.Sprintf("%s (divide right)", task.Name),
		Description:  "Divide and conquer: right half",
		Status:       StatusPending,
		Priority:     task.Priority,
		Kind:         task.Kind,
		Input:        map[string]any{"items": right},
		Dependencies: task.Dependencies,
		Timeout:      task.Timeout,
		MaxRetries:   task.MaxRetries,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	combineTask := &Task{
		ID:           TaskID(fmt.Sprintf("%s-combine", task.ID)),
		Name:         fmt.Sprintf("%s (combine)", task.Name),
		Description:  "Divide and conquer: combine results",
		Status:       StatusPending,
		Priority:     task.Priority,
		Kind:         KindAggregate,
		Input:        map[string]any{},
		Dependencies: []TaskID{leftTask.ID, rightTask.ID},
		Timeout:      task.Timeout,
		MaxRetries:   task.MaxRetries,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	return []*Task{leftTask, rightTask, combineTask}, nil
}

func chunkSlice(input []any, chunkSize int) [][]any {
	if chunkSize <= 0 {
		chunkSize = 100
	}

	var chunks [][]any
	for i := 0; i < len(input); i += chunkSize {
		end := i + chunkSize
		if end > len(input) {
			end = len(input)
		}
		chunks = append(chunks, input[i:end])
	}
	return chunks
}

func getTaskIDs(tasks []*Task) []TaskID {
	ids := make([]TaskID, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return ids
}
