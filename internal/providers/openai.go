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
	openaiDefaultBaseURL = "https://api.openai.com"
	openaiChatPath       = "/v1/chat/completions"
)

// OpenAIProvider implements OpenAI API with real HTTP calls.
type OpenAIProvider struct {
	*BaseProvider
	httpClient       *http.Client
	streamHTTPClient *http.Client // separate client with no timeout for streaming
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
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		streamHTTPClient: &http.Client{
			// No timeout for streaming — duration is controlled by context
		},
	}
}

// Configure configures the OpenAI provider and updates HTTP client timeout.
func (p *OpenAIProvider) Configure(config *api.ProviderConfig) error {
	if err := p.BaseProvider.Configure(config); err != nil {
		return err
	}

	if config.Timeout > 0 {
		p.mu.Lock()
		p.httpClient.Timeout = config.Timeout
		p.mu.Unlock()
	}

	return nil
}

// baseURL returns the configured base URL or the default OpenAI URL.
func (p *OpenAIProvider) baseURL() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config != nil && p.config.BaseURL != "" {
		return strings.TrimRight(p.config.BaseURL, "/")
	}
	return openaiDefaultBaseURL
}

// apiKey returns the configured API key.
func (p *OpenAIProvider) apiKey() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if p.config != nil {
		return p.config.APIKey
	}
	return ""
}

// buildHTTPRequest creates an http.Request with OpenAI headers.
func (p *OpenAIProvider) buildHTTPRequest(ctx context.Context, body []byte) (*http.Request, error) {
	url := p.baseURL() + openaiChatPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("openai: create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey())

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

// handleErrorResponse parses an OpenAI error response and returns a ProviderError.
func (p *OpenAIProvider) handleErrorResponse(resp *http.Response) *ProviderError {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return NewProviderError(
			"read_error",
			fmt.Sprintf("failed to read error response: %v", err),
			ErrorTypeServer,
			err,
		)
	}

	var errResp openaiErrorResponse
	if err := json.Unmarshal(body, &errResp); err != nil {
		// Could not parse structured error — return raw body
		return NewProviderError(
			fmt.Sprintf("http_%d", resp.StatusCode),
			string(body),
			ErrorTypeServer,
			fmt.Errorf("openai: HTTP %d: %s", resp.StatusCode, string(body)),
		)
	}

	return errResp.Error.toProviderError(resp.StatusCode)
}

// Chat sends a chat request to the OpenAI API and returns a normalized response.
func (p *OpenAIProvider) Chat(ctx context.Context, req *api.ChatRequest) (*api.ChatResponse, error) {
	start := time.Now()

	key := p.apiKey()
	if key == "" {
		return nil, NewProviderError("missing_api_key", "OpenAI API key not configured", ErrorTypeAuth, fmt.Errorf("openai: API key not configured"))
	}

	// Normalize the request to OpenAI format
	normalized, err := p.normalizer.NormalizeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("openai: normalize request: %w", err)
	}

	or := normalized.(*openaiRequest)
	or.Stream = false

	body, err := json.Marshal(or)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	httpReq, err := p.buildHTTPRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		latency := float64(time.Since(start).Milliseconds())
		p.recordMetrics(nil, latency, err)
		return nil, fmt.Errorf("openai: HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		latency := float64(time.Since(start).Milliseconds())
		provErr := p.handleErrorResponse(resp)
		p.recordMetrics(nil, latency, provErr)
		return nil, provErr
	}

	// Parse the response
	var apiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		latency := float64(time.Since(start).Milliseconds())
		p.recordMetrics(nil, latency, err)
		return nil, fmt.Errorf("openai: decode response: %w", err)
	}

	// Normalize to ChatResponse
	chatResp, err := p.normalizer.NormalizeResponse(&apiResp)
	if err != nil {
		return nil, fmt.Errorf("openai: normalize response: %w", err)
	}

	// Record metrics and cost
	latency := float64(time.Since(start).Milliseconds())
	p.recordMetrics(&chatResp.Usage, latency, nil)
	p.trackCost(req.Model, &chatResp.Usage)

	return chatResp, nil
}

// StreamChat sends a streaming chat request to OpenAI and returns a channel of events.
func (p *OpenAIProvider) StreamChat(ctx context.Context, req *api.ChatRequest) (<-chan api.ChatStreamEvent, error) {
	key := p.apiKey()
	if key == "" {
		return nil, NewProviderError("missing_api_key", "OpenAI API key not configured", ErrorTypeAuth, fmt.Errorf("openai: API key not configured"))
	}

	// Normalize the request to OpenAI format
	normalized, err := p.normalizer.NormalizeRequest(req)
	if err != nil {
		return nil, fmt.Errorf("openai: normalize request: %w", err)
	}

	or := normalized.(*openaiRequest)
	or.Stream = true

	body, err := json.Marshal(or)
	if err != nil {
		return nil, fmt.Errorf("openai: marshal request: %w", err)
	}

	httpReq, err := p.buildHTTPRequest(ctx, body)
	if err != nil {
		return nil, err
	}

	// Use the stream HTTP client (no timeout)
	resp, err := p.streamHTTPClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("openai: HTTP stream request failed: %w", err)
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

// consumeSSEStream reads OpenAI SSE events from a reader, normalizes them,
// and sends them to the channel. Closes the channel and reader when done.
func (p *OpenAIProvider) consumeSSEStream(ctx context.Context, body io.ReadCloser, ch chan<- api.ChatStreamEvent, model string) {
	defer close(ch)
	defer body.Close()

	start := time.Now()
	scanner := bufio.NewScanner(body)

	var totalUsage api.Usage

	for scanner.Scan() {
		line := scanner.Text()

		// Check context cancellation
		select {
		case <-ctx.Done():
			ch <- api.ChatStreamEvent{
				Type:  api.StreamEventError,
				Error: ctx.Err(),
			}
			return
		default:
		}

		// OpenAI SSE format: "data: {json}" lines, terminated by "data: [DONE]"
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")

		// Check for stream termination
		if data == "[DONE]" {
			ch <- api.ChatStreamEvent{
				Type: api.StreamEventDone,
				Done: true,
			}
			break
		}

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			ch <- api.ChatStreamEvent{
				Type:  api.StreamEventError,
				Error: fmt.Errorf("openai: parse stream chunk: %w", err),
			}
			continue
		}

		normalized, err := p.normalizer.NormalizeStreamEvent(&chunk)
		if err != nil {
			ch <- api.ChatStreamEvent{
				Type:  api.StreamEventError,
				Error: err,
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

		ch <- *normalized
	}

	if err := scanner.Err(); err != nil {
		ch <- api.ChatStreamEvent{
			Type:  api.StreamEventError,
			Error: fmt.Errorf("openai: stream read error: %w", err),
		}
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
