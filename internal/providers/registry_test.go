package providers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestNewProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()
	assert.NotNil(t, registry)
}

func TestProviderRegistry_Register(t *testing.T) {
	registry := NewProviderRegistry()

	mockProvider := &MockProvider{}
	registry.Register("mock", mockProvider)

	p, err := registry.Get("mock")
	assert.NoError(t, err)
	assert.Equal(t, mockProvider, p)
}

func TestProviderRegistry_Get_Default(t *testing.T) {
	registry := NewProviderRegistry()

	err := registry.SetDefault("anthropic")
	require.NoError(t, err)

	p, err := registry.Get("")
	assert.NoError(t, err)
	assert.NotNil(t, p)
}

func TestProviderRegistry_Get_NotFound(t *testing.T) {
	registry := NewProviderRegistry()

	_, err := registry.Get("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")
}

func TestProviderRegistry_Configure(t *testing.T) {
	registry := NewProviderRegistry()

	config := &api.ProviderConfig{
		APIKey: "test-key",
	}

	err := registry.Configure("anthropic", config)
	assert.NoError(t, err)
}

func TestProviderRegistry_Configure_NotFound(t *testing.T) {
	registry := NewProviderRegistry()

	config := &api.ProviderConfig{
		APIKey: "test-key",
	}

	err := registry.Configure("nonexistent", config)
	assert.Error(t, err)
}

func TestProviderRegistry_List(t *testing.T) {
	registry := NewProviderRegistry()

	names := registry.List()
	assert.NotEmpty(t, names)
	assert.Contains(t, names, "anthropic")
	assert.Contains(t, names, "openai")
}

func TestProviderRegistry_SetDefault(t *testing.T) {
	registry := NewProviderRegistry()

	err := registry.SetDefault("openai")
	assert.NoError(t, err)

	err = registry.SetDefault("nonexistent")
	assert.Error(t, err)
}

func TestGlobalRegistry(t *testing.T) {
	registry := GlobalRegistry()
	assert.NotNil(t, registry)

	sameRegistry := GlobalRegistry()
	assert.Same(t, registry, sameRegistry)
}

type MockProvider struct {
	config *api.ProviderConfig
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) Models() []api.ModelInfo {
	return []api.ModelInfo{}
}

func (m *MockProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return &api.ChatResponse{}, nil
}

func (m *MockProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	ch := make(chan api.ChatStreamEvent)
	close(ch)
	return ch, nil
}

func (m *MockProvider) SupportsStreaming() bool {
	return true
}

func (m *MockProvider) SupportsFunctionCalling() bool {
	return true
}

func (m *MockProvider) SupportsVision() bool {
	return false
}

func (m *MockProvider) SupportsAudio() bool {
	return false
}

func (m *MockProvider) MaxTokens(model string) int {
	return 4096
}

func (m *MockProvider) Configure(config *api.ProviderConfig) error {
	m.config = config
	return nil
}

func (m *MockProvider) Metrics() *api.ProviderMetrics {
	return &api.ProviderMetrics{}
}
