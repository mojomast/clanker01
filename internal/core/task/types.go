package task

import (
	"time"
)

type TaskID string
type AgentID string

type Task struct {
	ID            TaskID            `json:"id"`
	Name          string            `json:"name"`
	Description   string            `json:"description"`
	Status        TaskStatus        `json:"status"`
	Priority      Priority          `json:"priority"`
	Kind          TaskKind          `json:"kind"`
	Input         map[string]any    `json:"input"`
	Output        map[string]any    `json:"output,omitempty"`
	Dependencies  []TaskID          `json:"dependencies"`
	AssignedAgent AgentID           `json:"assigned_agent,omitempty"`
	CreatedAt     time.Time         `json:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at"`
	StartedAt     *time.Time        `json:"started_at,omitempty"`
	CompletedAt   *time.Time        `json:"completed_at,omitempty"`
	Timeout       time.Duration     `json:"timeout"`
	RetryCount    int               `json:"retry_count"`
	MaxRetries    int               `json:"max_retries"`
	Verification  *VerificationSpec `json:"verification,omitempty"`
	Metadata      map[string]any    `json:"metadata,omitempty"`
}

type TaskStatus string

const (
	StatusPending   TaskStatus = "pending"
	StatusReady     TaskStatus = "ready"
	StatusRunning   TaskStatus = "running"
	StatusCompleted TaskStatus = "completed"
	StatusFailed    TaskStatus = "failed"
	StatusCancelled TaskStatus = "cancelled"
	StatusBlocked   TaskStatus = "blocked"
)

type Priority int

const (
	PriorityLow      Priority = 0
	PriorityNormal   Priority = 1
	PriorityHigh     Priority = 2
	PriorityCritical Priority = 3
)

type TaskKind string

const (
	KindCompute   TaskKind = "compute"
	KindIO        TaskKind = "io"
	KindNetwork   TaskKind = "network"
	KindAggregate TaskKind = "aggregate"
	KindDecision  TaskKind = "decision"
)

type TaskTemplate struct {
	NameTemplate   string         `json:"name_template"`
	Kind           TaskKind       `json:"kind"`
	DefaultTimeout time.Duration  `json:"default_timeout"`
	RequiredInput  []InputSpec    `json:"required_input"`
	OutputSchema   map[string]any `json:"output_schema"`
	DefaultRetries int            `json:"default_retries"`
}

type InputSpec struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

type VerificationSpec struct {
	Type       VerificationType `json:"type"`
	Assertions []Assertion      `json:"assertions"`
	Timeout    time.Duration    `json:"timeout"`
	RetryCount int              `json:"retry_count"`
}

type VerificationType string

const (
	VerifyOutput   VerificationType = "output"
	VerifySchema   VerificationType = "schema"
	VerifyCustom   VerificationType = "custom"
	VerifyExternal VerificationType = "external"
)

type Assertion struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Expected any    `json:"expected"`
	Path     string `json:"path"`
	Message  string `json:"message"`
}

type ResourceLimit struct {
	MaxCPU    int `json:"max_cpu"`
	MaxMemory int `json:"max_memory"`
	MaxTasks  int `json:"max_tasks"`
	MaxAgents int `json:"max_agents"`
}
