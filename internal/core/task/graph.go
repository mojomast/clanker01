package task

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

type DependencyGraph struct {
	nodes   map[TaskID]*TaskNode
	edges   map[TaskID][]TaskID
	reverse map[TaskID][]TaskID
	mu      sync.RWMutex
}

type TaskNode struct {
	Task      *Task
	Level     int
	Critical  bool
	SlackTime time.Duration
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:   make(map[TaskID]*TaskNode),
		edges:   make(map[TaskID][]TaskID),
		reverse: make(map[TaskID][]TaskID),
	}
}

func (g *DependencyGraph) AddTask(task *Task) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if _, exists := g.nodes[task.ID]; exists {
		return fmt.Errorf("task %s already exists", task.ID)
	}

	g.nodes[task.ID] = &TaskNode{Task: task}
	g.edges[task.ID] = make([]TaskID, 0)
	g.reverse[task.ID] = make([]TaskID, 0)

	for _, dep := range task.Dependencies {
		if _, exists := g.nodes[dep]; !exists {
			return fmt.Errorf("dependency %s not found", dep)
		}
		g.edges[dep] = append(g.edges[dep], task.ID)
		g.reverse[task.ID] = append(g.reverse[task.ID], dep)
	}

	return nil
}

func (g *DependencyGraph) GetTask(taskID TaskID) *Task {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if node, ok := g.nodes[taskID]; ok {
		return node.Task
	}
	return nil
}

func (g *DependencyGraph) GetNode(taskID TaskID) *TaskNode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.nodes[taskID]
}

func (g *DependencyGraph) TopologicalSort() ([]TaskID, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	inDegree := make(map[TaskID]int)
	for id := range g.nodes {
		inDegree[id] = len(g.reverse[id])
	}

	queue := make([]TaskID, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]TaskID, 0, len(g.nodes))

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, neighbor := range g.edges[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected in dependency graph")
	}

	return result, nil
}

func (g *DependencyGraph) AssignLevels() error {
	g.mu.Lock()
	defer g.mu.Unlock()

	sorted, err := g.topologicalSortUnsafe()
	if err != nil {
		return err
	}

	for _, id := range sorted {
		maxParentLevel := -1
		for _, parent := range g.reverse[id] {
			if g.nodes[parent].Level > maxParentLevel {
				maxParentLevel = g.nodes[parent].Level
			}
		}
		g.nodes[id].Level = maxParentLevel + 1
	}

	return nil
}

func (g *DependencyGraph) GetReadyTasks() []TaskID {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ready := make([]TaskID, 0)

	for id, node := range g.nodes {
		if node.Task.Status != StatusPending {
			continue
		}

		allCompleted := true
		for _, dep := range g.reverse[id] {
			if g.nodes[dep].Task.Status != StatusCompleted {
				allCompleted = false
				break
			}
		}

		if allCompleted {
			ready = append(ready, id)
		}
	}

	sort.Slice(ready, func(i, j int) bool {
		return g.nodes[ready[i]].Task.Priority > g.nodes[ready[j]].Task.Priority
	})

	return ready
}

func (g *DependencyGraph) CriticalPath() ([]TaskID, time.Duration) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	earliestFinish := make(map[TaskID]time.Duration)
	latestStart := make(map[TaskID]time.Duration)

	sorted, _ := g.topologicalSortUnsafe()

	var maxDuration time.Duration

	for _, id := range sorted {
		var maxDepFinish time.Duration
		for _, dep := range g.reverse[id] {
			if earliestFinish[dep] > maxDepFinish {
				maxDepFinish = earliestFinish[dep]
			}
		}
		taskDuration := g.nodes[id].Task.Timeout
		if taskDuration == 0 {
			taskDuration = 5 * time.Minute
		}
		earliestFinish[id] = maxDepFinish + taskDuration

		if earliestFinish[id] > maxDuration {
			maxDuration = earliestFinish[id]
		}
	}

	for id := range g.nodes {
		latestStart[id] = maxDuration
	}

	for i := len(sorted) - 1; i >= 0; i-- {
		id := sorted[i]
		for _, dep := range g.reverse[id] {
			if latestStart[id]-g.nodes[dep].Task.Timeout < latestStart[dep] {
				latestStart[dep] = latestStart[id] - g.nodes[dep].Task.Timeout
			}
		}
	}

	critical := make([]TaskID, 0)
	for _, id := range sorted {
		taskDuration := g.nodes[id].Task.Timeout
		if taskDuration == 0 {
			taskDuration = 5 * time.Minute
		}
		ef := earliestFinish[id]
		ls := latestStart[id]
		es := ef - taskDuration
		isCritical := es == ls
		if isCritical {
			critical = append(critical, id)
		}
	}

	return critical, maxDuration
}

func (g *DependencyGraph) HasCycle() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[TaskID]bool)
	recStack := make(map[TaskID]bool)

	var hasCycle func(TaskID) bool
	hasCycle = func(id TaskID) bool {
		visited[id] = true
		recStack[id] = true

		for _, dep := range g.reverse[id] {
			if !visited[dep] {
				if hasCycle(dep) {
					return true
				}
			} else if recStack[dep] {
				return true
			}
		}

		recStack[id] = false
		return false
	}

	for id := range g.nodes {
		if !visited[id] {
			if hasCycle(id) {
				return true
			}
		}
	}

	return false
}

func (g *DependencyGraph) topologicalSortUnsafe() ([]TaskID, error) {
	inDegree := make(map[TaskID]int)
	for id := range g.nodes {
		inDegree[id] = len(g.reverse[id])
	}

	queue := make([]TaskID, 0)
	for id, deg := range inDegree {
		if deg == 0 {
			queue = append(queue, id)
		}
	}

	result := make([]TaskID, 0, len(g.nodes))

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		for _, neighbor := range g.edges[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	if len(result) != len(g.nodes) {
		return nil, fmt.Errorf("cycle detected")
	}

	return result, nil
}

func (g *DependencyGraph) Count() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.nodes)
}
