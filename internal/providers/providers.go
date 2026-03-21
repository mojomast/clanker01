package providers

import (
	"context"

	"github.com/swarm-ai/swarm/pkg/api"
)

// GoogleProvider implements Google AI API
type GoogleProvider struct {
	*BaseProvider
}

// NewGoogleProvider creates a new Google provider
func NewGoogleProvider() *GoogleProvider {
	models := []api.ModelInfo{
		{
			ID:                "gemini-2.0-flash-exp",
			Alias:             "gemini-2.0",
			MaxTokens:         1000000,
			MaxOutputTokens:   8192,
			SupportsVision:    true,
			SupportsAudio:     true,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.075,
			OutputPricePer1K:  0.3,
			ContextWindow:     1000000,
		},
		{
			ID:                "gemini-1.5-pro",
			Alias:             "gemini-pro",
			MaxTokens:         2800000,
			MaxOutputTokens:   8192,
			SupportsVision:    true,
			SupportsAudio:     true,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   1.25,
			OutputPricePer1K:  5.0,
			ContextWindow:     2800000,
		},
	}

	return &GoogleProvider{
		BaseProvider: NewBaseProvider("google", models, nil),
	}
}

// AzureProvider implements Azure OpenAI API
type AzureProvider struct {
	*BaseProvider
}

