package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/swarm-ai/swarm/pkg/api"
)

func TestAnthropicNormalizer_NormalizeRequest(t *testing.T) {
	normalizer := &AnthropicNormalizer{}

	req := &api.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []api.Message{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
		MaxTokens:    100,
		SystemPrompt: "You are a helpful assistant.",
	}

	result, err := normalizer.NormalizeRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAnthropicNormalizer_NormalizeRequest_WithTools(t *testing.T) {
	normalizer := &AnthropicNormalizer{}

	req := &api.ChatRequest{
		Model: "claude-sonnet-4",
		Messages: []api.Message{
			{
				Role:    "user",
				Content: "What's the weather?",
			},
		},
		MaxTokens: 100,
		Tools: []api.Tool{
			{
				Type: "function",
				Function: api.FunctionDef{
					Name:        "get_weather",
					Description: "Get the current weather",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]string{
								"type": "string",
							},
						},
					},
				},
			},
		},
		ToolChoice: &api.ToolChoice{
			Type: "auto",
		},
	}

	result, err := normalizer.NormalizeRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAnthropicNormalizer_NormalizeRequest_WithTemperature(t *testing.T) {
	normalizer := &AnthropicNormalizer{}

	temp := 0.7
	req := &api.ChatRequest{
		Model:       "claude-sonnet-4",
		Messages:    []api.Message{{Role: "user", Content: "Hello"}},
		MaxTokens:   100,
		Temperature: &temp,
	}

	result, err := normalizer.NormalizeRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestAnthropicNormalizer_NormalizeResponse(t *testing.T) {
	normalizer := &AnthropicNormalizer{}

	resp, err := normalizer.NormalizeResponse(nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "response is nil")
}

func TestAnthropicNormalizer_NormalizeStreamEvent(t *testing.T) {
	normalizer := &AnthropicNormalizer{}

	event, err := normalizer.NormalizeStreamEvent(nil)
	assert.Error(t, err)
	assert.Nil(t, event)
	assert.Contains(t, err.Error(), "event is nil")
}

func TestAnthropicNormalizer_NormalizeError(t *testing.T) {
	normalizer := &AnthropicNormalizer{}

	pe := normalizer.NormalizeError(assert.AnError)
	assert.NotNil(t, pe)
	assert.IsType(t, &ProviderError{}, pe)
}

func TestOpenAINormalizer_NormalizeRequest(t *testing.T) {
	normalizer := &OpenAINormalizer{}

	req := &api.ChatRequest{
		Model: "gpt-4",
		Messages: []api.Message{
			{
				Role:    "user",
				Content: "Hello, world!",
			},
		},
		MaxTokens: 100,
	}

	result, err := normalizer.NormalizeRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenAINormalizer_NormalizeRequest_WithTools(t *testing.T) {
	normalizer := &OpenAINormalizer{}

	req := &api.ChatRequest{
		Model: "gpt-4",
		Messages: []api.Message{
			{
				Role:    "user",
				Content: "What's the weather?",
			},
		},
		MaxTokens: 100,
		Tools: []api.Tool{
			{
				Type: "function",
				Function: api.FunctionDef{
					Name:        "get_weather",
					Description: "Get the current weather",
					Parameters: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"location": map[string]string{
								"type": "string",
							},
						},
					},
				},
			},
		},
		ToolChoice: &api.ToolChoice{
			Type: "auto",
		},
	}

	result, err := normalizer.NormalizeRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenAINormalizer_NormalizeRequest_SystemMessage(t *testing.T) {
	normalizer := &OpenAINormalizer{}

	req := &api.ChatRequest{
		Model: "gpt-4",
		Messages: []api.Message{
			{
				Role:    "system",
				Content: "You are a helpful assistant.",
			},
			{
				Role:    "user",
				Content: "Hello",
			},
		},
		MaxTokens: 100,
	}

	result, err := normalizer.NormalizeRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenAINormalizer_NormalizeRequest_ResponseFormat(t *testing.T) {
	normalizer := &OpenAINormalizer{}

	req := &api.ChatRequest{
		Model: "gpt-4",
		Messages: []api.Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
		ResponseFormat: &api.ResponseFormat{
			Type: "json_object",
		},
	}

	result, err := normalizer.NormalizeRequest(req)
	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestOpenAINormalizer_NormalizeResponse(t *testing.T) {
	normalizer := &OpenAINormalizer{}

	resp, err := normalizer.NormalizeResponse(nil)
	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Contains(t, err.Error(), "response is nil")
}

func TestOpenAINormalizer_NormalizeStreamEvent(t *testing.T) {
	normalizer := &OpenAINormalizer{}

	event, err := normalizer.NormalizeStreamEvent(nil)
	assert.Error(t, err)
	assert.Nil(t, event)
	assert.Contains(t, err.Error(), "event is nil")
}

func TestOpenAINormalizer_NormalizeError(t *testing.T) {
	normalizer := &OpenAINormalizer{}

	pe := normalizer.NormalizeError(assert.AnError)
	assert.NotNil(t, pe)
	assert.IsType(t, &ProviderError{}, pe)
}

func TestNewProviderError(t *testing.T) {
	pe := NewProviderError("test_code", "test message", ErrorTypeServer, assert.AnError)

	assert.NotNil(t, pe)
	assert.IsType(t, &ProviderError{}, pe)

	assert.Equal(t, "test_code", pe.Code)
	assert.Equal(t, "test message", pe.Message)
	assert.Equal(t, ErrorTypeServer, pe.Type)
	assert.True(t, pe.Retryable)
}

func TestProviderError_Error(t *testing.T) {
	err := NewProviderError("code", "message", ErrorTypeAuth, assert.AnError)

	errStr := err.Error()
	assert.Contains(t, errStr, "code")
	assert.Contains(t, errStr, "message")
}

func TestProviderError_Unwrap(t *testing.T) {
	originalErr := assert.AnError
	err := NewProviderError("code", "message", ErrorTypeServer, originalErr)

	assert.Equal(t, originalErr, err.Unwrap())
}
