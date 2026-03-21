package task

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDependencyGraph(t *testing.T) {
	graph := NewDependencyGraph()

	assert.NotNil(t, graph)
	assert.NotNil(t, graph.nodes)
	assert.NotNil(t, graph.edges)
	assert.NotNil(t, graph.reverse)
}

func TestDependencyGraph_AddTask(t *testing.T) {
	graph := NewDependencyGraph()

	task := &Task{
		ID:         "task-1",
		Name:       "Task 1",
		Status:     StatusPending,
		Timeout:    time.Minute,
		MaxRetries: 1,
	}

	err := graph.AddTask(task)
	require.NoError(t, err)

	assert.Equal(t, 1, graph.Count())
	assert.NotNil(t, graph.GetTask("task-1"))
}

func TestDependencyGraph_AddTask_Duplicate(t *testing.T) {
	graph := NewDependencyGraph()

	task := &Task{
		ID:      "task-1",
		Name:    "Task 1",
		Status:  StatusPending,
		Timeout: time.Minute,
	}

	err := graph.AddTask(task)
	require.NoError(t, err)

	err = graph.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestDependencyGraph_AddTask_WithDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	task1 := &Task{
		ID:      "task-1",
		Name:    "Task 1",
		Status:  StatusPending,
		Timeout: time.Minute,
	}

	task2 := &Task{
		ID:           "task-2",
		Name:         "Task 2",
		Status:       StatusPending,
		Dependencies: []TaskID{"task-1"},
		Timeout:      time.Minute,
	}

	err := graph.AddTask(task1)
	require.NoError(t, err)

	err = graph.AddTask(task2)
	require.NoError(t, err)

	assert.Equal(t, 2, graph.Count())
}

func TestDependencyGraph_AddTask_InvalidDependency(t *testing.T) {
	graph := NewDependencyGraph()

	task := &Task{
		ID:           "task-1",
		Name:         "Task 1",
		Status:       StatusPending,
		Dependencies: []TaskID{"non-existent"},
		Timeout:      time.Minute,
	}

	err := graph.AddTask(task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "dependency")
}

func TestDependencyGraph_TopologicalSort(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "t1", Status: StatusPending, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t3", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t4", Status: StatusPending, Dependencies: []TaskID{"t2", "t3"}, Timeout: time.Minute})

	sorted, err := graph.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, sorted, 4)
	assert.Equal(t, TaskID("t1"), sorted[0])
}

func TestDependencyGraph_TopologicalSort_Cycle(t *testing.T) {
	t.Skip("Cycle detection with current API requires manual graph manipulation")
}

func TestDependencyGraph_AssignLevels(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "t1", Status: StatusPending, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t3", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t4", Status: StatusPending, Dependencies: []TaskID{"t2", "t3"}, Timeout: time.Minute})

	err := graph.AssignLevels()
	require.NoError(t, err)

	assert.Equal(t, 0, graph.GetNode("t1").Level)
	assert.Equal(t, 1, graph.GetNode("t2").Level)
	assert.Equal(t, 1, graph.GetNode("t3").Level)
	assert.Equal(t, 2, graph.GetNode("t4").Level)
}

func TestDependencyGraph_GetReadyTasks(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "t1", Status: StatusPending, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t3", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t4", Status: StatusCompleted, Dependencies: []TaskID{"t2", "t3"}, Timeout: time.Minute})

	graph.GetNode("t4").Task.Status = StatusCompleted

	ready := graph.GetReadyTasks()
	assert.Len(t, ready, 1)
	assert.Equal(t, TaskID("t1"), ready[0])
}

func TestDependencyGraph_GetReadyTasks_AfterDependenciesComplete(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "t1", Status: StatusCompleted, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t3", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})

	ready := graph.GetReadyTasks()
	assert.Len(t, ready, 2)
}

