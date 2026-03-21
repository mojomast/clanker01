package orchestrator

import (
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/pkg/api"
)

type DependencyGraph struct {
	mu    sync.RWMutex
	tasks map[string]*api.Task
	deps  map[string][]string
	rdeps map[string][]string
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		tasks: make(map[string]*api.Task),
		deps:  make(map[string][]string),
		rdeps: make(map[string][]string),
	}
}

func (g *DependencyGraph) Add(task *api.Task) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.tasks[task.ID] = task
	g.deps[task.ID] = append([]string{}, task.Dependencies...)

	for _, depID := range task.Dependencies {
		g.rdeps[depID] = append(g.rdeps[depID], task.ID)
	}
}

func (g *DependencyGraph) IsReady(taskID string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	deps := g.deps[taskID]
	for _, depID := range deps {
		task, ok := g.tasks[depID]
		if !ok || task.Status != api.TaskStatusCompleted {
			return false
		}
	}
	return true
}

func (g *DependencyGraph) GetTask(taskID string) *api.Task {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.tasks[taskID]
}

func (g *DependencyGraph) GetDependents(taskID string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if deps, ok := g.rdeps[taskID]; ok {
		return append([]string{}, deps...)
	}
	return nil
}

func (g *DependencyGraph) DetectCycle(taskID string) ([]string, bool) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	path := []string{}

	return g.detectCycleDFS(taskID, visited, path)
}

func (g *DependencyGraph) detectCycleDFS(
	current string,
	visited map[string]bool,
	path []string,
) ([]string, bool) {
	if visited[current] {
		for i, id := range path {
			if id == current {
				return append(path[i:], current), true
			}
		}
		return nil, false
	}

	visited[current] = true
	path = append(path, current)

	for _, dep := range g.deps[current] {
		if cycle, found := g.detectCycleDFS(dep, visited, path); found {
			return cycle, true
		}
	}

	visited[current] = false
	return nil, false
}

func (g *DependencyGraph) TopologicalOrder() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	result := []string{}
	visited := make(map[string]bool)
	temp := make(map[string]bool)

	var visit func(string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		if temp[id] {
			return fmt.Errorf("cycle detected at %s", id)
		}

		temp[id] = true

		for _, dep := range g.deps[id] {
			if err := visit(dep); err != nil {
				return err
			}
		}

		delete(temp, id)
		visited[id] = true
		result = append(result, id)
		return nil
	}

	for id := range g.tasks {
		if err := visit(id); err != nil {
			return nil
		}
	}

	return result
}

func (g *DependencyGraph) ExecutionBatches() [][]string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	inDegree := make(map[string]int)
	for id := range g.tasks {
		inDegree[id] = len(g.deps[id])
	}

	var batches [][]string

	for len(inDegree) > 0 {
		var ready []string
		for id, deg := range inDegree {
			if deg == 0 {
				ready = append(ready, id)
			}
		}

		if len(ready) == 0 {
			break
		}

		batches = append(batches, ready)

		for _, id := range ready {
			delete(inDegree, id)
			for _, dependent := range g.rdeps[id] {
				if _, ok := inDegree[dependent]; ok {
					inDegree[dependent]--
				}
			}
		}
	}

	return batches
}

func (g *DependencyGraph) HasCycle() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	visited := make(map[string]bool)
	recStack := make(map[string]bool)

	var hasCycle func(string) bool
	hasCycle = func(id string) bool {
		visited[id] = true
		recStack[id] = true

		for _, dep := range g.deps[id] {
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

	for id := range g.tasks {
		if !visited[id] {
			if hasCycle(id) {
				return true
			}
		}
	}

	return false
}

func (g *DependencyGraph) Count() int {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return len(g.tasks)
}

func (g *DependencyGraph) Remove(taskID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.tasks, taskID)
	delete(g.deps, taskID)

	for depID := range g.rdeps {
		filtered := []string{}
		for _, id := range g.rdeps[depID] {
			if id != taskID {
				filtered = append(filtered, id)
			}
		}
		g.rdeps[depID] = filtered
	}

	delete(g.rdeps, taskID)
}
