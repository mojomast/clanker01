package providers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

// OpenAINormalizer converts between SWARM and OpenAI formats
type OpenAINormalizer struct{}

type openaiRequest struct {
	Model          string          `json:"model"`
	Messages       []openaiMessage `json:"messages"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	Temperature    float64         `json:"temperature,omitempty"`
	TopP           float64         `json:"top_p,omitempty"`
	Tools          []openaiTool    `json:"tools,omitempty"`
	ToolChoice     any             `json:"tool_choice,omitempty"`
	Stream         bool            `json:"stream,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type openaiMessage struct {
	Role       string           `json:"role"`
	Content    any              `json:"content"`
	ToolCalls  []openaiToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type openaiTool struct {
	Type     string     `json:"type"`
	Function openaiFunc `json:"function"`
}

type openaiFunc struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"`
}

type openaiToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openaiFuncCall `json:"function"`
}

type openaiFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// --- OpenAI API response structs ---

// openaiResponse maps to the OpenAI chat.completions response body.
type openaiResponse struct {
	ID                string         `json:"id"`
	Object            string         `json:"object"`
	Created           int64          `json:"created"`
	Model             string         `json:"model"`
	Choices           []openaiChoice `json:"choices"`
	Usage             openaiUsage    `json:"usage"`
	SystemFingerprint string         `json:"system_fingerprint,omitempty"`
}

// openaiChoice represents a single choice in the OpenAI response.
type openaiChoice struct {
	Index        int           `json:"index"`
	Message      openaiMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

// openaiUsage contains token usage data from the OpenAI API.
type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// openaiErrorResponse wraps an API error from OpenAI.
type openaiErrorResponse struct {
	Error openaiAPIError `json:"error"`
}

// openaiAPIError describes a typed error from the OpenAI API.
type openaiAPIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Param   string `json:"param,omitempty"`
	Code    string `json:"code"`
}

// toProviderError converts an OpenAI API error into a ProviderError.
func (e *openaiAPIError) toProviderError(statusCode int) *ProviderError {
	errType := ErrorTypeServer
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		errType = ErrorTypeAuth
	case statusCode == http.StatusTooManyRequests:
		errType = ErrorTypeRateLimit
	case statusCode == http.StatusBadRequest:
		errType = ErrorTypeInvalid
	case statusCode == http.StatusNotFound:
		errType = ErrorTypeNotFound
	case e.Code == "context_length_exceeded":
		errType = ErrorTypeContextLimit
	case e.Type == "invalid_request_error":
		errType = ErrorTypeInvalid
	case e.Type == "authentication_error":
		errType = ErrorTypeAuth
	case e.Type == "rate_limit_error":
		errType = ErrorTypeRateLimit
	}

	code := e.Code
	if code == "" {
		code = e.Type
	}

	pe := NewProviderError(code, e.Message, errType, fmt.Errorf("openai: %s", e.Message))
	if errType == ErrorTypeRateLimit {
		pe.RetryAfter = 5 * time.Second
	}
	return pe
}

// openaiStreamChunk represents a single streaming chunk from the OpenAI API.
type openaiStreamChunk struct {
	ID      string               `json:"id"`
	Object  string               `json:"object"`
	Created int64                `json:"created"`
	Model   string               `json:"model"`
	Choices []openaiStreamChoice `json:"choices"`
	Usage   *openaiUsage         `json:"usage,omitempty"`
}

// openaiStreamChoice represents a choice within a streaming chunk.
type openaiStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openaiStreamDelta `json:"delta"`
	FinishReason *string           `json:"finish_reason"`
}

// openaiStreamDelta represents the delta content in a streaming choice.
type openaiStreamDelta struct {
	Role      string           `json:"role,omitempty"`
	Content   *string          `json:"content,omitempty"`
	ToolCalls []openaiToolCall `json:"tool_calls,omitempty"`
}

func (n *OpenAINormalizer) NormalizeRequest(req *api.ChatRequest) (any, error) {
	or := &openaiRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		Stream:    req.Stream,
	}

	if req.Temperature != nil {
		or.Temperature = *req.Temperature
	}

	if req.TopP != nil {
		or.TopP = *req.TopP
	}

	if req.ResponseFormat != nil {
		or.ResponseFormat = &responseFormat{
			Type: req.ResponseFormat.Type,
		}
	}

	for _, msg := range req.Messages {
		om := openaiMessage{Role: msg.Role}

		if msg.Role == "system" {
			om.Content = toString(msg.Content)
		} else {
			om.Content = msg.Content
		}

		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				om.ToolCalls = append(om.ToolCalls, openaiToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: openaiFuncCall{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				})
			}
		}

		if msg.ToolCallID != "" {
			om.ToolCallID = msg.ToolCallID
		}

		if msg.Name != "" {
			om.Name = msg.Name
		}

		or.Messages = append(or.Messages, om)
	}

	for _, tool := range req.Tools {
		or.Tools = append(or.Tools, openaiTool{
			Type: "function",
			Function: openaiFunc{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		})
	}

	if req.ToolChoice != nil {
		switch req.ToolChoice.Type {
		case "auto", "none", "required":
			or.ToolChoice = req.ToolChoice.Type
		case "function":
			or.ToolChoice = map[string]any{
				"type": "function",
				"function": map[string]string{
					"name": req.ToolChoice.Function.Name,
				},
			}
		}
	}

	return or, nil
}

