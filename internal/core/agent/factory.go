package agent

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type AgentFactory struct {
	registry *AgentRegistry
	provider api.LLMProvider
	skills   *SkillManager
	mcp      *MCPConnector
	mu       sync.RWMutex
	agents   map[string]api.Agent
	pools    map[api.AgentType]*AgentPool
}

type AgentPool struct {
	Type      api.AgentType
	MinSize   int
	MaxSize   int
	agents    []api.Agent
	available chan api.Agent
	mu        sync.RWMutex
}

type FactoryConfig struct {
	Provider api.LLMProvider
	Skills   *SkillManager
	MCP      *MCPConnector
}

type AgentTemplate struct {
	Type           api.AgentType
	SystemPrompt   string
	Model          string
	Skills         []string
	MaxConcurrent  int
	Timeout        int
	MaxRetries     int
	ResourceLimits api.ResourceLimits
}

type AgentFilter struct {
	Type   api.AgentType
	Status api.AgentStatus
}

type AgentRegistry struct {
	mu        sync.RWMutex
	templates map[api.AgentType]*AgentTemplate
}

func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		templates: make(map[api.AgentType]*AgentTemplate),
	}
}

func (r *AgentRegistry) RegisterTemplate(agentType api.AgentType, template *AgentTemplate) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.templates[agentType] = template
}

func (r *AgentRegistry) GetTemplate(agentType api.AgentType) (*AgentTemplate, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	template, ok := r.templates[agentType]
	if !ok {
		return nil, fmt.Errorf("template not found for agent type: %s", agentType)
	}

	return template, nil
}

func (r *AgentRegistry) ListTemplates() map[api.AgentType]*AgentTemplate {
	r.mu.RLock()
	defer r.mu.RUnlock()

	templates := make(map[api.AgentType]*AgentTemplate)
	for k, v := range r.templates {
		templates[k] = v
	}
	return templates
}

func NewAgentFactory(cfg *FactoryConfig) *AgentFactory {
	return &AgentFactory{
		registry: NewAgentRegistry(),
		provider: cfg.Provider,
		skills:   cfg.Skills,
		mcp:      cfg.MCP,
		agents:   make(map[string]api.Agent),
		pools:    make(map[api.AgentType]*AgentPool),
	}
}

func (f *AgentFactory) CreateAgent(
	ctx context.Context,
	agentType api.AgentType,
	config *api.AgentConfig,
) (api.Agent, error) {
	template, err := f.registry.GetTemplate(agentType)
	if err != nil {
		return nil, fmt.Errorf("get template: %w", err)
	}

	cfg := f.mergeConfig(template, config)

	agentID := generateID()
	agent := NewBaseAgent(agentID, agentType, cfg, f.provider, f.skills, f.mcp)

	if err := agent.Initialize(ctx, cfg); err != nil {
		return nil, fmt.Errorf("initialize agent: %w", err)
	}

	f.mu.Lock()
	f.agents[agent.ID()] = agent
	f.mu.Unlock()

	return agent, nil
}

func (f *AgentFactory) DestroyAgent(ctx context.Context, id string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	agent, ok := f.agents[id]
	if !ok {
		return fmt.Errorf("agent not found: %s", id)
	}

	if err := agent.Stop(ctx); err != nil {
		return fmt.Errorf("stop agent: %w", err)
	}

	delete(f.agents, id)

	f.removeFromPools(id)

	return nil
}

func (f *AgentFactory) GetAgent(id string) (api.Agent, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	agent, ok := f.agents[id]
	if !ok {
		return nil, fmt.Errorf("agent not found: %s", id)
	}
	return agent, nil
}

func (f *AgentFactory) ListAgents(filter *AgentFilter) []api.Agent {
	f.mu.RLock()
	defer f.mu.RUnlock()

	var result []api.Agent
	for _, agent := range f.agents {
		if filter == nil || filter.Match(agent) {
			result = append(result, agent)
		}
	}
	return result
}

func (f *AgentFactory) CreatePool(
	ctx context.Context,
	agentType api.AgentType,
	minSize, maxSize int,
) (*AgentPool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.pools[agentType]; exists {
		return nil, fmt.Errorf("pool already exists for type: %s", agentType)
	}

	pool := &AgentPool{
		Type:      agentType,
		MinSize:   minSize,
		MaxSize:   maxSize,
		agents:    make([]api.Agent, 0, maxSize),
		available: make(chan api.Agent, maxSize),
	}

	f.pools[agentType] = pool

	f.mu.Unlock()
	err := f.ScalePool(ctx, agentType, minSize)
	f.mu.Lock()

	if err != nil {
		delete(f.pools, agentType)
		return nil, fmt.Errorf("scale pool: %w", err)
	}

	return pool, nil
}

