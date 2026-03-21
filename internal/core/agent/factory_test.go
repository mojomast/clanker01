package agent

import (
	"context"
	"testing"

	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewAgentFactory(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	cfg := &FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	}

	factory := NewAgentFactory(cfg)

	if factory == nil {
		t.Fatal("Expected factory to be created")
	}

	if factory.registry == nil {
		t.Error("Expected registry to be initialized")
	}

	if factory.agents == nil {
		t.Error("Expected agents map to be initialized")
	}
}

func TestAgentRegistry(t *testing.T) {
	registry := NewAgentRegistry()

	template := &AgentTemplate{
		Type:          api.AgentTypeCoder,
		SystemPrompt:  "You are a coding assistant",
		Model:         "mock-model",
		Skills:        []string{"coding", "testing"},
		MaxConcurrent: 5,
		Timeout:       30,
		MaxRetries:    3,
		ResourceLimits: api.ResourceLimits{
			MaxTokensPerTask: 2000,
		},
	}

	registry.RegisterTemplate(api.AgentTypeCoder, template)

	retrieved, err := registry.GetTemplate(api.AgentTypeCoder)
	if err != nil {
		t.Fatalf("Failed to get template: %v", err)
	}

	if retrieved.Type != api.AgentTypeCoder {
		t.Errorf("Expected type coder, got %s", retrieved.Type)
	}

	_, err = registry.GetTemplate(api.AgentTypeArchitect)
	if err == nil {
		t.Error("Expected error for non-existent template")
	}
}

func TestFactoryCreateAgent(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	template := &AgentTemplate{
		Type:          api.AgentTypeCoder,
		SystemPrompt:  "You are a coding assistant",
		Model:         "mock-model",
		Skills:        []string{"coding"},
		MaxConcurrent: 5,
		Timeout:       30,
		MaxRetries:    3,
		ResourceLimits: api.ResourceLimits{
			MaxTokensPerTask: 2000,
			MaxTasksPerHour:  100,
		},
	}

	factory.registry.RegisterTemplate(api.AgentTypeCoder, template)

	ctx := context.Background()
	agent, err := factory.CreateAgent(ctx, api.AgentTypeCoder, nil)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Expected agent to be created")
	}

	if agent.Type() != api.AgentTypeCoder {
		t.Errorf("Expected type coder, got %s", agent.Type())
	}

	if agent.Status() != api.AgentStatusReady {
		t.Errorf("Expected status ready, got %s", agent.Status())
	}
}

func TestFactoryDestroyAgent(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	template := &AgentTemplate{
		Type:         api.AgentTypeCoder,
		SystemPrompt: "You are a coding assistant",
		Model:        "mock-model",
		Skills:       []string{"coding"},
		ResourceLimits: api.ResourceLimits{
			MaxTokensPerTask: 2000,
		},
	}

	factory.registry.RegisterTemplate(api.AgentTypeCoder, template)

	ctx := context.Background()
	agent, err := factory.CreateAgent(ctx, api.AgentTypeCoder, nil)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	agentID := agent.ID()

	err = factory.DestroyAgent(ctx, agentID)
	if err != nil {
		t.Fatalf("Failed to destroy agent: %v", err)
	}

	_, err = factory.GetAgent(agentID)
	if err == nil {
		t.Error("Expected error when getting destroyed agent")
	}
}

func TestFactoryListAgents(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	template := &AgentTemplate{
		Type:         api.AgentTypeCoder,
		SystemPrompt: "You are a coding assistant",
		Model:        "mock-model",
		ResourceLimits: api.ResourceLimits{
			MaxTokensPerTask: 2000,
		},
	}

	factory.registry.RegisterTemplate(api.AgentTypeCoder, template)

	ctx := context.Background()
	_, _ = factory.CreateAgent(ctx, api.AgentTypeCoder, nil)
	_, _ = factory.CreateAgent(ctx, api.AgentTypeCoder, nil)

	agents := factory.ListAgents(nil)
	if len(agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(agents))
	}

	filtered := factory.ListAgents(&AgentFilter{
		Type: api.AgentTypeCoder,
	})
	if len(filtered) != 2 {
		t.Errorf("Expected 2 coder agents, got %d", len(filtered))
	}
}

func TestFactoryCreatePool(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	template := &AgentTemplate{
		Type:         api.AgentTypeCoder,
		SystemPrompt: "You are a coding assistant",
		Model:        "mock-model",
		ResourceLimits: api.ResourceLimits{
			MaxTokensPerTask: 2000,
		},
	}

	factory.registry.RegisterTemplate(api.AgentTypeCoder, template)

	ctx := context.Background()
	pool, err := factory.CreatePool(ctx, api.AgentTypeCoder, 2, 5)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	if pool == nil {
		t.Fatal("Expected pool to be created")
	}

	if pool.Type != api.AgentTypeCoder {
		t.Errorf("Expected type coder, got %s", pool.Type)
	}

	if pool.MinSize != 2 {
		t.Errorf("Expected min size 2, got %d", pool.MinSize)
	}

	if pool.MaxSize != 5 {
		t.Errorf("Expected max size 5, got %d", pool.MaxSize)
	}
}

func TestFactoryScalePool(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	template := &AgentTemplate{
		Type:         api.AgentTypeCoder,
		SystemPrompt: "You are a coding assistant",
		Model:        "mock-model",
		ResourceLimits: api.ResourceLimits{
			MaxTokensPerTask: 2000,
		},
	}

	factory.registry.RegisterTemplate(api.AgentTypeCoder, template)

	ctx := context.Background()
	_, err := factory.CreatePool(ctx, api.AgentTypeCoder, 2, 5)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}

	err = factory.ScalePool(ctx, api.AgentTypeCoder, 3)
	if err != nil {
		t.Fatalf("Failed to scale pool: %v", err)
	}

	pool, err := factory.GetPool(api.AgentTypeCoder)
	if err != nil {
		t.Fatalf("Failed to get pool: %v", err)
	}

	if len(pool.agents) != 3 {
		t.Errorf("Expected 3 agents in pool, got %d", len(pool.agents))
	}
}

func TestAgentFilterMatch(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	template := &AgentTemplate{
		Type:         api.AgentTypeCoder,
		SystemPrompt: "You are a coding assistant",
		Model:        "mock-model",
		ResourceLimits: api.ResourceLimits{
			MaxTokensPerTask: 2000,
		},
	}

	factory.registry.RegisterTemplate(api.AgentTypeCoder, template)

	ctx := context.Background()
	agent, _ := factory.CreateAgent(ctx, api.AgentTypeCoder, nil)

	filter := &AgentFilter{
		Type:   api.AgentTypeCoder,
		Status: api.AgentStatusReady,
	}

	if !filter.Match(agent) {
		t.Error("Expected filter to match agent")
	}

	filter.Type = api.AgentTypeArchitect
	if filter.Match(agent) {
		t.Error("Expected filter not to match agent with different type")
	}
}
