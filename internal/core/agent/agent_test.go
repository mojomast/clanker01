package agent

import (
	"context"
	"testing"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type mockProvider struct{}

func (m *mockProvider) Name() string {
	return "mock"
}

func (m *mockProvider) Models() []api.ModelInfo {
	return []api.ModelInfo{
		{ID: "mock-model", MaxTokens: 4000},
	}
}

func (m *mockProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return &api.ChatResponse{
		ID:    "test-response",
		Model: req.Model,
		Choices: []api.Choice{
			{
				Index: 0,
				Message: api.Message{
					Role:    "assistant",
					Content: "Test response",
				},
			},
		},
		Usage: api.Usage{
			PromptTokens:     10,
			CompletionTokens: 20,
			TotalTokens:      30,
		},
	}, nil
}

func (m *mockProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	ch := make(chan api.ChatStreamEvent, 1)
	close(ch)
	return ch, nil
}

func (m *mockProvider) SupportsStreaming() bool {
	return false
}

func (m *mockProvider) SupportsFunctionCalling() bool {
	return true
}

func (m *mockProvider) SupportsVision() bool {
	return false
}

func (m *mockProvider) SupportsAudio() bool {
	return false
}

func (m *mockProvider) MaxTokens(model string) int {
	return 4000
}

func (m *mockProvider) Configure(config *api.ProviderConfig) error {
	return nil
}

func (m *mockProvider) Metrics() *api.ProviderMetrics {
	return &api.ProviderMetrics{}
}

func TestNewBaseAgent(t *testing.T) {
	config := &api.AgentConfig{
		ID:   "test-agent",
		Type: api.AgentTypeCoder,
		Name: "Test Agent",
	}

	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	agent := NewBaseAgent("agent-1", api.AgentTypeCoder, config, provider, skills, mcp)

	if agent.ID() != "agent-1" {
		t.Errorf("Expected ID agent-1, got %s", agent.ID())
	}

	if agent.Type() != api.AgentTypeCoder {
		t.Errorf("Expected type coder, got %s", agent.Type())
	}

	if agent.Name() != "Test Agent" {
		t.Errorf("Expected name Test Agent, got %s", agent.Name())
	}

	if agent.Status() != api.AgentStatusCreated {
		t.Errorf("Expected status created, got %s", agent.Status())
	}
}

func TestBaseAgentInitialize(t *testing.T) {
	config := &api.AgentConfig{
		ID:            "test-agent",
		Type:          api.AgentTypeCoder,
		Name:          "Test Agent",
		Model:         "mock-model",
		SystemPrompt:  "You are a helpful assistant",
		Skills:        []string{"coding", "testing"},
		MaxConcurrent: 5,
		Timeout:       30 * time.Second,
		MaxRetries:    3,
		ResourceLimits: api.ResourceLimits{
			MaxMemoryMB:      1024,
			MaxCPUPercent:    80,
			MaxTokensPerTask: 2000,
			MaxTasksPerHour:  100,
		},
	}

	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	agent := NewBaseAgent("agent-1", api.AgentTypeCoder, config, provider, skills, mcp)

	ctx := context.Background()
	err := agent.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	if agent.Status() != api.AgentStatusReady {
		t.Errorf("Expected status ready, got %s", agent.Status())
	}

	if _, ok := skills.GetSkill("coding"); !ok {
		t.Error("Expected coding skill to be loaded")
	}
}

func TestBaseAgentStartStop(t *testing.T) {
	config := &api.AgentConfig{
		ID:    "test-agent",
		Type:  api.AgentTypeCoder,
		Name:  "Test Agent",
		Model: "mock-model",
	}

	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	agent := NewBaseAgent("agent-1", api.AgentTypeCoder, config, provider, skills, mcp)

	ctx := context.Background()
	err := agent.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	err = agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = agent.Stop(ctx)
	if err != nil {
		t.Fatalf("Failed to stop agent: %v", err)
	}

	if agent.Status() != api.AgentStatusTerminated {
		t.Errorf("Expected status terminated, got %s", agent.Status())
	}
}

func TestBaseAgentExecute(t *testing.T) {
	config := &api.AgentConfig{
		ID:    "test-agent",
		Type:  api.AgentTypeCoder,
		Name:  "Test Agent",
		Model: "mock-model",
	}

	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	agent := NewBaseAgent("agent-1", api.AgentTypeCoder, config, provider, skills, mcp)

	ctx := context.Background()
	err := agent.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	task := &api.Task{
		ID:        "task-1",
		Type:      "code",
		Prompt:    "Write a function to add two numbers",
		Priority:  1,
		Status:    api.TaskStatusQueued,
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}

	result, err := agent.Execute(ctx, task)
	if err != nil {
		t.Fatalf("Failed to execute task: %v", err)
	}

	if !result.Success {
		t.Error("Expected task to succeed")
	}

	if result.TaskID != "task-1" {
		t.Errorf("Expected task ID task-1, got %s", result.TaskID)
	}

	metrics := agent.Metrics()
	if metrics.TasksCompleted != 1 {
		t.Errorf("Expected 1 completed task, got %d", metrics.TasksCompleted)
	}

	if metrics.TotalTokensUsed != 30 {
		t.Errorf("Expected 30 total tokens used, got %d", metrics.TotalTokensUsed)
	}
}

func TestBaseAgentPauseResume(t *testing.T) {
	config := &api.AgentConfig{
		ID:    "test-agent",
		Type:  api.AgentTypeCoder,
		Name:  "Test Agent",
		Model: "mock-model",
	}

	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	agent := NewBaseAgent("agent-1", api.AgentTypeCoder, config, provider, skills, mcp)

	ctx := context.Background()
	err := agent.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	task := &api.Task{
		ID:        "task-1",
		Type:      "code",
		Prompt:    "Write a function",
		Priority:  1,
		Status:    api.TaskStatusQueued,
		CreatedAt: time.Now(),
		StartedAt: time.Now(),
	}

	_, err = agent.Execute(ctx, task)
	if err != nil {
		t.Fatalf("Failed to execute task: %v", err)
	}

	err = agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	err = agent.Pause(ctx)
	if err != nil {
		t.Fatalf("Failed to pause agent: %v", err)
	}

	if agent.Status() != api.AgentStatusPaused {
		t.Errorf("Expected status paused, got %s", agent.Status())
	}

	err = agent.Resume(ctx)
	if err != nil {
		t.Fatalf("Failed to resume agent: %v", err)
	}

	if agent.Status() != api.AgentStatusReady {
		t.Errorf("Expected status ready, got %s", agent.Status())
	}

	agent.Stop(ctx)
}

func TestBaseAgentMessaging(t *testing.T) {
	config := &api.AgentConfig{
		ID:    "test-agent",
		Type:  api.AgentTypeCoder,
		Name:  "Test Agent",
		Model: "mock-model",
	}

	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	agent := NewBaseAgent("agent-1", api.AgentTypeCoder, config, provider, skills, mcp)

	ctx := context.Background()
	err := agent.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	err = agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	msg := &api.AgentMessage{
		ID:        "msg-1",
		Timestamp: time.Now(),
		Sender: api.AgentRef{
			ID:   "sender-1",
			Type: api.AgentTypeCoordinator,
		},
		Receiver: api.AgentRef{
			ID:   "agent-1",
			Type: api.AgentTypeCoder,
		},
		Type:     api.MessageTypeTaskAssignment,
		Priority: api.PriorityNormal,
		Payload:  nil,
	}

	err = agent.SendMessage(ctx, msg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	agent.Stop(ctx)
}

func TestBaseAgentHealth(t *testing.T) {
	config := &api.AgentConfig{
		ID:    "test-agent",
		Type:  api.AgentTypeCoder,
		Name:  "Test Agent",
		Model: "mock-model",
	}

	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	agent := NewBaseAgent("agent-1", api.AgentTypeCoder, config, provider, skills, mcp)

	ctx := context.Background()
	err := agent.Initialize(ctx, config)
	if err != nil {
		t.Fatalf("Failed to initialize agent: %v", err)
	}

	err = agent.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start agent: %v", err)
	}

	health := agent.Health()
	if health.Status != "ready" {
		t.Errorf("Expected health status ready, got %s", health.Status)
	}

	if health.ActiveRequests != 0 {
		t.Errorf("Expected 0 active requests, got %d", health.ActiveRequests)
	}

	agent.Stop(ctx)
}
