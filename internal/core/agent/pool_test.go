package agent

import (
	"context"
	"testing"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewAgentPoolManager(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	manager := NewAgentPoolManager(factory)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.factory != factory {
		t.Error("Expected factory to be set")
	}
}

func TestPoolManagerRegisterConfig(t *testing.T) {
	provider := &mockProvider{}
	skills := NewSkillManager()
	mcp := NewMCPConnector()

	factory := NewAgentFactory(&FactoryConfig{
		Provider: provider,
		Skills:   skills,
		MCP:      mcp,
	})

	manager := NewAgentPoolManager(factory)

	config := &PoolConfig{
		Type:              api.AgentTypeCoder,
		MinSize:           2,
		MaxSize:           5,
		TargetSize:        3,
		ScaleUpCooldown:   30 * time.Second,
		ScaleDownCooldown: 60 * time.Second,
		AutoScale:         true,
	}

	manager.RegisterPoolConfig(config)

	if len(manager.configs) != 1 {
		t.Errorf("Expected 1 config, got %d", len(manager.configs))
	}
}

func TestPoolManagerInitializePools(t *testing.T) {
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

	manager := NewAgentPoolManager(factory)

	config := &PoolConfig{
		Type:              api.AgentTypeCoder,
		MinSize:           2,
		MaxSize:           5,
		TargetSize:        3,
		ScaleUpCooldown:   30 * time.Second,
		ScaleDownCooldown: 60 * time.Second,
		AutoScale:         true,
	}

	manager.RegisterPoolConfig(config)

	ctx := context.Background()
	err := manager.InitializePools(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize pools: %v", err)
	}

	if len(manager.pools) != 1 {
		t.Errorf("Expected 1 pool, got %d", len(manager.pools))
	}

	pool, ok := manager.pools[api.AgentTypeCoder]
	if !ok {
		t.Fatal("Expected coder pool to exist")
	}

	if len(pool.Agents) != 2 {
		t.Errorf("Expected 2 agents, got %d", len(pool.Agents))
	}
}

func TestPoolManagerGetAgent(t *testing.T) {
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

	manager := NewAgentPoolManager(factory)

	config := &PoolConfig{
		Type:       api.AgentTypeCoder,
		MinSize:    2,
		MaxSize:    5,
		TargetSize: 3,
		AutoScale:  false,
	}

	manager.RegisterPoolConfig(config)

	ctx := context.Background()
	err := manager.InitializePools(ctx)
	if err != nil {
		t.Fatalf("Failed to initialize pools: %v", err)
	}

	agent, err := manager.GetAgent(ctx, api.AgentTypeCoder)
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if agent == nil {
		t.Fatal("Expected agent to be returned")
	}

	if agent.Type() != api.AgentTypeCoder {
		t.Errorf("Expected type coder, got %s", agent.Type())
	}

	stats, _ := manager.GetStats(api.AgentTypeCoder)
	if stats.Available != 1 {
		t.Errorf("Expected 1 available agents after assignment, got %d", stats.Available)
	}
}

func TestPoolManagerReturnAgent(t *testing.T) {
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

	manager := NewAgentPoolManager(factory)

	config := &PoolConfig{
		Type:      api.AgentTypeCoder,
		MinSize:   2,
		MaxSize:   5,
		AutoScale: false,
	}

	manager.RegisterPoolConfig(config)

	ctx := context.Background()
	_ = manager.InitializePools(ctx)

	agent, _ := manager.GetAgent(ctx, api.AgentTypeCoder)

	err := manager.ReturnAgent(agent)
	if err != nil {
		t.Fatalf("Failed to return agent: %v", err)
	}

	stats, _ := manager.GetStats(api.AgentTypeCoder)
	if stats.Available != 2 {
		t.Errorf("Expected 2 available agents after return, got %d", stats.Available)
	}
}

func TestPoolManagerGetStats(t *testing.T) {
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

	manager := NewAgentPoolManager(factory)

	config := &PoolConfig{
		Type:      api.AgentTypeCoder,
		MinSize:   2,
		MaxSize:   5,
		AutoScale: false,
	}

	manager.RegisterPoolConfig(config)

	ctx := context.Background()
	_ = manager.InitializePools(ctx)

	stats, err := manager.GetStats(api.AgentTypeCoder)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats.TotalAgents != 2 {
		t.Errorf("Expected 2 total agents, got %d", stats.TotalAgents)
	}

	if stats.Available != 2 {
		t.Errorf("Expected 2 available agents, got %d", stats.Available)
	}
}

func TestPoolManagerGetAllStats(t *testing.T) {
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

	manager := NewAgentPoolManager(factory)

	config := &PoolConfig{
		Type:      api.AgentTypeCoder,
		MinSize:   2,
		MaxSize:   5,
		AutoScale: false,
	}

	manager.RegisterPoolConfig(config)

	ctx := context.Background()
	_ = manager.InitializePools(ctx)

	allStats := manager.GetAllStats()

	if len(allStats) != 1 {
		t.Errorf("Expected stats for 1 pool type, got %d", len(allStats))
	}

	stats, ok := allStats[api.AgentTypeCoder]
	if !ok {
		t.Fatal("Expected coder stats to exist")
	}

	if stats.TotalAgents != 2 {
		t.Errorf("Expected 2 total agents, got %d", stats.TotalAgents)
	}
}

func TestDefaultPoolConfigs(t *testing.T) {
	configs := DefaultPoolConfigs()

	expectedTypes := []api.AgentType{
		api.AgentTypeArchitect,
		api.AgentTypeCoder,
		api.AgentTypeTester,
		api.AgentTypeReviewer,
		api.AgentTypeResearcher,
		api.AgentTypeCoordinator,
	}

	for _, agentType := range expectedTypes {
		config, ok := configs[agentType]
		if !ok {
			t.Errorf("Expected config for %s", agentType)
		}

		if config.Type != agentType {
			t.Errorf("Expected type %s, got %s", agentType, config.Type)
		}

		if config.MinSize <= 0 {
			t.Errorf("Expected min size > 0 for %s", agentType)
		}

		if config.MaxSize < config.MinSize {
			t.Errorf("Expected max size >= min size for %s", agentType)
		}
	}
}
