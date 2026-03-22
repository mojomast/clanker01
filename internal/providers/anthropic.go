package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

const (
	anthropicDefaultBaseURL = "https://api.anthropic.com"
	anthropicMessagesPath   = "/v1/messages"
	anthropicAPIVersion     = "2023-06-01"
)

// AnthropicProvider implements Anthropic API with real HTTP calls.
type AnthropicProvider struct {
	*BaseProvider
	httpClient       *http.Client
	streamHTTPClient *http.Client // separate client with no timeout for streaming
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
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		streamHTTPClient: &http.Client{
			// No timeout for streaming — duration is controlled by context
		},
	}
}

// Configure configures the Anthropic provider and updates HTTP client timeout.
func (p *AnthropicProvider) Configure(config *api.ProviderConfig) error {
	if err := p.BaseProvider.Configure(config); err != nil {
		return err
	}

	if config.Timeout > 0 {
		p.mu.Lock()
		// Recreate the HTTP client to avoid racing with concurrent Do calls
		p.httpClient = &http.Client{
			Timeout: config.Timeout,
		}
		p.mu.Unlock()
	}

	return nil
}

// getHTTPClient returns the standard HTTP client under the lock.
func (p *AnthropicProvider) getHTTPClient() *http.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.httpClient
}

// getStreamHTTPClient returns the streaming HTTP client under the lock.
func (p *AnthropicProvider) getStreamHTTPClient() *http.Client {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.streamHTTPClient
}

// baseURL returns the configured base URL or the default Anthropic URL.
func (p *AnthropicProvider) baseURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config != nil && p.config.BaseURL != "" {
		return strings.TrimRight(p.config.BaseURL, "/")
	}
	return anthropicDefaultBaseURL
}

// apiKey returns the configured API key.
func (p *AnthropicProvider) apiKey() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config != nil {
		return p.config.APIKey
	}
	return ""
}

// buildHTTPRequest creates an http.Request with Anthropic headers.
func (p *AnthropicProvider) buildHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	url := p.baseURL() + anthropicMessagesPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("anthropic: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey())
	req.Header.Set("anthropic-version", anthropicAPIVersion)

	// Add any extra headers from config
	p.mu.RLock()
	if p.config != nil {
		for k, v := range p.config.Headers {
			req.Header.Set(k, v)
		}
	}
	p.mu.RUnlock()

	return req, nil
}

// handleErrorResponse parses an Anthropic error response and returns a ProviderError.
func (p *AnthropicProvider) handleErrorResponse(resp *http.Response) *ProviderError {
	// Limit error body to 1 MB to prevent OOM from malicious/buggy servers
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return NewProviderError(
			"read_error",
			fmt.Sprintf("failed to read error response: %v", err),
			ErrorTypeServer,
			err,
		)
	}

	var errResp anthropicErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// Could not parse structured error — return raw body
		return NewProviderError(
			fmt.Sprintf("http_%d", resp.StatusCode),
			string(body),
			ErrorTypeServer,
			fmt.Errorf("anthropic: HTTP %d: %s", resp.StatusCode, string(body)),
		)
	}

	return errResp.Error.toProviderError(resp.StatusCode)
}

// Chat sends a chat request to the Anthropic API and returns a normalized response.
func (p *AnthropicProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	start := time.Now()

	key := p.apiKey()
	if key == "" {
		return nil, NewProviderError("missing_api_key", "Anthropic API key not configured", ErrorTypeAuth, fmt.Errorf("anthropic: API key not configured"))
	}

	// Normalize the request to Anthropic format
	normalized, err := p.normalizer.NormalizeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: normalize request: %w", err)
	}

	ar, ok := normalized.(*anthropicRequest)
	if !ok {
		return nil, fmt.Errorf("anthropic: normalize returned unexpected type %T", normalized)
	}
	ar.Stream = false

	// Set default max_tokens if not specified
	if ar.MaxTokens == 0 {
		ar.MaxTokens = p.MaxTokens(req.Model)
	}

	body, err := json.Marshal(ar)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := p.buildHTTPRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	resp, err := p.getHTTPClient().Do(httpReq)
	if err != nil {
		latency := float64(time.Since(start).Milliseconds())
		p.recordMetrics(nil, latency, err)
		return nil, fmt.Errorf("anthropic: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		latency := float64(time.Since(start).Milliseconds())
		provErr := p.handleErrorResponse(resp)
		p.recordMetrics(nil, latency, provErr)
		return nil, provErr
	}

	// Parse the response
	var apiResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		latency := float64(time.Since(start).Milliseconds())
		p.recordMetrics(nil, latency, err)
		return nil, fmt.Errorf("anthropic: decode response: %w", err)
	}

	// Normalize to ChatResponse
	chatResp, err := p.normalizer.NormalizeResponse(&apiResp)
	if err != nil {
		return nil, fmt.Errorf("anthropic: normalize response: %w", err)
	}

	// Record metrics and cost
	latency := float64(time.Since(start).Milliseconds())
	p.recordMetrics(&chatResp.Usage, latency, nil)
	p.trackCost(req.Model, &chatResp.Usage)

	return chatResp, nil
}

