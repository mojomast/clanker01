package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type AgentPoolManager struct {
	mu      sync.RWMutex
	factory *AgentFactory
	pools   map[api.AgentType]*ManagedPool
	configs map[api.AgentType]*PoolConfig
}

type ManagedPool struct {
	Type       api.AgentType
	Config     *PoolConfig
	Agents     map[string]api.Agent
	Available  map[string]bool
	Busy       map[string]bool
	mu         sync.RWMutex
	metrics    *PoolMetrics
	lastScaled time.Time
}

type PoolConfig struct {
	Type              api.AgentType
	MinSize           int
	MaxSize           int
	TargetSize        int
	ScaleUpCooldown   time.Duration
	ScaleDownCooldown time.Duration
	AutoScale         bool
}

type PoolMetrics struct {
	TotalRequests         int64
	SuccessfulAssignments int64
	FailedAssignments     int64
	AvgWaitTime           time.Duration
	ActiveAgents          int
	IdleAgents            int
}

type PoolStats struct {
	Type        api.AgentType
	TotalAgents int
	Available   int
	Busy        int
	Queued      int
	Metrics     *PoolMetrics
}

func NewAgentPoolManager(factory *AgentFactory) *AgentPoolManager {
	return &AgentPoolManager{
		factory: factory,
		pools:   make(map[api.AgentType]*ManagedPool),
		configs: make(map[api.AgentType]*PoolConfig),
	}
}

func (m *AgentPoolManager) RegisterPoolConfig(config *PoolConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[config.Type] = config
}

func (m *AgentPoolManager) InitializePools(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for agentType, config := range m.configs {
		pool := &ManagedPool{
			Type:      agentType,
			Config:    config,
			Agents:    make(map[string]api.Agent),
			Available: make(map[string]bool),
			Busy:      make(map[string]bool),
			metrics:   &PoolMetrics{},
		}

		for i := 0; i < config.MinSize; i++ {
			agent, err := m.factory.CreateAgent(ctx, agentType, nil)
			if err != nil {
				return fmt.Errorf("create agent for %s: %w", agentType, err)
			}
			pool.Agents[agent.ID()] = agent
			pool.Available[agent.ID()] = true
			pool.metrics.IdleAgents++
		}

		m.pools[agentType] = pool
	}

	return nil
}

func (m *AgentPoolManager) GetAgent(ctx context.Context, agentType api.AgentType) (api.Agent, error) {
	pool, err := m.getPool(agentType)
	if err != nil {
		return nil, err
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	for agentID, available := range pool.Available {
		if available {
			pool.Available[agentID] = false
			pool.Busy[agentID] = true
			pool.metrics.ActiveAgents++
			pool.metrics.IdleAgents--
			pool.metrics.TotalRequests++
			pool.metrics.SuccessfulAssignments++

			return pool.Agents[agentID], nil
		}
	}

	if pool.Config.AutoScale && len(pool.Agents) < pool.Config.MaxSize {
		if time.Since(pool.lastScaled) >= pool.Config.ScaleUpCooldown {
			newAgent, err := m.factory.CreateAgent(ctx, agentType, nil)
			if err == nil {
				pool.Agents[newAgent.ID()] = newAgent
				pool.Busy[newAgent.ID()] = true
				pool.metrics.ActiveAgents++
				pool.lastScaled = time.Now()

				return newAgent, nil
			}
		}
	}

	pool.metrics.FailedAssignments++
	return nil, fmt.Errorf("no available agents of type: %s", agentType)
}

func (m *AgentPoolManager) ReturnAgent(agent api.Agent) error {
	pool, err := m.getPool(agent.Type())
	if err != nil {
		return err
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	agentID := agent.ID()
	if busy, exists := pool.Busy[agentID]; exists && busy {
		pool.Busy[agentID] = false
		pool.Available[agentID] = true
		pool.metrics.ActiveAgents--
		pool.metrics.IdleAgents++
	}

	return nil
}

func (m *AgentPoolManager) GetStats(agentType api.AgentType) (*PoolStats, error) {
	pool, err := m.getPool(agentType)
	if err != nil {
		return nil, err
	}

	pool.mu.RLock()
	defer pool.mu.RUnlock()

	availableCount := 0
	for _, available := range pool.Available {
		if available {
			availableCount++
		}
	}

	busyCount := 0
	for _, busy := range pool.Busy {
		if busy {
			busyCount++
		}
	}

	return &PoolStats{
		Type:        pool.Type,
		TotalAgents: len(pool.Agents),
		Available:   availableCount,
		Busy:        busyCount,
		Metrics:     pool.metrics,
	}, nil
}

func (m *AgentPoolManager) GetAllStats() map[api.AgentType]*PoolStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[api.AgentType]*PoolStats)
	for agentType, pool := range m.pools {
		pool.mu.RLock()

		availableCount := 0
		for _, available := range pool.Available {
			if available {
				availableCount++
			}
		}

		busyCount := 0
		for _, busy := range pool.Busy {
			if busy {
				busyCount++
			}
		}

		stats[agentType] = &PoolStats{
			Type:        pool.Type,
			TotalAgents: len(pool.Agents),
			Available:   availableCount,
			Busy:        busyCount,
			Metrics:     pool.metrics,
		}
		pool.mu.RUnlock()
	}

	return stats
}

