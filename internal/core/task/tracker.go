package task

import (
	"sync"
	"time"
)

type ProgressTracker struct {
	tasks  map[TaskID]*TaskProgress
	planID string
	mu     sync.RWMutex
}

type TaskProgress struct {
	TaskID      TaskID             `json:"task_id"`
	Status      TaskStatus         `json:"status"`
	Percent     float64            `json:"percent"`
	Message     string             `json:"message"`
	StartedAt   time.Time          `json:"started_at,omitempty"`
	UpdatedAt   time.Time          `json:"updated_at"`
	Error       string             `json:"error,omitempty"`
	SubProgress map[string]float64 `json:"sub_progress,omitempty"`
}

type PlanProgress struct {
	PlanID       string        `json:"plan_id"`
	TotalTasks   int           `json:"total_tasks"`
	Completed    int           `json:"completed"`
	Running      int           `json:"running"`
	Pending      int           `json:"pending"`
	Failed       int           `json:"failed"`
	Percent      float64       `json:"percent"`
	ETA          time.Duration `json:"eta"`
	CriticalPath []TaskID      `json:"critical_path_remaining"`
}

type ProgressUpdate struct {
	Status   *TaskStatus
	Percent  *float64
	Message  *string
	Error    *string
	Progress *TaskProgress
}

func NewProgressTracker(planID string) *ProgressTracker {
	return &ProgressTracker{
		tasks:  make(map[TaskID]*TaskProgress),
		planID: planID,
	}
}

func (pt *ProgressTracker) Initialize(tasks []*Task) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	now := time.Now()
	for _, task := range tasks {
		pt.tasks[task.ID] = &TaskProgress{
			TaskID:    task.ID,
			Status:    task.Status,
			Percent:   0,
			Message:   "Task initialized",
			UpdatedAt: now,
		}
	}
}

func (pt *ProgressTracker) Update(taskID TaskID, update ProgressUpdate) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	progress, exists := pt.tasks[taskID]
	if !exists {
		progress = &TaskProgress{
			TaskID: taskID,
		}
		pt.tasks[taskID] = progress
	}

	if update.Percent != nil {
		progress.Percent = *update.Percent
	}
	if update.Message != nil {
		progress.Message = *update.Message
	}
	if update.Status != nil {
		progress.Status = *update.Status
		if progress.Status == StatusRunning && progress.StartedAt.IsZero() {
			progress.StartedAt = time.Now()
		}
	}
	if update.Error != nil {
		progress.Error = *update.Error
	}
	if update.Progress != nil {
		if update.Progress.SubProgress != nil {
			progress.SubProgress = update.Progress.SubProgress
		}
	}

	progress.UpdatedAt = time.Now()

	return nil
}

func (pt *ProgressTracker) Get(taskID TaskID) (*TaskProgress, error) {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	progress, exists := pt.tasks[taskID]
	if !exists {
		return nil, ErrTaskNotFound
	}

	return progress, nil
}

func (pt *ProgressTracker) GetPlanProgress() *PlanProgress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	stats := PlanProgress{
		PlanID: pt.planID,
	}

	var totalCompletedDuration time.Duration
	var completedCount int

	for _, p := range pt.tasks {
		stats.TotalTasks++
		switch p.Status {
		case StatusCompleted:
			stats.Completed++
			if !p.StartedAt.IsZero() {
				totalCompletedDuration += p.UpdatedAt.Sub(p.StartedAt)
				completedCount++
			}
		case StatusRunning:
			stats.Running++
		case StatusPending, StatusReady:
			stats.Pending++
		case StatusFailed:
			stats.Failed++
		case StatusBlocked:
			stats.Pending++
		case StatusCancelled:
			stats.Failed++
		}
	}

	if stats.TotalTasks > 0 {
		stats.Percent = float64(stats.Completed) / float64(stats.TotalTasks) * 100
	}

	if stats.Pending > 0 && completedCount > 0 {
		avgDuration := totalCompletedDuration / time.Duration(completedCount)
		stats.ETA = avgDuration * time.Duration(stats.Pending+stats.Running)
	}

	return &stats
}

func (pt *ProgressTracker) GetTasksByStatus(status TaskStatus) []TaskID {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var taskIDs []TaskID
	for taskID, progress := range pt.tasks {
		if progress.Status == status {
			taskIDs = append(taskIDs, taskID)
		}
	}

	return taskIDs
}

func (pt *ProgressTracker) SetSubProgress(taskID TaskID, key string, value float64) error {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	progress, exists := pt.tasks[taskID]
	if !exists {
		return ErrTaskNotFound
	}

	if progress.SubProgress == nil {
		progress.SubProgress = make(map[string]float64)
	}

	progress.SubProgress[key] = value

	var total float64
	for _, v := range progress.SubProgress {
		total += v
	}
	progress.Percent = total / float64(len(progress.SubProgress))
	progress.UpdatedAt = time.Now()

	return nil
}

func (pt *ProgressTracker) UpdateStatus(taskID TaskID, status TaskStatus) error {
	return pt.Update(taskID, ProgressUpdate{
		Status: &status,
	})
}

func (pt *ProgressTracker) UpdatePercent(taskID TaskID, percent float64) error {
	return pt.Update(taskID, ProgressUpdate{
		Percent: &percent,
	})
}

func (pt *ProgressTracker) UpdateMessage(taskID TaskID, message string) error {
	return pt.Update(taskID, ProgressUpdate{
		Message: &message,
	})
}

func (pt *ProgressTracker) UpdateError(taskID TaskID, err string) error {
	return pt.Update(taskID, ProgressUpdate{
		Error: &err,
	})
}

func (pt *ProgressTracker) MarkTaskStarted(taskID TaskID) error {
	status := StatusRunning
	return pt.Update(taskID, ProgressUpdate{
		Status: &status,
	})
}

func (pt *ProgressTracker) MarkTaskCompleted(taskID TaskID) error {
	percent := 100.0
	status := StatusCompleted
	return pt.Update(taskID, ProgressUpdate{
		Status:  &status,
		Percent: &percent,
	})
}

func (pt *ProgressTracker) MarkTaskFailed(taskID TaskID, err string) error {
	status := StatusFailed
	return pt.Update(taskID, ProgressUpdate{
		Status: &status,
		Error:  &err,
	})
}

func (pt *ProgressTracker) Reset() {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	pt.tasks = make(map[TaskID]*TaskProgress)
}

func (pt *ProgressTracker) GetAllProgress() map[TaskID]*TaskProgress {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	copy := make(map[TaskID]*TaskProgress, len(pt.tasks))
	for id, progress := range pt.tasks {
		progressCopy := *progress
		copy[id] = &progressCopy
	}

	return copy
}

func (pt *ProgressTracker) TaskCount() int {
	pt.mu.RLock()
	defer pt.mu.RUnlock()
	return len(pt.tasks)
}

func (pt *ProgressTracker) IsComplete() bool {
	progress := pt.GetPlanProgress()
	return progress.TotalTasks > 0 && progress.Completed == progress.TotalTasks
}

func (pt *ProgressTracker) HasFailedTasks() bool {
	progress := pt.GetPlanProgress()
	return progress.Failed > 0
}

var ErrTaskNotFound = errTaskNotFound{}

type errTaskNotFound struct{}

func (e errTaskNotFound) Error() string {
	return "task not found"
}

func IsTaskNotFound(err error) bool {
	_, ok := err.(errTaskNotFound)
	return ok
}
