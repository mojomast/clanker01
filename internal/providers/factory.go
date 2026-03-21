package providers

import (
	"fmt"
	"time"

	"github.com/swarm-ai/swarm/internal/config"
	"github.com/swarm-ai/swarm/pkg/api"
)

// NewProviderFromConfig creates a fully-wired LLM provider from a config name and
// provider configuration. It creates the appropriate base provider, wraps it with
// retry logic, and optionally wraps with caching.
func NewProviderFromConfig(name string, cfg config.ProviderConfig) (api.LLMProvider, error) {
	// Create the base provider based on the name
	var provider api.LLMProvider
	switch name {
	case "anthropic":
		provider = NewAnthropicProvider()
	case "openai":
		provider = NewOpenAIProvider()
	case "google":
		provider = NewGoogleProvider()
	case "azure":
		provider = NewAzureProvider()
	case "aws", "bedrock":
		provider = NewBedrockProvider()
	case "ollama":
		provider = NewOllamaProvider()
	case "openrouter":
		provider = NewOpenRouterProvider()
	case "together":
		provider = NewTogetherProvider()
	case "groq":
		provider = NewGroqProvider()
	case "mistral":
		provider = NewMistralProvider()
	case "deepseek":
		provider = NewDeepSeekProvider()
	case "cohere":
		provider = NewCohereProvider()
	case "perplexity":
		provider = NewPerplexityProvider()
	case "replicate":
		provider = NewReplicateProvider()
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}

	// Build the api.ProviderConfig from the config.ProviderConfig
	apiCfg := &api.ProviderConfig{
		APIKey:  cfg.APIKey,
		BaseURL: cfg.BaseURL,
	}

	// Apply timeout from options if present
	if cfg.Options != nil {
		if timeoutStr, ok := cfg.Options["timeout"]; ok {
			if ts, ok := timeoutStr.(string); ok {
				if d, err := time.ParseDuration(ts); err == nil {
					apiCfg.Timeout = d
				}
			}
		}
	}

	// Configure the provider
	if err := provider.Configure(apiCfg); err != nil {
		return nil, fmt.Errorf("configure provider %s: %w", name, err)
	}

	// Wrap with retry logic using defaults
	retryConfig := DefaultRetryConfig

	// Allow overriding max retries from options
	if cfg.Options != nil {
		if maxRetries, ok := cfg.Options["max_retries"]; ok {
			if mr, ok := maxRetries.(float64); ok {
				retryConfig.MaxAttempts = int(mr)
			}
		}
	}

	provider = WithRetry(provider, retryConfig)

	// Optionally wrap with caching if enabled in options
	if cfg.Options != nil {
		if cacheEnabled, ok := cfg.Options["cache"]; ok {
			if enabled, ok := cacheEnabled.(bool); ok && enabled {
				cacheTTL := 5 * time.Minute
				if ttlStr, ok := cfg.Options["cache_ttl"]; ok {
					if ts, ok := ttlStr.(string); ok {
						if d, err := time.ParseDuration(ts); err == nil {
							cacheTTL = d
						}
					}
				}
				cache := NewInMemoryCache(cacheTTL)
				provider = WithCache(provider, cache)
			}
		}
	}

	return provider, nil
}

// InitializeProviders creates and configures all providers from the LLM configuration.
// It returns a ProviderRegistry with all configured providers registered and the
// default provider set according to the config.
func InitializeProviders(llmCfg config.LLMConfig) (*ProviderRegistry, error) {
	registry := &ProviderRegistry{
		providers: make(map[string]api.LLMProvider),
		configs:   make(map[string]*api.ProviderConfig),
	}

	for name, provCfg := range llmCfg.Providers {
		provider, err := NewProviderFromConfig(name, provCfg)
		if err != nil {
			return nil, fmt.Errorf("initialize provider %s: %w", name, err)
		}
		registry.Register(name, provider)
	}

	// Set the default provider
	if llmCfg.DefaultProvider != "" {
		if err := registry.SetDefault(llmCfg.DefaultProvider); err != nil {
			return nil, fmt.Errorf("set default provider: %w", err)
		}
	}

	return registry, nil
}