func (m *AgentPoolManager) ScalePool(ctx context.Context, agentType api.AgentType, targetSize int) error {
	pool, err := m.getPool(agentType)
	if err != nil {
		return err
	}

	if targetSize < pool.Config.MinSize {
		return fmt.Errorf("target size %d below minimum %d", targetSize, pool.Config.MinSize)
	}

	if targetSize > pool.Config.MaxSize {
		return fmt.Errorf("target size %d above maximum %d", targetSize, pool.Config.MaxSize)
	}

	pool.mu.Lock()
	defer pool.mu.Unlock()

	return m.factory.ScalePool(ctx, agentType, targetSize)
}

func (m *AgentPoolManager) AutoScale(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, pool := range m.pools {
		if !pool.Config.AutoScale {
			continue
		}

		pool.mu.Lock()

		available := len(pool.Available)
		total := len(pool.Agents)
		target := pool.Config.TargetSize

		if available < 2 && total < pool.Config.MaxSize {
			if time.Since(pool.lastScaled) >= pool.Config.ScaleUpCooldown {
				newAgent, err := m.factory.CreateAgent(ctx, pool.Type, nil)
				if err == nil {
					pool.Agents[newAgent.ID()] = newAgent
					pool.Available[newAgent.ID()] = true
					pool.metrics.IdleAgents++
					pool.lastScaled = time.Now()
				}
			}
		} else if available > target && total > pool.Config.MinSize {
			if time.Since(pool.lastScaled) >= pool.Config.ScaleDownCooldown {
				for agentID := range pool.Agents {
					if pool.Available[agentID] {
						if err := m.factory.DestroyAgent(ctx, agentID); err == nil {
							delete(pool.Agents, agentID)
							delete(pool.Available, agentID)
							pool.metrics.IdleAgents--
							pool.lastScaled = time.Now()
							break
						}
					}
				}
			}
		}

		pool.mu.Unlock()
	}

	return nil
}

func (m *AgentPoolManager) getPool(agentType api.AgentType) (*ManagedPool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	pool, ok := m.pools[agentType]
	if !ok {
		return nil, fmt.Errorf("pool not found for type: %s", agentType)
	}
	return pool, nil
}

func (m *AgentPoolManager) StartAutoScale(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = m.AutoScale(ctx)
		}
	}
}

func DefaultPoolConfigs() map[api.AgentType]*PoolConfig {
	return map[api.AgentType]*PoolConfig{
		api.AgentTypeArchitect: {
			Type:              api.AgentTypeArchitect,
			MinSize:           1,
			MaxSize:           3,
			TargetSize:        2,
			ScaleUpCooldown:   30 * time.Second,
			ScaleDownCooldown: 60 * time.Second,
			AutoScale:         true,
		},
		api.AgentTypeCoder: {
			Type:              api.AgentTypeCoder,
			MinSize:           2,
			MaxSize:           5,
			TargetSize:        3,
			ScaleUpCooldown:   10 * time.Second,
			ScaleDownCooldown: 30 * time.Second,
			AutoScale:         true,
		},
		api.AgentTypeTester: {
			Type:              api.AgentTypeTester,
			MinSize:           1,
			MaxSize:           3,
			TargetSize:        2,
			ScaleUpCooldown:   15 * time.Second,
			ScaleDownCooldown: 45 * time.Second,
			AutoScale:         true,
		},
		api.AgentTypeReviewer: {
			Type:              api.AgentTypeReviewer,
			MinSize:           1,
			MaxSize:           2,
			TargetSize:        1,
			ScaleUpCooldown:   20 * time.Second,
			ScaleDownCooldown: 60 * time.Second,
			AutoScale:         true,
		},
		api.AgentTypeResearcher: {
			Type:              api.AgentTypeResearcher,
			MinSize:           1,
			MaxSize:           2,
			TargetSize:        1,
			ScaleUpCooldown:   30 * time.Second,
			ScaleDownCooldown: 60 * time.Second,
			AutoScale:         true,
		},
		api.AgentTypeCoordinator: {
			Type:              api.AgentTypeCoordinator,
			MinSize:           1,
			MaxSize:           1,
			TargetSize:        1,
			ScaleUpCooldown:   0,
			ScaleDownCooldown: 0,
			AutoScale:         false,
		},
	}
}
