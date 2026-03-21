package providers

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestInMemoryCache_GetSet(t *testing.T) {
	cache := NewInMemoryCache(5 * time.Minute)

	resp := &api.ChatResponse{
		ID:    "test-123",
		Model: "gpt-4",
		Choices: []api.Choice{
			{
				Message: api.Message{
					Role:    "assistant",
					Content: "Hello, world!",
				},
			},
		},
	}

	err := cache.Set(context.Background(), "test-key", resp)
	assert.NoError(t, err)

	cached, found := cache.Get(context.Background(), "test-key")
	assert.True(t, found)
	assert.Equal(t, resp.ID, cached.ID)
}

func TestInMemoryCache_NotFound(t *testing.T) {
	cache := NewInMemoryCache(5 * time.Minute)

	_, found := cache.Get(context.Background(), "nonexistent")
	assert.False(t, found)
}

func TestInMemoryCache_Expiration(t *testing.T) {
	cache := NewInMemoryCache(10 * time.Millisecond)

	resp := &api.ChatResponse{
		ID: "test-123",
	}

	err := cache.Set(context.Background(), "test-key", resp)
	assert.NoError(t, err)

	_, found := cache.Get(context.Background(), "test-key")
	assert.True(t, found)

	time.Sleep(20 * time.Millisecond)

	_, found = cache.Get(context.Background(), "test-key")
	assert.False(t, found)
}

func TestInMemoryCache_Delete(t *testing.T) {
	cache := NewInMemoryCache(5 * time.Minute)

	resp := &api.ChatResponse{
		ID: "test-123",
	}

	err := cache.Set(context.Background(), "test-key", resp)
	assert.NoError(t, err)

	err = cache.Delete(context.Background(), "test-key")
	assert.NoError(t, err)

	_, found := cache.Get(context.Background(), "test-key")
	assert.False(t, found)
}

func TestInMemoryCache_Clear(t *testing.T) {
	cache := NewInMemoryCache(5 * time.Minute)

	resp := &api.ChatResponse{
		ID: "test-123",
	}

	cache.Set(context.Background(), "key1", resp)
	cache.Set(context.Background(), "key2", resp)

	err := cache.Clear(context.Background())
	assert.NoError(t, err)

	_, found1 := cache.Get(context.Background(), "key1")
	_, found2 := cache.Get(context.Background(), "key2")
	assert.False(t, found1)
	assert.False(t, found2)
}

func TestInMemoryCache_Stats(t *testing.T) {
	cache := NewInMemoryCache(5 * time.Minute)

	resp := &api.ChatResponse{
		ID: "test-123",
	}

	cache.Set(context.Background(), "key1", resp)
	cache.Set(context.Background(), "key2", resp)

	stats := cache.Stats()
	assert.Equal(t, 2, stats.TotalEntries)
}

func TestCachedProvider_CacheHit(t *testing.T) {
	mockProvider := &TestProvider{
		responses: []*api.ChatResponse{
			{
				ID: "test-123",
				Choices: []api.Choice{
					{
						Message: api.Message{
							Role:    "assistant",
							Content: "Cached response",
						},
					},
				},
			},
		},
	}

	cache := NewInMemoryCache(5 * time.Minute)
	cachedProvider := NewCachedProvider(mockProvider, cache)

	req := &api.ChatRequest{
		Model: "gpt-4",
		Messages: []api.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp1, err := cachedProvider.Chat(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Cached response", resp1.Choices[0].Message.Content)

	resp2, err := cachedProvider.Chat(context.Background(), req)
	require.NoError(t, err)
	assert.Equal(t, "Cached response", resp2.Choices[0].Message.Content)

	assert.Equal(t, 1, mockProvider.chatCallCount)
}

func TestCachedProvider_CacheDisabled(t *testing.T) {
	mockProvider := &TestProvider{
		responses: []*api.ChatResponse{
			{
				ID: "test-123",
				Choices: []api.Choice{
					{
						Message: api.Message{
							Role:    "assistant",
							Content: "Response",
						},
					},
				},
			},
			{
				ID: "test-456",
				Choices: []api.Choice{
					{
						Message: api.Message{
							Role:    "assistant",
							Content: "Response 2",
						},
					},
				},
			},
		},
	}

	cache := NewInMemoryCache(5 * time.Minute)
	cachedProvider := NewCachedProvider(mockProvider, cache)
	cachedProvider.enabled = false

	req := &api.ChatRequest{
		Model: "gpt-4",
		Messages: []api.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp1, err := cachedProvider.Chat(context.Background(), req)
	require.NoError(t, err)

	resp2, err := cachedProvider.Chat(context.Background(), req)
	require.NoError(t, err)

	assert.NotEqual(t, resp1.ID, resp2.ID)
	assert.Equal(t, 2, mockProvider.chatCallCount)
}

type TestProvider struct {
	responses       []*api.ChatResponse
	chatCallCount   int
	streamCallCount int
}

func (t *TestProvider) Name() string {
	return "test"
}

func (t *TestProvider) Models() []api.ModelInfo {
	return []api.ModelInfo{}
}

func (t *TestProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	t.chatCallCount++
	if t.chatCallCount > len(t.responses) {
		return &api.ChatResponse{}, nil
	}
	return t.responses[t.chatCallCount-1], nil
}

func (t *TestProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	t.streamCallCount++
	ch := make(chan api.ChatStreamEvent)
	close(ch)
	return ch, nil
}

func (t *TestProvider) SupportsStreaming() bool {
	return true
}

func (t *TestProvider) SupportsFunctionCalling() bool {
	return true
}

func (t *TestProvider) SupportsVision() bool {
	return false
}

func (t *TestProvider) SupportsAudio() bool {
	return false
}

func (t *TestProvider) MaxTokens(model string) int {
	return 4096
}

func (t *TestProvider) Configure(config *api.ProviderConfig) error {
	return nil
}

func (t *TestProvider) Metrics() *api.ProviderMetrics {
	return &api.ProviderMetrics{}
}
