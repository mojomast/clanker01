package providers

import (
	"context"
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/pkg/api"
)

// BaseProvider provides common functionality for all providers
type BaseProvider struct {
	name       string
	models     []api.ModelInfo
	config     *api.ProviderConfig // config is stored for use when real HTTP calls are implemented
	normalizer Normalizer
	metrics    *api.ProviderMetrics
	mu         sync.RWMutex
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string, models []api.ModelInfo, normalizer Normalizer) *BaseProvider {
	return &BaseProvider{
		name:       name,
		models:     models,
		normalizer: normalizer,
		metrics: &api.ProviderMetrics{
			TotalRequests:     0,
			TotalTokens:       0,
			TotalPromptTokens: 0,
			TotalOutputTokens: 0,
			TotalCost:         0,
			Errors:            0,
		},
	}
}

// Name returns the provider name
func (p *BaseProvider) Name() string {
	return p.name
}

// Models returns available models
func (p *BaseProvider) Models() []api.ModelInfo {
	return p.models
}

// Configure configures the provider
func (p *BaseProvider) Configure(config *api.ProviderConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config = config
	return nil
}

// Metrics returns a copy of provider metrics
func (p *BaseProvider) Metrics() *api.ProviderMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()
	// Return a copy to prevent callers from mutating internal state
	m := *p.metrics
	return &m
}

// SupportsStreaming checks if streaming is supported
func (p *BaseProvider) SupportsStreaming() bool {
	return true
}

// SupportsFunctionCalling checks if function calling is supported
func (p *BaseProvider) SupportsFunctionCalling() bool {
	return true
}

// SupportsVision checks if vision is supported
func (p *BaseProvider) SupportsVision() bool {
	return true
}

// SupportsAudio checks if audio is supported
func (p *BaseProvider) SupportsAudio() bool {
	return false
}

// MaxTokens returns the max tokens for a model
func (p *BaseProvider) MaxTokens(model string) int {
	for _, m := range p.models {
		if m.ID == model || m.Alias == model {
			return m.MaxTokens
		}
	}
	return 4096
}

// recordMetrics records metrics for a request
func (p *BaseProvider) recordMetrics(usage *api.Usage, latencyMs float64, err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.TotalRequests++
	if usage != nil {
		p.metrics.TotalTokens += int64(usage.TotalTokens)
		p.metrics.TotalPromptTokens += int64(usage.PromptTokens)
		p.metrics.TotalOutputTokens += int64(usage.CompletionTokens)
	}
	if err != nil {
		p.metrics.Errors++
	}

	if p.metrics.TotalRequests == 1 {
		p.metrics.AvgLatencyMs = latencyMs
	} else {
		p.metrics.AvgLatencyMs = (p.metrics.AvgLatencyMs*float64(p.metrics.TotalRequests-1) + latencyMs) / float64(p.metrics.TotalRequests)
	}
}

// trackCost tracks cost for a request
func (p *BaseProvider) trackCost(model string, usage *api.Usage) float64 {
	if usage == nil {
		return 0
	}

	var cost float64
	for _, m := range p.models {
		if m.ID == model || m.Alias == model {
			promptCost := (float64(usage.PromptTokens) / 1000) * m.InputPricePer1K
			outputCost := (float64(usage.CompletionTokens) / 1000) * m.OutputPricePer1K
			cost = promptCost + outputCost
			break
		}
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.metrics.TotalCost += cost
	return cost
}

// Chat is the base implementation that should be overridden
func (p *BaseProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return nil, fmt.Errorf("Chat not implemented")
}

// StreamChat is the base implementation that should be overridden
func (p *BaseProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return nil, fmt.Errorf("StreamChat not implemented")
}

// WithRetry wraps a provider with retry logic
func WithRetry(provider api.LLMProvider, config RetryConfig) api.LLMProvider {
	return &RetryableProvider{
		provider: provider,
		config:   config,
	}
}

// WithCache wraps a provider with caching
func WithCache(provider api.LLMProvider, cache Cache) api.LLMProvider {
	return &CachedProvider{
		provider: provider,
		cache:    cache,
		enabled:  true,
	}
}