// StreamChat sends a streaming chat request to Anthropic and returns a channel of events.
func (p *AnthropicProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	key := p.apiKey()
	if key == "" {
		return nil, NewProviderError("missing_api_key", "Anthropic API key not configured", ErrorTypeAuth, fmt.Errorf("anthropic: API key not configured"))
	}

	// Normalize the request to Anthropic format
	normalized, err := p.normalizer.NormalizeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic: normalize request: %w", err)
	}

	ar, ok := normalized.(*anthropicRequest)
	if !ok {
		return nil, fmt.Errorf("anthropic: normalize returned unexpected type %T", normalized)
	}
	ar.Stream = true

	// Set default max_tokens if not specified
	if ar.MaxTokens == 0 {
		ar.MaxTokens = p.MaxTokens(req.Model)
	}

	body, err := json.Marshal(ar)
	if err != nil {
		return nil, fmt.Errorf("anthropic: marshal request: %w", err)
	}

	httpReq, err := p.buildHTTPRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	// Use the stream HTTP client (no timeout)
	resp, err := p.getStreamHTTPClient().Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic: HTTP stream request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		provErr := p.handleErrorResponse(resp)
		resp.Body.Close()
		return nil, provErr
	}

	ch := make(chan api.ChatStreamEvent, 64)

	go p.consumeSSEStream(ctx, resp.Body, ch, req.Model)

	return ch, nil
}

// consumeSSEStream reads Anthropic SSE events from a reader, normalizes them,
// and sends them to the channel. Closes the channel and reader when done.
func (p *AnthropicProvider) consumeSSEStream(ctx context.Context, body io.ReadCloser, ch chan<- api.ChatStreamEvent, model string) {
	defer close(ch)
	defer body.Close()

	// sendEvent sends an event to the channel, returning false if the context
	// is cancelled (meaning the consumer is gone and we should stop).
	sendEvent := func(evt api.ChatStreamEvent) bool {
		select {
		case ch <- evt:
			return true
		case <-ctx.Done():
			return false
		}
	}

	start := time.Now()
	scanner := bufio.NewScanner(body)
	// Allow up to 1 MB per SSE line (default 64KB is too small for large tool call payloads)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var eventType string
	var totalUsage api.Usage

	for scanner.Scan() {
		line := scanner.Text()

		// Check context cancellation
		select {
		case <-ctx.Done():
			sendEvent(api.ChatStreamEvent{
				Type:  api.StreamEventError,
				Error: ctx.Err(),
			})
			return
		default:
		}

		// SSE format: "event: <type>" followed by "data: <json>"
		if strings.HasPrefix(line, "event: ") {
			eventType = strings.TrimPrefix(line, "event: ")
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")

			var streamEvent anthropicStreamEvent
			if err := json.Unmarshal([]byte(data), &streamEvent); err != nil {
				if !sendEvent(api.ChatStreamEvent{
					Type:  api.StreamEventError,
					Error: fmt.Errorf("anthropic: parse stream event: %w", err),
				}) {
					return
				}
				continue
			}

			// Use the event type from the SSE "event:" line if available
			if eventType != "" {
				streamEvent.Type = eventType
			}

			normalized, err := p.normalizer.NormalizeStreamEvent(&streamEvent)
			if err != nil {
				if !sendEvent(api.ChatStreamEvent{
					Type:  api.StreamEventError,
					Error: err,
				}) {
					return
				}
				continue
			}

			// Accumulate usage for metrics
			if normalized.Response != nil {
				if normalized.Response.Usage.PromptTokens > 0 {
					totalUsage.PromptTokens = normalized.Response.Usage.PromptTokens
				}
				if normalized.Response.Usage.CompletionTokens > 0 {
					totalUsage.CompletionTokens = normalized.Response.Usage.CompletionTokens
				}
				totalUsage.TotalTokens = totalUsage.PromptTokens + totalUsage.CompletionTokens
			}

			if !sendEvent(*normalized) {
				return
			}

			eventType = "" // Reset for next event
		}

		// Empty lines separate events — skip them
	}

	if err := scanner.Err(); err != nil {
		sendEvent(api.ChatStreamEvent{
			Type:  api.StreamEventError,
			Error: fmt.Errorf("anthropic: stream read error: %w", err),
		})
	}

	// Record metrics after stream completes
	latency := float64(time.Since(start).Milliseconds())
	if totalUsage.TotalTokens > 0 {
		p.recordMetrics(&totalUsage, latency, nil)
		p.trackCost(model, &totalUsage)
	} else {
		p.recordMetrics(nil, latency, nil)
	}
}
