package providers

import (
	"context"
	"fmt"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

type RetryConfig struct {
	MaxAttempts     int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []ErrorType
}

var DefaultRetryConfig = RetryConfig{
	MaxAttempts:   3,
	InitialDelay:  1 * time.Second,
	MaxDelay:      30 * time.Second,
	BackoffFactor: 2.0,
	RetryableErrors: []ErrorType{
		ErrorTypeRateLimit,
		ErrorTypeServer,
	},
}

type RetryableProvider struct {
	provider api.LLMProvider
	config   RetryConfig
}

func (p *RetryableProvider) Name() string {
	return p.provider.Name()
}

func (p *RetryableProvider) Models() []api.ModelInfo {
	return p.provider.Models()
}

func (p *RetryableProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	var lastErr error
	delay := p.config.InitialDelay
	maxAttempts := p.config.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := p.provider.Chat(ctx, req)

		if err == nil {
			// Metrics are already recorded by the underlying provider
			return resp, nil
		}

		pErr := p.normalizeError(err)
		if !p.isRetryable(pErr) {
			return nil, err
		}

		lastErr = err

		if pErr.RetryAfter > 0 {
			delay = pErr.RetryAfter
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			delay = time.Duration(float64(delay) * p.config.BackoffFactor)
			if delay > p.config.MaxDelay {
				delay = p.config.MaxDelay
			}
		}
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (p *RetryableProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	var lastErr error
	delay := p.config.InitialDelay
	maxAttempts := p.config.MaxAttempts
	if maxAttempts < 1 {
		maxAttempts = 1
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		ch, err := p.provider.StreamChat(ctx, req)
		if err == nil {
			return ch, nil
		}

		pErr := p.normalizeError(err)
		if !p.isRetryable(pErr) {
			return nil, err
		}

		lastErr = err

		if pErr.RetryAfter > 0 {
			delay = pErr.RetryAfter
		}

		if attempt < maxAttempts {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}

			delay = time.Duration(float64(delay) * p.config.BackoffFactor)
			if delay > p.config.MaxDelay {
				delay = p.config.MaxDelay
			}
		}
	}

	return nil, fmt.Errorf("max retries exceeded for stream: %w", lastErr)
}

func (p *RetryableProvider) SupportsStreaming() bool {
	return p.provider.SupportsStreaming()
}

func (p *RetryableProvider) SupportsFunctionCalling() bool {
	return p.provider.SupportsFunctionCalling()
}

func (p *RetryableProvider) SupportsVision() bool {
	return p.provider.SupportsVision()
}

func (p *RetryableProvider) SupportsAudio() bool {
	return p.provider.SupportsAudio()
}

func (p *RetryableProvider) MaxTokens(model string) int {
	return p.provider.MaxTokens(model)
}

func (p *RetryableProvider) Configure(config *api.ProviderConfig) error {
	return p.provider.Configure(config)
}

func (p *RetryableProvider) Metrics() *api.ProviderMetrics {
	return p.provider.Metrics()
}

func (p *RetryableProvider) normalizeError(err error) *ProviderError {
	if pe, ok := err.(*ProviderError); ok {
		return pe
	}
	return &ProviderError{
		Code:      "unknown",
		Message:   err.Error(),
		Retryable: false,
		Type:      ErrorTypeServer,
	}
}

func (p *RetryableProvider) isRetryable(err *ProviderError) bool {
	if err.Retryable {
		return true
	}
	for _, t := range p.config.RetryableErrors {
		if err.Type == t {
			return true
		}
	}
	return false
}
