package orchestrator

import (
	"container/heap"
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type TaskEvent struct {
	Type   TaskEventType
	TaskID string
	Task   *api.Task
	Result *api.TaskResult
	Error  error
}

type TaskEventType string

const (
	TaskEventEnqueued  TaskEventType = "enqueued"
	TaskEventDequeued  TaskEventType = "dequeued"
	TaskEventCompleted TaskEventType = "completed"
	TaskEventFailed    TaskEventType = "failed"
)

type TaskQueue struct {
	mu           sync.RWMutex
	pending      *PriorityQueue
	running      map[string]*api.Task
	completed    map[string]*api.Task
	failed       map[string]*api.Task
	dependencies *DependencyGraph
	subscribers  []chan TaskEvent
}

type PriorityQueue struct {
	items []*api.Task
}

var _ heap.Interface = (*PriorityQueue)(nil)

func (pq *PriorityQueue) Len() int { return len(pq.items) }

func (pq *PriorityQueue) Less(i, j int) bool {
	return pq.items[i].Priority > pq.items[j].Priority
}

func (pq *PriorityQueue) Swap(i, j int) {
	pq.items[i], pq.items[j] = pq.items[j], pq.items[i]
}

func (pq *PriorityQueue) Push(x any) {
	pq.items = append(pq.items, x.(*api.Task))
}

func (pq *PriorityQueue) Pop() any {
	old := pq.items
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	pq.items = old[0 : n-1]
	return item
}

func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		pending:      &PriorityQueue{},
		running:      make(map[string]*api.Task),
		completed:    make(map[string]*api.Task),
		failed:       make(map[string]*api.Task),
		dependencies: NewDependencyGraph(),
	}
}

func (q *TaskQueue) Enqueue(ctx context.Context, task *api.Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if err := q.validate(task); err != nil {
		return err
	}

	q.dependencies.Add(task)

	if q.dependencies.IsReady(task.ID) {
		heap.Push(q.pending, task)
		task.Status = api.TaskStatusQueued
	} else {
		task.Status = api.TaskStatusBlocked
	}

	q.notify(TaskEvent{
		Type:   TaskEventEnqueued,
		TaskID: task.ID,
		Task:   task,
	})

	return nil
}

func (q *TaskQueue) Dequeue(ctx context.Context, agentType api.AgentType) (*api.Task, error) {
	q.mu.Lock()
	defer q.mu.Unlock()

	var bestTask *api.Task
	var bestIndex int
	var bestScore float64

	items := q.pending.items
	for i := 0; i < len(items); i++ {
		task := items[i]
		if !q.matchesAgent(task, agentType) {
			continue
		}

		score := q.scoreTask(task, agentType)
		if score > bestScore {
			bestScore = score
			bestTask = task
			bestIndex = i
		}
	}

	if bestTask == nil {
		return nil, fmt.Errorf("no task available")
	}

	heap.Remove(q.pending, bestIndex)

	bestTask.Status = api.TaskStatusRunning
	bestTask.StartedAt = time.Now()
	q.running[bestTask.ID] = bestTask

	q.notify(TaskEvent{
		Type:   TaskEventDequeued,
		TaskID: bestTask.ID,
		Task:   bestTask,
	})

	return bestTask, nil
}

func (q *TaskQueue) Complete(taskID string, result *api.TaskResult) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.running[taskID]
	if !ok {
		return fmt.Errorf("task not running: %s", taskID)
	}

	task.Status = api.TaskStatusCompleted
	task.Result = result
	task.CompletedAt = time.Now()

	delete(q.running, taskID)
	q.completed[taskID] = task

	dependents := q.dependencies.GetDependents(taskID)
	for _, depID := range dependents {
		if q.dependencies.IsReady(depID) {
			depTask := q.dependencies.GetTask(depID)
			if depTask != nil && depTask.Status == api.TaskStatusBlocked {
				heap.Push(q.pending, depTask)
				depTask.Status = api.TaskStatusQueued
			}
		}
	}

	q.notify(TaskEvent{
		Type:   TaskEventCompleted,
		TaskID: taskID,
		Task:   task,
		Result: result,
	})

	return nil
}

func (q *TaskQueue) Fail(taskID string, err error) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	task, ok := q.running[taskID]
	if !ok {
		return fmt.Errorf("task not running: %s", taskID)
	}

	task.Status = api.TaskStatusFailed
	task.Error = err
	task.CompletedAt = time.Now()

	delete(q.running, taskID)
	q.failed[taskID] = task

	q.failDependents(taskID, err)

	q.notify(TaskEvent{
		Type:   TaskEventFailed,
		TaskID: taskID,
		Task:   task,
		Error:  err,
	})

	return nil
}

func (q *TaskQueue) validate(task *api.Task) error {
	if task.ID == "" {
		return fmt.Errorf("task ID is required")
	}
	if task.Priority < 0 {
		return fmt.Errorf("task priority must be non-negative")
	}
	return nil
}

func (q *TaskQueue) matchesAgent(task *api.Task, agentType api.AgentType) bool {
	if task.AgentType == "" || task.AgentType == agentType {
		return true
	}
	return false
}

func (q *TaskQueue) scoreTask(task *api.Task, agentType api.AgentType) float64 {
	score := float64(task.Priority)

	if task.AgentType == "" || task.AgentType == agentType {
		score += 100
	}

	age := time.Since(task.CreatedAt).Minutes()
	score += age * 0.5

	dependents := q.dependencies.GetDependents(task.ID)
	score += float64(len(dependents)) * 10

	return score
}

func (q *TaskQueue) failDependents(taskID string, err error) {
	dependents := q.dependencies.GetDependents(taskID)
	for _, depID := range dependents {
		task := q.dependencies.GetTask(depID)
		if task != nil && (task.Status == api.TaskStatusBlocked || task.Status == api.TaskStatusPending || task.Status == api.TaskStatusQueued) {
			task.Status = api.TaskStatusFailed
			task.Error = err
			task.CompletedAt = time.Now()
			q.failed[depID] = task

			q.notify(TaskEvent{
				Type:   TaskEventFailed,
				TaskID: depID,
				Task:   task,
				Error:  err,
			})

			q.failDependents(depID, err)
		}
	}
}

func (q *TaskQueue) Subscribe() <-chan TaskEvent {
	q.mu.Lock()
	defer q.mu.Unlock()

	ch := make(chan TaskEvent, 100)
	q.subscribers = append(q.subscribers, ch)
	return ch
}

func (q *TaskQueue) notify(event TaskEvent) {
	for _, sub := range q.subscribers {
		select {
		case sub <- event:
		default:
		}
	}
}

func (q *TaskQueue) GetStats() (pending, running, completed, failed int) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.pending.Len(), len(q.running), len(q.completed), len(q.failed)
}

func (q *TaskQueue) GetTask(taskID string) *api.Task {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if task, ok := q.running[taskID]; ok {
		return task
	}
	if task, ok := q.completed[taskID]; ok {
		return task
	}
	if task, ok := q.failed[taskID]; ok {
		return task
	}
	return q.dependencies.GetTask(taskID)
}
