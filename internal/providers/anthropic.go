package providers

import (
	"context"

	"github.com/swarm-ai/swarm/pkg/api"
)

// AnthropicProvider implements Anthropic API
type AnthropicProvider struct {
	*BaseProvider
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider() *AnthropicProvider {
	models := []api.ModelInfo{
		{
			ID:                "claude-sonnet-4-20250514",
			Alias:             "claude-sonnet-4",
			MaxTokens:         200000,
			MaxOutputTokens:   8192,
			SupportsVision:    true,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   3.0,
			OutputPricePer1K:  15.0,
			ContextWindow:     200000,
		},
		{
			ID:                "claude-3-5-sonnet-20241022",
			Alias:             "claude-3-5-sonnet",
			MaxTokens:         200000,
			MaxOutputTokens:   8192,
			SupportsVision:    true,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   3.0,
			OutputPricePer1K:  15.0,
			ContextWindow:     200000,
		},
		{
			ID:                "claude-3-opus-20240229",
			Alias:             "claude-opus",
			MaxTokens:         200000,
			MaxOutputTokens:   4096,
			SupportsVision:    true,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   15.0,
			OutputPricePer1K:  75.0,
			ContextWindow:     200000,
		},
		{
			ID:                "claude-3-haiku-20240307",
			Alias:             "claude-haiku",
			MaxTokens:         200000,
			MaxOutputTokens:   4096,
			SupportsVision:    true,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.25,
			OutputPricePer1K:  1.25,
			ContextWindow:     200000,
		},
	}

	return &AnthropicProvider{
		BaseProvider: NewBaseProvider("anthropic", models, &AnthropicNormalizer{}),
	}
}

// Chat sends a chat request to Anthropic
func (p *AnthropicProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

// StreamChat sends a streaming chat request to Anthropic
func (p *AnthropicProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}
