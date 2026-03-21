package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDecomposer(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	assert.NotNil(t, decomposer)
	assert.NotNil(t, decomposer.rules)
	assert.NotNil(t, decomposer.complexity)
	assert.Equal(t, 5, decomposer.maxDepth)
}

func TestDecomposer_Decompose_NoApplicableRules(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	task := &Task{
		ID:          "task-1",
		Name:        "Small Task",
		Description: "Small task",
		Status:      StatusPending,
		Kind:        KindIO,
		Input: map[string]any{
			"items": make([]any, 10),
		},
		Timeout: time.Minute,
	}

	result, err := decomposer.Decompose(context.Background(), task)
	require.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, task.ID, result[0].ID)
}

func TestDecomposer_Decompose_MaxDepthExceeded(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 0)

	task := &Task{
		ID:          "task-1",
		Name:        "Task",
		Description: "Task",
		Status:      StatusPending,
		Timeout:     time.Minute,
	}

	result, err := decomposer.Decompose(context.Background(), task)
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestDecomposer_ParallelDecompose(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	items := make([]any, 1200)
	for i := 0; i < 1200; i++ {
		items[i] = i
	}

	task := &Task{
		ID:          "task-1",
		Name:        "Large Compute Task",
		Description: "Process items",
		Status:      StatusPending,
		Kind:        KindCompute,
		Input: map[string]any{
			"items": items,
		},
		Timeout:    10 * time.Minute,
		MaxRetries: 2,
	}

	result, err := decomposer.Decompose(context.Background(), task)
	require.NoError(t, err)
	assert.True(t, len(result) > 1, "Should decompose into multiple tasks")
	if len(result) > 0 {
		assert.Contains(t, result[0].ID, "chunk")
	}
}

func TestDecomposer_SequentialDecompose(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	steps := []any{
		map[string]any{"description": "Step 1", "input": map[string]any{"step": 1}},
		map[string]any{"description": "Step 2", "input": map[string]any{"step": 2}},
		map[string]any{"description": "Step 3", "input": map[string]any{"step": 3}},
	}

	task := &Task{
		ID:          "task-1",
		Name:        "Sequential Task",
		Description: "Sequential execution",
		Status:      StatusPending,
		Kind:        KindCompute,
		Input: map[string]any{
			"steps":   steps,
			"pattern": "strict_order",
		},
		Timeout: time.Minute,
	}

	result, err := decomposer.Decompose(context.Background(), task)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Contains(t, result[0].ID, "step-0")
	assert.Contains(t, result[1].ID, "step-1")
	assert.Contains(t, result[2].ID, "step-2")
}

func TestDecomposer_PipelineDecompose(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	stages := []any{
		map[string]any{"name": "Extract", "input": map[string]any{}},
		map[string]any{"name": "Transform", "input": map[string]any{}},
		map[string]any{"name": "Load", "input": map[string]any{}},
	}

	task := &Task{
		ID:          "task-1",
		Name:        "Pipeline Task",
		Description: "Pipeline execution",
		Status:      StatusPending,
		Kind:        KindIO,
		Input: map[string]any{
			"stages":  stages,
			"pattern": "network_required",
		},
		Timeout: time.Minute,
	}

	result, err := decomposer.Decompose(context.Background(), task)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Contains(t, result[0].ID, "stage-0")
	assert.Contains(t, result[1].ID, "stage-1")
	assert.Contains(t, result[2].ID, "stage-2")
}

func TestDecomposer_MapReduceDecompose(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	items := make([]any, 600)
	for i := 0; i < 600; i++ {
		items[i] = i
	}

	task := &Task{
		ID:          "task-1",
		Name:        "MapReduce Task",
		Description: "MapReduce execution",
		Status:      StatusPending,
		Kind:        KindAggregate,
		Input: map[string]any{
			"items":   items,
			"mapper":  "word-split",
			"reducer": "sum",
		},
		Timeout: time.Minute,
	}

	result, err := decomposer.Decompose(context.Background(), task)
	require.NoError(t, err)
	assert.True(t, len(result) > 2, "Should create map and reduce tasks")

	mapCount := 0
	reduceCount := 0
	for _, t := range result {
		if t.Kind == KindCompute {
			mapCount++
		} else if t.Kind == KindAggregate {
			reduceCount++
		}
	}
	assert.Greater(t, mapCount, 0, "Should have map tasks")
	assert.Equal(t, 1, reduceCount, "Should have exactly one reduce task")
}

