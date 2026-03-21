package api

import (
	"context"
	"time"
)

type AgentType string

const (
	AgentTypeArchitect   AgentType = "architect"
	AgentTypeCoder       AgentType = "coder"
	AgentTypeTester      AgentType = "tester"
	AgentTypeReviewer    AgentType = "reviewer"
	AgentTypeResearcher  AgentType = "researcher"
	AgentTypeCoordinator AgentType = "coordinator"
)

type AgentStatus string

const (
	AgentStatusCreated    AgentStatus = "created"
	AgentStatusReady      AgentStatus = "ready"
	AgentStatusRunning    AgentStatus = "running"
	AgentStatusPaused     AgentStatus = "paused"
	AgentStatusError      AgentStatus = "error"
	AgentStatusTerminated AgentStatus = "terminated"
)

type Agent interface {
	ID() string
	Type() AgentType
	Name() string
	Initialize(ctx context.Context, config *AgentConfig) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Pause(ctx context.Context) error
	Resume(ctx context.Context) error
	Execute(ctx context.Context, task *Task) (*AgentResult, error)
	SendMessage(ctx context.Context, msg *AgentMessage) error
	ReceiveMessage() <-chan *AgentMessage
	Broadcast(ctx context.Context, msg *AgentMessage) error
	Status() AgentStatus
	CurrentTask() *Task
	Metrics() *AgentMetrics
	Health() *AgentHealth
}

type AgentConfig struct {
	ID             string
	Type           AgentType
	Name           string
	Model          string
	SystemPrompt   string
	Skills         []string
	MaxConcurrent  int
	Timeout        time.Duration
	MaxRetries     int
	ResourceLimits ResourceLimits
}

type ResourceLimits struct {
	MaxMemoryMB      int
	MaxCPUPercent    int
	MaxTokensPerTask int
	MaxTasksPerHour  int
}

type AgentResult struct {
	TaskID      string
	Success     bool
	Output      any
	Artifacts   []Artifact
	Metrics     *TaskMetrics
	Error       error
	CompletedAt time.Time
}

type AgentMetrics struct {
	TasksCompleted  int64
	TasksFailed     int64
	AvgTaskDuration time.Duration
	TotalTokensUsed int64
	TotalCost       float64
	LastActivity    time.Time
}

type AgentHealth struct {
	Status         string
	LastHeartbeat  time.Time
	CPUUsage       float64
	MemoryUsageMB  int
	ActiveRequests int
	ErrorCount     int
}

type Artifact struct {
	ID        string
	Type      string
	Name      string
	Content   []byte
	Metadata  map[string]any
	CreatedAt time.Time
}

type TaskMetrics struct {
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	TokensUsed int
	ToolCalls  int
	Messages   int
	RetryCount int
}