func (f *AgentFactory) GetPool(agentType api.AgentType) (*AgentPool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	pool, ok := f.pools[agentType]
	if !ok {
		return nil, fmt.Errorf("pool not found for type: %s", agentType)
	}
	return pool, nil
}

func (f *AgentFactory) ScalePool(
	ctx context.Context,
	agentType api.AgentType,
	targetSize int,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	pool, ok := f.pools[agentType]
	if !ok {
		return fmt.Errorf("pool not found for type: %s", agentType)
	}

	if targetSize < pool.MinSize {
		return fmt.Errorf("target size %d below minimum %d", targetSize, pool.MinSize)
	}

	if targetSize > pool.MaxSize {
		return fmt.Errorf("target size %d above maximum %d", targetSize, pool.MaxSize)
	}

	pool.mu.Lock()
	current := len(pool.agents)
	pool.mu.Unlock()

	if targetSize > current {
		f.mu.Unlock()
		defer f.mu.Lock()

		for i := 0; i < targetSize-current; i++ {
			agent, err := f.CreateAgent(ctx, agentType, nil)
			if err != nil {
				return fmt.Errorf("create agent: %w", err)
			}

			pool.mu.Lock()
			pool.agents = append(pool.agents, agent)
			pool.mu.Unlock()

			pool.available <- agent
		}
	} else if targetSize < current {
		f.mu.Unlock()
		defer f.mu.Lock()

		for i := 0; i < current-targetSize; i++ {
			pool.mu.Lock()
			if len(pool.agents) == 0 {
				pool.mu.Unlock()
				break
			}
			agent := pool.agents[len(pool.agents)-1]
			pool.agents = pool.agents[:len(pool.agents)-1]
			pool.mu.Unlock()

			if err := f.DestroyAgent(ctx, agent.ID()); err != nil {
				return fmt.Errorf("destroy agent: %w", err)
			}
		}
	}

	return nil
}

func (f *AgentFactory) GetAvailableAgent(agentType api.AgentType) (api.Agent, error) {
	f.mu.RLock()
	pool, ok := f.pools[agentType]
	f.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("pool not found for type: %s", agentType)
	}

	select {
	case agent := <-pool.available:
		return agent, nil
	default:
		return nil, fmt.Errorf("no available agents of type: %s", agentType)
	}
}

func (f *AgentFactory) ReturnAgent(agent api.Agent) error {
	f.mu.RLock()
	pool, ok := f.pools[agent.Type()]
	f.mu.RUnlock()

	if !ok {
		return nil
	}

	select {
	case pool.available <- agent:
		return nil
	default:
		return fmt.Errorf("pool full")
	}
}

func (f *AgentFactory) mergeConfig(template *AgentTemplate, config *api.AgentConfig) *api.AgentConfig {
	if config == nil {
		config = &api.AgentConfig{}
	}

	if config.ID == "" {
		config.ID = generateID()
	}

	if config.Type == "" {
		config.Type = template.Type
	}

	if config.Model == "" {
		config.Model = template.Model
	}

	if config.SystemPrompt == "" {
		config.SystemPrompt = template.SystemPrompt
	}

	if len(config.Skills) == 0 {
		config.Skills = template.Skills
	}

	if config.MaxConcurrent == 0 {
		config.MaxConcurrent = template.MaxConcurrent
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = template.MaxRetries
	}

	if config.ResourceLimits.MaxTokensPerTask == 0 {
		config.ResourceLimits.MaxTokensPerTask = template.ResourceLimits.MaxTokensPerTask
	}

	if config.ResourceLimits.MaxTasksPerHour == 0 {
		config.ResourceLimits.MaxTasksPerHour = template.ResourceLimits.MaxTasksPerHour
	}

	return config
}

func (f *AgentFactory) removeFromPools(agentID string) {
	for _, pool := range f.pools {
		pool.mu.Lock()
		for i, agent := range pool.agents {
			if agent.ID() == agentID {
				pool.agents = append(pool.agents[:i], pool.agents[i+1:]...)
				break
			}
		}
		pool.mu.Unlock()
	}
}

func (af *AgentFilter) Match(agent api.Agent) bool {
	if af.Type != "" && agent.Type() != af.Type {
		return false
	}
	if af.Status != "" && agent.Status() != af.Status {
		return false
	}
	return true
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("agent-%x", b)
}

type timeProvider interface {
	Now() time.Time
}

type defaultTimeProvider struct{}

func (d *defaultTimeProvider) Now() time.Time {
	return time.Now()
}

var timeInst timeProvider = &defaultTimeProvider{}

func now() time.Time {
	return timeInst.Now()
}

func setTimeProvider(tp timeProvider) {
	timeInst = tp
}