func TestDecomposer_DivideConquerDecompose(t *testing.T) {
	rules := []DecompositionRule{
		{
			ID:   "divide-conquer",
			Name: "Divide and Conquer",
			Condition: Condition{
				TaskKind: KindCompute,
			},
			Action: DecomposeAction{
				Strategy: StrategyDivide,
			},
			Priority: 10,
		},
	}

	decomposer := NewDecomposer(rules, 1)

	items := make([]any, 100)
	for i := 0; i < 100; i++ {
		items[i] = i
	}

	task := &Task{
		ID:          "task-1",
		Name:        "Divide Task",
		Description: "Divide and conquer",
		Status:      StatusPending,
		Kind:        KindCompute,
		Input: map[string]any{
			"items": items,
		},
		Timeout: time.Minute,
	}

	result, err := decomposer.Decompose(context.Background(), task)
	require.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Contains(t, result[0].ID, "divide-left")
	assert.Contains(t, result[1].ID, "divide-right")
	assert.Contains(t, result[2].ID, "combine")
}

func TestDecomposer_FindApplicableRules(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	task := &Task{
		ID:          "task-1",
		Name:        "Large Compute",
		Description: "Large compute task",
		Status:      StatusPending,
		Kind:        KindCompute,
		Input: map[string]any{
			"items": make([]any, 1500),
		},
	}

	applicable := decomposer.findApplicableRules(task)
	assert.NotEmpty(t, applicable)
	assert.Equal(t, 10, applicable[0].Priority)
}

func TestDecomposer_MatchesCondition(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	task := &Task{
		ID:     "task-1",
		Name:   "Test Task",
		Status: StatusPending,
		Kind:   KindCompute,
		Input: map[string]any{
			"items":   make([]any, 1000),
			"pattern": "strict_order",
		},
	}

	cond := Condition{
		TaskKind:      KindCompute,
		MinComplexity: 0.5,
		InputSize:     100,
	}

	assert.True(t, decomposer.matchesCondition(task, cond))
}

func TestDecomposer_MatchesCondition_NoMatch(t *testing.T) {
	decomposer := NewDecomposer(DefaultRules, 5)

	task := &Task{
		ID:     "task-1",
		Name:   "Test Task",
		Status: StatusPending,
		Kind:   KindIO,
		Input: map[string]any{
			"items": make([]any, 10),
		},
	}

	cond := Condition{
		TaskKind:      KindCompute,
		MinComplexity: 0.8,
		InputSize:     1000,
	}

	assert.False(t, decomposer.matchesCondition(task, cond))
}

func TestChunkSlice(t *testing.T) {
	items := make([]any, 100)
	for i := 0; i < 100; i++ {
		items[i] = i
	}

	chunks := chunkSlice(items, 30)
	assert.Len(t, chunks, 4)
	assert.Len(t, chunks[0], 30)
	assert.Len(t, chunks[1], 30)
	assert.Len(t, chunks[2], 30)
	assert.Len(t, chunks[3], 10)
}

func TestGetTaskIDs(t *testing.T) {
	tasks := []*Task{
		{ID: "t1"},
		{ID: "t2"},
		{ID: "t3"},
	}

	ids := getTaskIDs(tasks)
	assert.Len(t, ids, 3)
	assert.Equal(t, TaskID("t1"), ids[0])
	assert.Equal(t, TaskID("t2"), ids[1])
	assert.Equal(t, TaskID("t3"), ids[2])
}

func TestDefaultComplexityAnalyzer(t *testing.T) {
	analyzer := &DefaultComplexityAnalyzer{}

	task := &Task{
		ID:    "task-1",
		Input: map[string]any{"key": "value"},
	}

	complexity := analyzer.Analyze(task)
	assert.Greater(t, complexity, 0.0)
	assert.LessOrEqual(t, complexity, 1.0)
}

func TestDefaultComplexityAnalyzer_WithItems(t *testing.T) {
	analyzer := &DefaultComplexityAnalyzer{}

	task := &Task{
		ID: "task-1",
		Input: map[string]any{
			"items": make([]any, 100),
		},
	}

	complexity := analyzer.Analyze(task)
	assert.Greater(t, complexity, 0.0)
	assert.Greater(t, complexity, 0.5)
}
