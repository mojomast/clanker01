package task

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockLLMPlanner struct {
	Tasks []*Task
	Err   error
}

func (m *MockLLMPlanner) GeneratePlan(ctx context.Context, objective Objective) ([]*Task, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Tasks, nil
}

func TestNewPlanner(t *testing.T) {
	templates := map[string]TaskTemplate{}
	constraints := Constraints{}
	llmPlanner := &MockLLMPlanner{}

	planner := NewPlanner(templates, constraints, llmPlanner)

	assert.NotNil(t, planner)
	assert.NotNil(t, planner.templates)
	assert.NotNil(t, planner.graph)
	assert.NotNil(t, planner.decomposer)
	assert.Equal(t, llmPlanner, planner.llmPlanner)
}

func TestPlanner_Plan_WithLLM(t *testing.T) {
	tasks := []*Task{
		{
			ID:          "task-1",
			Name:        "Test Task",
			Description: "Test Description",
			Status:      StatusPending,
			Priority:    PriorityNormal,
			Kind:        KindCompute,
			Input:       map[string]any{"key": "value"},
			Timeout:     time.Minute,
			MaxRetries:  1,
		},
	}

	llmPlanner := &MockLLMPlanner{Tasks: tasks}
	planner := NewPlanner(nil, Constraints{}, llmPlanner)

	objective := Objective{
		ID:          "obj-1",
		Description: "Test Objective",
		Goals: []Goal{
			{ID: "g1", Description: "Goal 1"},
		},
	}

	plan, err := planner.Plan(context.Background(), objective)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, "obj-1", plan.ObjectiveID)
	assert.Len(t, plan.Tasks, 1)
	assert.Equal(t, TaskID("task-1"), plan.Tasks[0].ID)
}

func TestPlanner_Plan_WithoutLLM(t *testing.T) {
	planner := NewPlanner(nil, Constraints{}, nil)

	objective := Objective{
		ID:          "obj-1",
		Description: "Test Objective",
		Goals: []Goal{
			{ID: "g1", Description: "Goal 1", SuccessCriteria: "Done"},
		},
	}

	plan, err := planner.Plan(context.Background(), objective)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.Equal(t, "obj-1", plan.ObjectiveID)
	assert.Len(t, plan.Tasks, 1)
	assert.Equal(t, TaskID("task-g1"), plan.Tasks[0].ID)
}

func TestPlanner_Plan_WithDecomposition(t *testing.T) {
	tasks := []*Task{
		{
			ID:          "task-1",
			Name:        "Large Compute Task",
			Description: "Process large dataset",
			Status:      StatusPending,
			Priority:    PriorityHigh,
			Kind:        KindCompute,
			Input: map[string]any{
				"items": make([]any, 1500),
			},
			Timeout:    10 * time.Minute,
			MaxRetries: 2,
		},
	}

	llmPlanner := &MockLLMPlanner{Tasks: tasks}
	planner := NewPlanner(nil, Constraints{}, llmPlanner)

	objective := Objective{
		ID:          "obj-1",
		Description: "Test Objective",
	}

	plan, err := planner.Plan(context.Background(), objective)
	require.NoError(t, err)
	assert.NotNil(t, plan)
	assert.True(t, len(plan.Tasks) > 1, "Task should be decomposed")
}

func TestPlanner_Plan_LLMError(t *testing.T) {
	llmPlanner := &MockLLMPlanner{Err: assert.AnError}
	planner := NewPlanner(nil, Constraints{}, llmPlanner)

	objective := Objective{
		ID:          "obj-1",
		Description: "Test Objective",
		Goals: []Goal{
			{ID: "g1", Description: "Goal 1"},
		},
	}

	_, err := planner.Plan(context.Background(), objective)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "LLM planning failed")
}

func TestPlanner_ResolveDependencies(t *testing.T) {
	tasks := []*Task{
		{
			ID:      "task-1",
			Name:    "Task 1",
			Status:  StatusPending,
			Timeout: time.Minute,
		},
		{
			ID:           "task-2",
			Name:         "Task 2",
			Status:       StatusPending,
			Dependencies: []TaskID{"task-1"},
			Timeout:      time.Minute,
		},
	}

	planner := NewPlanner(nil, Constraints{}, nil)
	err := planner.resolveDependencies(tasks)
	require.NoError(t, err)

	assert.False(t, planner.graph.HasCycle())
	assert.NotNil(t, planner.graph.GetTask("task-1"))
	assert.NotNil(t, planner.graph.GetTask("task-2"))
}

func TestPlanner_EstimateDuration(t *testing.T) {
	tasks := []*Task{
		{ID: "t1", Timeout: time.Minute},
		{ID: "t2", Timeout: 2 * time.Minute},
		{ID: "t3", Timeout: 0},
	}

	planner := NewPlanner(nil, Constraints{}, nil)
	duration := planner.estimateDuration(tasks)

	expected := 3*time.Minute + 5*time.Minute
	assert.Equal(t, expected, duration)
}

func TestPlanner_DecomposeGoal(t *testing.T) {
	planner := NewPlanner(nil, Constraints{}, nil)

	goal := Goal{
		ID:              "g1",
		Description:     "Test Goal",
		SuccessCriteria: "Done",
	}
	ctx := map[string]any{"key": "value"}

	tasks, err := planner.decomposeGoal(context.Background(), goal, ctx)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, TaskID("task-g1"), tasks[0].ID)
	assert.Equal(t, "Test Goal", tasks[0].Description)
}

func TestReplaceTask(t *testing.T) {
	old := &Task{ID: "old"}
	newTasks := []*Task{{ID: "new1"}, {ID: "new2"}}
	other := &Task{ID: "other"}

	tasks := []*Task{old, other}

	result := replaceTask(tasks, old, newTasks)

	assert.Len(t, result, 3)
	assert.Equal(t, TaskID("new1"), result[0].ID)
	assert.Equal(t, TaskID("new2"), result[1].ID)
	assert.Equal(t, TaskID("other"), result[2].ID)
}
