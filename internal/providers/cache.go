package providers

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

// Cache interface for provider response caching
type Cache interface {
	Get(ctx context.Context, key string) (*api.ChatResponse, bool)
	Set(ctx context.Context, key string, resp *api.ChatResponse) error
	Delete(ctx context.Context, key string) error
	Clear(ctx context.Context) error
}

// InMemoryCache is a simple in-memory cache
type InMemoryCache struct {
	mu    sync.RWMutex
	store map[string]*cacheEntry
	ttl   time.Duration
}

type cacheEntry struct {
	value     *api.ChatResponse
	expiresAt time.Time
	hits      int
}

// NewInMemoryCache creates a new in-memory cache
func NewInMemoryCache(ttl time.Duration) *InMemoryCache {
	return &InMemoryCache{
		store: make(map[string]*cacheEntry),
		ttl:   ttl,
	}
}

// Get retrieves a cached response
func (c *InMemoryCache) Get(ctx context.Context, key string) (*api.ChatResponse, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.store[key]
	if !ok {
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		// Delete expired entry to prevent memory leak
		delete(c.store, key)
		return nil, false
	}

	entry.hits++
	return entry.value, true
}

// Set stores a response in the cache
func (c *InMemoryCache) Set(ctx context.Context, key string, resp *api.ChatResponse) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.store[key] = &cacheEntry{
		value:     resp,
		expiresAt: time.Now().Add(c.ttl),
		hits:      0,
	}

	return nil
}

// Delete removes a cached response
func (c *InMemoryCache) Delete(ctx context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.store, key)
	return nil
}

// Clear clears all cached responses
func (c *InMemoryCache) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store = make(map[string]*cacheEntry)
	return nil
}

// Stats returns cache statistics
func (c *InMemoryCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		TotalEntries: len(c.store),
	}

	for _, entry := range c.store {
		stats.TotalHits += entry.hits
	}

	return stats
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalEntries int
	TotalHits    int
}

// CachedProvider wraps a provider with caching
type CachedProvider struct {
	provider api.LLMProvider
	cache    Cache
	enabled  bool
}

// NewCachedProvider creates a new cached provider
func NewCachedProvider(provider api.LLMProvider, cache Cache) *CachedProvider {
	return &CachedProvider{
		provider: provider,
		cache:    cache,
		enabled:  true,
	}
}

// Name returns the provider name
func (p *CachedProvider) Name() string {
	return p.provider.Name()
}

// Models returns available models
func (p *CachedProvider) Models() []api.ModelInfo {
	return p.provider.Models()
}

// Chat with caching
func (p *CachedProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	if !p.enabled {
		return p.provider.Chat(ctx, req)
	}

	key := p.cacheKey(req)
	if resp, ok := p.cache.Get(ctx, key); ok {
		return resp, nil
	}

	resp, err := p.provider.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	if err := p.cache.Set(ctx, key, resp); err != nil {
		return resp, nil
	}

	return resp, nil
}

// StreamChat passes through to underlying provider (no caching for streaming)
func (p *CachedProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.provider.StreamChat(ctx, req)
}

// SupportsStreaming checks if streaming is supported
func (p *CachedProvider) SupportsStreaming() bool {
	return p.provider.SupportsStreaming()
}

// SupportsFunctionCalling checks if function calling is supported
func (p *CachedProvider) SupportsFunctionCalling() bool {
	return p.provider.SupportsFunctionCalling()
}

// SupportsVision checks if vision is supported
func (p *CachedProvider) SupportsVision() bool {
	return p.provider.SupportsVision()
}

// SupportsAudio checks if audio is supported
func (p *CachedProvider) SupportsAudio() bool {
	return p.provider.SupportsAudio()
}

// MaxTokens returns the max tokens for a model
func (p *CachedProvider) MaxTokens(model string) int {
	return p.provider.MaxTokens(model)
}

// Configure configures the provider
func (p *CachedProvider) Configure(config *api.ProviderConfig) error {
	return p.provider.Configure(config)
}

// Metrics returns provider metrics
func (p *CachedProvider) Metrics() *api.ProviderMetrics {
	return p.provider.Metrics()
}

// cacheKey generates a cache key for a request
func (p *CachedProvider) cacheKey(req *api.ChatRequest) string {
	h := sha256.New()

	if err := json.NewEncoder(h).Encode(struct {
		Model       string
		Messages    []api.Message
		Tools       []api.Tool
		Temperature float64
		TopP        float64
		MaxTokens   int
	}{
		Model:       req.Model,
		Messages:    req.Messages,
		Tools:       req.Tools,
		Temperature: safeFloat64(req.Temperature),
		TopP:        safeFloat64(req.TopP),
		MaxTokens:   req.MaxTokens,
	}); err != nil {
		// Fallback to a simple key based on model and message count
		return fmt.Sprintf("%s:%d", req.Model, len(req.Messages))
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func safeFloat64(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}