// NewAzureProvider creates a new Azure provider
func NewAzureProvider() *AzureProvider {
	models := []api.ModelInfo{
		{
			ID:                "gpt-4o",
			Alias:             "azure-gpt-4o",
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
	}

	return &AzureProvider{
		BaseProvider: NewBaseProvider("azure", models, &OpenAINormalizer{}),
	}
}

// BedrockProvider implements AWS Bedrock API
type BedrockProvider struct {
	*BaseProvider
}

// NewBedrockProvider creates a new Bedrock provider
func NewBedrockProvider() *BedrockProvider {
	models := []api.ModelInfo{
		{
			ID:                "anthropic.claude-3-5-sonnet-20241022-v2:0",
			Alias:             "bedrock-claude-3-5-sonnet",
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
	}

	return &BedrockProvider{
		BaseProvider: NewBaseProvider("aws", models, &AnthropicNormalizer{}),
	}
}

// OllamaProvider implements Ollama API
type OllamaProvider struct {
	*BaseProvider
}

// NewOllamaProvider creates a new Ollama provider
func NewOllamaProvider() *OllamaProvider {
	models := []api.ModelInfo{
		{
			ID:                "llama3.1",
			Alias:             "llama3.1",
			MaxTokens:         128000,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.0,
			OutputPricePer1K:  0.0,
			ContextWindow:     128000,
		},
		{
			ID:                "mistral-nemo",
			Alias:             "mistral-nemo",
			MaxTokens:         128000,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.0,
			OutputPricePer1K:  0.0,
			ContextWindow:     128000,
		},
	}

	return &OllamaProvider{
		BaseProvider: NewBaseProvider("ollama", models, nil),
	}
}

// OpenRouterProvider implements OpenRouter API
type OpenRouterProvider struct {
	*BaseProvider
}

// NewOpenRouterProvider creates a new OpenRouter provider
func NewOpenRouterProvider() *OpenRouterProvider {
	models := []api.ModelInfo{
		{
			ID:                "anthropic/claude-3.5-sonnet",
			Alias:             "openrouter-claude-3.5-sonnet",
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
	}

	return &OpenRouterProvider{
		BaseProvider: NewBaseProvider("openrouter", models, &OpenAINormalizer{}),
	}
}

// TogetherProvider implements Together AI API
type TogetherProvider struct {
	*BaseProvider
}

// NewTogetherProvider creates a new Together provider
func NewTogetherProvider() *TogetherProvider {
	models := []api.ModelInfo{
		{
			ID:                "meta-llama/Llama-3.1-405B-Instruct-Turbo",
			Alias:             "llama-3.1-405b",
			MaxTokens:         131072,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   2.7,
			OutputPricePer1K:  2.7,
			ContextWindow:     131072,
		},
	}

	return &TogetherProvider{
		BaseProvider: NewBaseProvider("together", models, &OpenAINormalizer{}),
	}
}

// GroqProvider implements Groq API
type GroqProvider struct {
	*BaseProvider
}

// NewGroqProvider creates a new Groq provider
func NewGroqProvider() *GroqProvider {
	models := []api.ModelInfo{
		{
			ID:                "llama-3.1-70b-versatile",
			Alias:             "groq-llama-3.1-70b",
			MaxTokens:         131072,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.59,
			OutputPricePer1K:  0.59,
			ContextWindow:     131072,
		},
	}

	return &GroqProvider{
		BaseProvider: NewBaseProvider("groq", models, &OpenAINormalizer{}),
	}
}

// MistralProvider implements Mistral API
type MistralProvider struct {
	*BaseProvider
}

// NewMistralProvider creates a new Mistral provider
func NewMistralProvider() *MistralProvider {
	models := []api.ModelInfo{
		{
			ID:                "mistral-large-latest",
			Alias:             "mistral-large",
			MaxTokens:         128000,
			MaxOutputTokens:   8192,
			SupportsVision:    true,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   2.0,
			OutputPricePer1K:  6.0,
			ContextWindow:     128000,
		},
	}

	return &MistralProvider{
		BaseProvider: NewBaseProvider("mistral", models, nil),
	}
}

// DeepSeekProvider implements DeepSeek API
type DeepSeekProvider struct {
	*BaseProvider
}

// NewDeepSeekProvider creates a new DeepSeek provider
func NewDeepSeekProvider() *DeepSeekProvider {
	models := []api.ModelInfo{
		{
			ID:                "deepseek-chat",
			Alias:             "deepseek-chat",
			MaxTokens:         128000,
			MaxOutputTokens:   8192,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.27,
			OutputPricePer1K:  1.1,
			ContextWindow:     128000,
		},
	}

	return &DeepSeekProvider{
		BaseProvider: NewBaseProvider("deepseek", models, &OpenAINormalizer{}),
	}
}

// CohereProvider implements Cohere API
type CohereProvider struct {
	*BaseProvider
}

// NewCohereProvider creates a new Cohere provider
func NewCohereProvider() *CohereProvider {
	models := []api.ModelInfo{
		{
			ID:                "command-r-plus-08-2024",
			Alias:             "command-r-plus",
			MaxTokens:         128000,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   3.0,
			OutputPricePer1K:  15.0,
			ContextWindow:     128000,
		},
	}

	return &CohereProvider{
		BaseProvider: NewBaseProvider("cohere", models, nil),
	}
}

// PerplexityProvider implements Perplexity API
type PerplexityProvider struct {
	*BaseProvider
}

// NewPerplexityProvider creates a new Perplexity provider
func NewPerplexityProvider() *PerplexityProvider {
	models := []api.ModelInfo{
		{
			ID:                "llama-3.1-sonar-small-128k-online",
			Alias:             "sonar-small",
			MaxTokens:         127072,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     true,
			SupportsJSON:      true,
			InputPricePer1K:   0.0,
			OutputPricePer1K:  0.0,
			ContextWindow:     127072,
		},
	}

	return &PerplexityProvider{
		BaseProvider: NewBaseProvider("perplexity", models, &OpenAINormalizer{}),
	}
}

// ReplicateProvider implements Replicate API
type ReplicateProvider struct {
	*BaseProvider
}

// NewReplicateProvider creates a new Replicate provider
func NewReplicateProvider() *ReplicateProvider {
	models := []api.ModelInfo{
		{
			ID:                "meta/meta-llama-3.1-405b-instruct",
			Alias:             "replicate-llama-3.1-405b",
			MaxTokens:         131072,
			MaxOutputTokens:   4096,
			SupportsVision:    false,
			SupportsAudio:     false,
			SupportsStreaming: true,
			SupportsTools:     false,
			SupportsJSON:      true,
			InputPricePer1K:   0.55,
			OutputPricePer1K:  0.55,
			ContextWindow:     131072,
		},
	}

	return &ReplicateProvider{
		BaseProvider: NewBaseProvider("replicate", models, &OpenAINormalizer{}),
	}
}

func (p *GoogleProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *GoogleProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *AzureProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *AzureProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *BedrockProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *BedrockProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *OllamaProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *OllamaProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *OpenRouterProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *OpenRouterProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *TogetherProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *TogetherProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *GroqProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *GroqProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *MistralProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *MistralProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *DeepSeekProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *DeepSeekProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *CohereProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *CohereProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *PerplexityProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *PerplexityProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}

func (p *ReplicateProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	return p.BaseProvider.Chat(ctx, req)
}

func (p *ReplicateProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	return p.BaseProvider.StreamChat(ctx, req)
}
