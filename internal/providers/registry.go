package providers

import (
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/pkg/api"
)

// ProviderRegistry manages all providers
type ProviderRegistry struct {
	mu          sync.RWMutex
	providers   map[string]api.LLMProvider
	configs     map[string]*api.ProviderConfig
	defaultName string
}

var (
	globalRegistry *ProviderRegistry
	once           sync.Once
)

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	r := &ProviderRegistry{
		providers: make(map[string]api.LLMProvider),
		configs:   make(map[string]*api.ProviderConfig),
	}

	r.Register("anthropic", NewAnthropicProvider())
	r.Register("openai", NewOpenAIProvider())
	r.Register("google", NewGoogleProvider())
	r.Register("azure", NewAzureProvider())
	r.Register("aws", NewBedrockProvider())
	r.Register("ollama", NewOllamaProvider())
	r.Register("openrouter", NewOpenRouterProvider())
	r.Register("together", NewTogetherProvider())
	r.Register("groq", NewGroqProvider())
	r.Register("mistral", NewMistralProvider())
	r.Register("deepseek", NewDeepSeekProvider())
	r.Register("cohere", NewCohereProvider())
	r.Register("perplexity", NewPerplexityProvider())
	r.Register("replicate", NewReplicateProvider())

	return r
}

// GlobalRegistry returns the global registry instance
func GlobalRegistry() *ProviderRegistry {
	once.Do(func() {
		globalRegistry = NewProviderRegistry()
	})
	return globalRegistry
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(name string, provider api.LLMProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[name] = provider
}

// Get retrieves a provider by name
func (r *ProviderRegistry) Get(name string) (api.LLMProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if name == "" {
		name = r.defaultName
	}

	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", name)
	}

	return p, nil
}

// Configure configures a provider
func (r *ProviderRegistry) Configure(name string, config *api.ProviderConfig) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	p, ok := r.providers[name]
	if !ok {
		return fmt.Errorf("provider not found: %s", name)
	}

	if err := p.Configure(config); err != nil {
		return fmt.Errorf("configure provider: %w", err)
	}

	r.configs[name] = config

	if r.defaultName == "" {
		r.defaultName = name
	}

	return nil
}

// SetDefault sets the default provider
func (r *ProviderRegistry) SetDefault(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.providers[name]; !ok {
		return fmt.Errorf("provider not found: %s", name)
	}

	r.defaultName = name
	return nil
}

// List returns all registered provider names
func (r *ProviderRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}
