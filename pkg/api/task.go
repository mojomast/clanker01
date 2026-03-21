package api

import (
	"time"
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusBlocked   TaskStatus = "blocked"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
	TaskStatusVerifying TaskStatus = "verifying"
)

type Task struct {
	ID            string
	Type          string
	ParentID      string
	Dependencies  []string
	Priority      int
	Status        TaskStatus
	AgentType     AgentType
	AssignedAgent string

	Prompt         string
	Description    string
	Requirements   []string
	ExpectedOutput string

	Progress float64
	Result   *TaskResult
	Error    error

	MaxRetries int
	RetryCount int
	Timeout    time.Duration

	IsComplex    bool           `json:"is_complex,omitempty"`
	Verification map[string]any `json:"verification,omitempty"`

	CreatedAt   time.Time
	StartedAt   time.Time
	CompletedAt time.Time
	Metadata    map[string]any
}

type TaskResult struct {
	TaskID      string
	Success     bool
	Output      any
	Artifacts   []Artifact
	Error       error
	Metrics     *TaskMetrics
	CompletedAt time.Time
}
