package providers

import (
	"context"

	"github.com/swarm-ai/swarm/pkg/api"
)

// OpenAIProvider implements OpenAI API
type OpenAIProvider struct {
	*BaseProvider
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider() *OpenAIProvider {
	models := []api.ModelInfo{
		{
			ID:                "gpt-4o-2024-11-20",
			Alias:             "gpt-4o",
			MaxTokens:         128000,
			MaxOutputTokens:   4096,
			SupportsVision:    true,
			SupportsAudio:     true,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   2.5,
			OutputPricePer1K:  10.0,
			ContextWindow:     128000,
		},
		{
			ID:                "gpt-4o-mini-2024-07-18",
			Alias:             "gpt-4o-mini",
			MaxTokens:         128000,
			MaxOutputTokens:   16384,
			SupportsVision:    true,
			SupportsAudio:     true,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.15,
			OutputPricePer1K:  0.6,
			ContextWindow:     128000,
		},
		{
			ID:                "gpt-4-turbo-2024-04-09",
			Alias:             "gpt-4-turbo",
			MaxTokens:         128000,
			MaxOutputTokens:   4096,
			SupportsVision:    true,
			SupportsAudio:     true,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   10.0,
			OutputPricePer1K:  30.0,
			ContextWindow:     128000,
		},
		{
			ID:                "gpt-3.5-turbo-0125",
			Alias:             "gpt-3.5-turbo",
			MaxTokens:         16385,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.5,
			OutputPricePer1K:  1.5,
			ContextWindow:     16385,
		},
	}

	return &OpenAIProvider{
		BaseProvider: NewBaseProvider("openai", models, &OpenAINormalizer{}),
	}
}

// Chat sends a chat request to OpenAI
func (p *OpenAIProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

// StreamChat sends a streaming chat request to OpenAI
func (p *OpenAIProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}