// NormalizeResponse converts an OpenAI API response (*openaiResponse) to a
// unified *api.ChatResponse. Returns an error if resp is nil or not the expected type.
func (n *OpenAINormalizer) NormalizeResponse(resp any) (*api.ChatResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("OpenAINormalizer.NormalizeResponse: not yet implemented")
	}

	or, ok := resp.(*openaiResponse)
	if !ok {
		return nil, fmt.Errorf("OpenAINormalizer.NormalizeResponse: unexpected type %T", resp)
	}

	chatResp := &api.ChatResponse{
		ID:      or.ID,
		Model:   or.Model,
		Created: or.Created,
		Usage: api.Usage{
			PromptTokens:     or.Usage.PromptTokens,
			CompletionTokens: or.Usage.CompletionTokens,
			TotalTokens:      or.Usage.TotalTokens,
		},
	}

	for _, choice := range or.Choices {
		msg := api.Message{
			Role: choice.Message.Role,
		}

		// Handle content — could be string or nil
		if choice.Message.Content != nil {
			switch v := choice.Message.Content.(type) {
			case string:
				msg.Content = v
			}
		}

		// Map tool calls
		for _, tc := range choice.Message.ToolCalls {
			msg.ToolCalls = append(msg.ToolCalls, api.ToolCall{
				ID:   tc.ID,
				Type: tc.Type,
				Function: api.FunctionCall{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}

		finishReason := mapOpenAIFinishReason(choice.FinishReason)

		chatResp.Choices = append(chatResp.Choices, api.Choice{
			Index:        choice.Index,
			Message:      msg,
			FinishReason: finishReason,
		})
	}

	// Set top-level finish reason from the first choice
	if len(chatResp.Choices) > 0 {
		chatResp.FinishReason = chatResp.Choices[0].FinishReason
	}

	return chatResp, nil
}

// NormalizeStreamEvent converts an OpenAI streaming chunk (*openaiStreamChunk) to a
// unified *api.ChatStreamEvent. Returns error if event is nil or wrong type.
func (n *OpenAINormalizer) NormalizeStreamEvent(event any) (*api.ChatStreamEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("OpenAINormalizer.NormalizeStreamEvent: not yet implemented")
	}

	chunk, ok := event.(*openaiStreamChunk)
	if !ok {
		return nil, fmt.Errorf("OpenAINormalizer.NormalizeStreamEvent: unexpected type %T", event)
	}

	evt := &api.ChatStreamEvent{
		Type: api.StreamEventDelta,
	}

	// Handle usage info (available when stream_options.include_usage is set)
	if chunk.Usage != nil {
		evt.Response = &api.ChatResponse{
			ID:    chunk.ID,
			Model: chunk.Model,
			Usage: api.Usage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			},
		}
	}

	if len(chunk.Choices) == 0 {
		return evt, nil
	}

	choice := chunk.Choices[0]

	// Check for finish reason (stream done for this choice)
	if choice.FinishReason != nil {
		fr := mapOpenAIFinishReason(*choice.FinishReason)
		evt.Done = true
		evt.Type = api.StreamEventDone
		if evt.Response == nil {
			evt.Response = &api.ChatResponse{
				ID:    chunk.ID,
				Model: chunk.Model,
			}
		}
		evt.Response.FinishReason = fr
		return evt, nil
	}

	// Process delta content
	delta := &api.MessageDelta{}
	hasDelta := false

	if choice.Delta.Role != "" {
		delta.Role = choice.Delta.Role
		hasDelta = true
		// First chunk with role — treat as start event
		evt.Type = api.StreamEventStart
	}

	if choice.Delta.Content != nil {
		delta.Content = *choice.Delta.Content
		hasDelta = true
		evt.Type = api.StreamEventDelta
	}

	if len(choice.Delta.ToolCalls) > 0 {
		evt.Type = api.StreamEventToolCall
		for _, tc := range choice.Delta.ToolCalls {
			delta.ToolCalls = append(delta.ToolCalls, api.ToolCallDelta{
				Index: choice.Index,
				ID:    tc.ID,
				Type:  tc.Type,
				Function: api.FunctionCallDelta{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
		hasDelta = true
	}

	if hasDelta {
		evt.Delta = delta
	}

	return evt, nil
}

func (n *OpenAINormalizer) NormalizeError(err error) *ProviderError {
	return NewProviderError("openai_error", err.Error(), ErrorTypeServer, err)
}

// mapOpenAIFinishReason maps OpenAI finish_reason values to standard finish reasons.
func mapOpenAIFinishReason(reason string) string {
	switch reason {
	case "stop":
		return "stop"
	case "length":
		return "length"
	case "tool_calls", "function_call":
		return "tool_calls"
	case "content_filter":
		return "content_filter"
	default:
		if reason == "" {
			return ""
		}
		return reason
	}
}