func TestDependencyGraph_GetReadyTasks_WithPriority(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "t1", Status: StatusCompleted, Priority: PriorityNormal, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Dependencies: []TaskID{"t1"}, Priority: PriorityHigh, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t3", Status: StatusPending, Dependencies: []TaskID{"t1"}, Priority: PriorityLow, Timeout: time.Minute})

	ready := graph.GetReadyTasks()
	assert.Len(t, ready, 2)
	assert.Equal(t, TaskID("t2"), ready[0])
}

func TestDependencyGraph_CriticalPath(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "t1", Status: StatusPending, Timeout: 10 * time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: 20 * time.Minute})
	graph.AddTask(&Task{ID: "t3", Status: StatusPending, Dependencies: []TaskID{"t2"}, Timeout: 15 * time.Minute})
	graph.AddTask(&Task{ID: "t4", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: 5 * time.Minute})

	_, duration := graph.CriticalPath()
	assert.Equal(t, 45*time.Minute, duration)
}

func TestDependencyGraph_HasCycle_NoCycle(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "t1", Status: StatusPending, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Dependencies: []TaskID{"t1"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t3", Status: StatusPending, Dependencies: []TaskID{"t2"}, Timeout: time.Minute})

	assert.False(t, graph.HasCycle())
}

func TestDependencyGraph_HasCycle_WithCycle(t *testing.T) {
	t.Skip("Cycle detection with current API requires manual graph manipulation")
}

func TestDependencyGraph_GetNode(t *testing.T) {
	graph := NewDependencyGraph()

	task := &Task{
		ID:      "t1",
		Name:    "Task 1",
		Status:  StatusPending,
		Timeout: time.Minute,
	}
	graph.AddTask(task)

	node := graph.GetNode("t1")
	assert.NotNil(t, node)
	assert.Equal(t, task, node.Task)
	assert.Equal(t, 0, node.Level)
}

func TestDependencyGraph_GetNode_NotFound(t *testing.T) {
	graph := NewDependencyGraph()
	assert.Nil(t, graph.GetNode("non-existent"))
}

func TestDependencyGraph_GetTask(t *testing.T) {
	graph := NewDependencyGraph()

	task := &Task{
		ID:      "t1",
		Name:    "Task 1",
		Status:  StatusPending,
		Timeout: time.Minute,
	}
	graph.AddTask(task)

	retrieved := graph.GetTask("t1")
	assert.NotNil(t, retrieved)
	assert.Equal(t, TaskID("t1"), retrieved.ID)
}

func TestDependencyGraph_Count(t *testing.T) {
	graph := NewDependencyGraph()

	assert.Equal(t, 0, graph.Count())

	graph.AddTask(&Task{ID: "t1", Status: StatusPending, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "t2", Status: StatusPending, Timeout: time.Minute})

	assert.Equal(t, 2, graph.Count())
}

func TestDependencyGraph_TopologicalSort_Empty(t *testing.T) {
	graph := NewDependencyGraph()

	sorted, err := graph.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, sorted, 0)
}

func TestDependencyGraph_ComplexDAG(t *testing.T) {
	graph := NewDependencyGraph()

	graph.AddTask(&Task{ID: "a", Status: StatusPending, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "b", Status: StatusPending, Dependencies: []TaskID{"a"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "c", Status: StatusPending, Dependencies: []TaskID{"a"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "d", Status: StatusPending, Dependencies: []TaskID{"b", "c"}, Timeout: time.Minute})
	graph.AddTask(&Task{ID: "e", Status: StatusPending, Dependencies: []TaskID{"d"}, Timeout: time.Minute})

	sorted, err := graph.TopologicalSort()
	require.NoError(t, err)
	assert.Len(t, sorted, 5)

	indexMap := make(map[TaskID]int)
	for i, id := range sorted {
		indexMap[id] = i
	}

	assert.Less(t, indexMap["a"], indexMap["b"])
	assert.Less(t, indexMap["a"], indexMap["c"])
	assert.Less(t, indexMap["b"], indexMap["d"])
	assert.Less(t, indexMap["c"], indexMap["d"])
	assert.Less(t, indexMap["d"], indexMap["e"])
}
