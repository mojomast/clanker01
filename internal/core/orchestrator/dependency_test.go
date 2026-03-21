package orchestrator

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewDependencyGraph(t *testing.T) {
	g := NewDependencyGraph()
	assert.NotNil(t, g)
	assert.NotNil(t, g.tasks)
	assert.NotNil(t, g.deps)
	assert.NotNil(t, g.rdeps)
}

func TestDependencyGraph_Add(t *testing.T) {
	g := NewDependencyGraph()

	task := &api.Task{
		ID:           "task-1",
		Priority:     10,
		Prompt:       "Test task",
		Dependencies: []string{"dep-1", "dep-2"},
	}

	g.Add(task)

	assert.Equal(t, task, g.tasks["task-1"])
	assert.Equal(t, []string{"dep-1", "dep-2"}, g.deps["task-1"])
	assert.Contains(t, g.rdeps["dep-1"], "task-1")
	assert.Contains(t, g.rdeps["dep-2"], "task-1")
}

func TestDependencyGraph_IsReady(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:     "task-1",
		Prompt: "First task",
		Status: api.TaskStatusCompleted,
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}

	g.Add(task1)
	g.Add(task2)

	assert.True(t, g.IsReady("task-1"))
	assert.True(t, g.IsReady("task-2"))
}

func TestDependencyGraph_IsReady_NotReady(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:     "task-1",
		Prompt: "First task",
		Status: api.TaskStatusPending,
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}

	g.Add(task1)
	g.Add(task2)

	assert.False(t, g.IsReady("task-2"))
}

func TestDependencyGraph_GetTask(t *testing.T) {
	g := NewDependencyGraph()

	task := &api.Task{
		ID:     "task-1",
		Prompt: "Test task",
	}

	g.Add(task)

	retrieved := g.GetTask("task-1")
	assert.Equal(t, task, retrieved)
}

func TestDependencyGraph_GetDependents(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:     "task-1",
		Prompt: "First task",
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}
	task3 := &api.Task{
		ID:           "task-3",
		Prompt:       "Third task",
		Dependencies: []string{"task-1"},
	}

	g.Add(task1)
	g.Add(task2)
	g.Add(task3)

	dependents := g.GetDependents("task-1")
	assert.Len(t, dependents, 2)
	assert.Contains(t, dependents, "task-2")
	assert.Contains(t, dependents, "task-3")
}

func TestDependencyGraph_DetectCycle_NoCycle(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:     "task-1",
		Prompt: "First task",
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}
	task3 := &api.Task{
		ID:           "task-3",
		Prompt:       "Third task",
		Dependencies: []string{"task-2"},
	}

	g.Add(task1)
	g.Add(task2)
	g.Add(task3)

	cycle, found := g.DetectCycle("task-3")
	assert.False(t, found)
	assert.Nil(t, cycle)
}

func TestDependencyGraph_DetectCycle_WithCycle(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:           "task-1",
		Prompt:       "First task",
		Dependencies: []string{"task-2"},
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}

	g.Add(task1)
	g.Add(task2)

	cycle, found := g.DetectCycle("task-1")
	assert.True(t, found)
	assert.NotNil(t, cycle)
	assert.Contains(t, cycle, "task-1")
	assert.Contains(t, cycle, "task-2")
}

func TestDependencyGraph_TopologicalOrder(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:     "task-1",
		Prompt: "First task",
		Status: api.TaskStatusCompleted,
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}
	task3 := &api.Task{
		ID:           "task-3",
		Prompt:       "Third task",
		Dependencies: []string{"task-2"},
	}

	g.Add(task1)
	g.Add(task2)
	g.Add(task3)

	order := g.TopologicalOrder()
	assert.Len(t, order, 3)

	task1Index := -1
	task2Index := -1
	task3Index := -1

	for i, id := range order {
		if id == "task-1" {
			task1Index = i
		} else if id == "task-2" {
			task2Index = i
		} else if id == "task-3" {
			task3Index = i
		}
	}

	assert.Less(t, task1Index, task2Index)
	assert.Less(t, task2Index, task3Index)
}

func TestDependencyGraph_TopologicalOrder_WithCycle(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:           "task-1",
		Prompt:       "First task",
		Dependencies: []string{"task-2"},
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}

	g.Add(task1)
	g.Add(task2)

	order := g.TopologicalOrder()
	assert.Nil(t, order)
}

func TestDependencyGraph_ExecutionBatches(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:     "task-1",
		Prompt: "First task",
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}
	task3 := &api.Task{
		ID:           "task-3",
		Prompt:       "Third task",
		Dependencies: []string{"task-1"},
	}
	task4 := &api.Task{
		ID:           "task-4",
		Prompt:       "Fourth task",
		Dependencies: []string{"task-2", "task-3"},
	}

	g.Add(task1)
	g.Add(task2)
	g.Add(task3)
	g.Add(task4)

	batches := g.ExecutionBatches()

	assert.Len(t, batches, 3)
	assert.Len(t, batches[0], 1)
	assert.Contains(t, batches[0], "task-1")
	assert.Len(t, batches[1], 2)
	assert.Contains(t, batches[1], "task-2")
	assert.Contains(t, batches[1], "task-3")
	assert.Len(t, batches[2], 1)
	assert.Contains(t, batches[2], "task-4")
}

func TestDependencyGraph_HasCycle(t *testing.T) {
	g := NewDependencyGraph()

	task1 := &api.Task{
		ID:     "task-1",
		Prompt: "First task",
	}
	task2 := &api.Task{
		ID:           "task-2",
		Prompt:       "Second task",
		Dependencies: []string{"task-1"},
	}

	g.Add(task1)
	g.Add(task2)

	assert.False(t, g.HasCycle())

	g.Remove("task-1")
	task1.Dependencies = []string{"task-2"}
	g.Add(task1)

	assert.True(t, g.HasCycle())
}
