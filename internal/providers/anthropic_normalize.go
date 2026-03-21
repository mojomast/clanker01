package providers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/swarm-ai/swarm/pkg/api"
)

// AnthropicNormalizer converts between SWARM and Anthropic formats
type AnthropicNormalizer struct{}

type anthropicRequest struct {
	Model       string             `json:"model"`
	Messages    []anthropicMessage `json:"messages"`
	System      string             `json:"system,omitempty"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	TopP        float64            `json:"top_p,omitempty"`
	Tools       []anthropicTool    `json:"tools,omitempty"`
	ToolChoice  any                `json:"tool_choice,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string             `json:"role"`
	Content []anthropicContent `json:"content"`
}

type anthropicContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`

	ID        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	ToolUseID string `json:"tool_use_id,omitempty"`
	Content_  string `json:"content,omitempty"`

	Source *anthropicSource `json:"source,omitempty"`
}

type anthropicSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type"`
	Data      string `json:"data"`
}

type anthropicTool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"input_schema"`
}

// --- Anthropic API response structs ---

// anthropicResponse maps to the Anthropic /v1/messages response body.
type anthropicResponse struct {
	ID           string             `json:"id"`
	Type         string             `json:"type"`
	Role         string             `json:"role"`
	Content      []anthropicContent `json:"content"`
	Model        string             `json:"model"`
	StopReason   string             `json:"stop_reason"`
	StopSequence *string            `json:"stop_sequence"`
	Usage        anthropicUsage     `json:"usage"`
}

// anthropicUsage contains token usage data from the Anthropic API.
type anthropicUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

// anthropicErrorResponse wraps an API error from Anthropic.
type anthropicErrorResponse struct {
	Type  string            `json:"type"`
	Error anthropicAPIError `json:"error"`
}

// anthropicAPIError describes a typed error from the Anthropic API.
type anthropicAPIError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// toProviderError converts an Anthropic API error into a ProviderError.
func (e *anthropicAPIError) toProviderError(statusCode int) *ProviderError {
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
	case e.Type == "overloaded_error":
		errType = ErrorTypeServer
	case e.Type == "invalid_request_error":
		errType = ErrorTypeInvalid
	case e.Type == "authentication_error":
		errType = ErrorTypeAuth
	case e.Type == "rate_limit_error":
		errType = ErrorTypeRateLimit
	case e.Type == "not_found_error":
		errType = ErrorTypeNotFound
	}

	pe := NewProviderError(e.Type, e.Message, errType, fmt.Errorf("anthropic: %s", e.Message))
	if errType == ErrorTypeRateLimit {
		pe.RetryAfter = 5 * time.Second
	}
	return pe
}

// anthropicStreamEvent represents a single SSE event from the Anthropic streaming API.
type anthropicStreamEvent struct {
	Type         string                `json:"type"`
	Index        int                   `json:"index,omitempty"`
	Message      *anthropicResponse    `json:"message,omitempty"`
	ContentBlock *anthropicContent     `json:"content_block,omitempty"`
	Delta        *anthropicStreamDelta `json:"delta,omitempty"`
	Usage        *anthropicUsage       `json:"usage,omitempty"`
	Error        *anthropicAPIError    `json:"error,omitempty"`
}

// anthropicStreamDelta carries incremental content in streaming events.
type anthropicStreamDelta struct {
	Type        string `json:"type,omitempty"`
	Text        string `json:"text,omitempty"`
	PartialJSON string `json:"partial_json,omitempty"`
	StopReason  string `json:"stop_reason,omitempty"`
}

func (n *AnthropicNormalizer) NormalizeRequest(req *api.ChatRequest) (any, error) {
	ar := &anthropicRequest{
		Model:     req.Model,
		MaxTokens: req.MaxTokens,
		System:    req.SystemPrompt,
	}

	if req.Temperature != nil {
		ar.Temperature = *req.Temperature
	}

	if req.TopP != nil {
		ar.TopP = *req.TopP
	}

	for _, msg := range req.Messages {
		// Skip system messages - Anthropic doesn't accept "system" role in
		// the messages array; the system prompt is set via the top-level field.
		if msg.Role == "system" {
			continue
		}

		am := anthropicMessage{Role: msg.Role}

		switch v := msg.Content.(type) {
		case string:
			am.Content = []anthropicContent{{
				Type: "text",
				Text: v,
			}}
		case []api.ContentPart:
			for _, part := range v {
				ac := anthropicContent{Type: "text"}
				switch part.Type {
				case "text":
					ac.Text = part.Text
				case "image":
					ac.Type = "image"
					ac.Source = &anthropicSource{
						Type:      "base64",
						MediaType: detectMediaType(part.ImageURL.URL),
						Data:      extractBase64(part.ImageURL.URL),
					}
				}
				am.Content = append(am.Content, ac)
			}
		}

		if len(msg.ToolCalls) > 0 {
			for _, tc := range msg.ToolCalls {
				var input any
				if tc.Function.Arguments != "" {
					input = parseJSON(tc.Function.Arguments)
				}
				am.Content = append(am.Content, anthropicContent{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
		}

		if msg.Role == "tool" {
			am.Content = []anthropicContent{{
				Type:      "tool_result",
				ToolUseID: msg.ToolCallID,
				Content_:  toString(msg.Content),
			}}
		}

		ar.Messages = append(ar.Messages, am)
	}

	for _, tool := range req.Tools {
		ar.Tools = append(ar.Tools, anthropicTool{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			InputSchema: tool.Function.Parameters,
		})
	}

	if req.ToolChoice != nil {
		switch req.ToolChoice.Type {
		case "auto":
			ar.ToolChoice = map[string]string{"type": "auto"}
		case "none":
			ar.ToolChoice = map[string]string{"type": "none"}
		case "required":
			ar.ToolChoice = map[string]string{"type": "any"}
		case "function":
			ar.ToolChoice = map[string]any{
				"type": "tool",
				"name": req.ToolChoice.Function.Name,
			}
		}
	}

	return ar, nil
}

// NormalizeResponse converts an Anthropic API response (*anthropicResponse) to a
// unified *api.ChatResponse. Returns an error if resp is nil or not the expected type.
func (n *AnthropicNormalizer) NormalizeResponse(resp any) (*api.ChatResponse, error) {
	if resp == nil {
		return nil, fmt.Errorf("AnthropicNormalizer.NormalizeResponse: not yet implemented")
	}

	ar, ok := resp.(*anthropicResponse)
	if !ok {
		return nil, fmt.Errorf("AnthropicNormalizer.NormalizeResponse: unexpected type %T", resp)
	}

	msg := api.Message{
		Role: ar.Role,
	}

	// Build text content and tool calls from content blocks
	var textParts []string
	for _, block := range ar.Content {
		switch block.Type {
		case "text":
			textParts = append(textParts, block.Text)
		case "tool_use":
			argsJSON, _ := json.Marshal(block.Input)
			msg.ToolCalls = append(msg.ToolCalls, api.ToolCall{
				ID:   block.ID,
				Type: "function",
				Function: api.FunctionCall{
					Name:      block.Name,
					Arguments: string(argsJSON),
				},
			})
		}
	}

	// Combine text parts into a single content string
	if len(textParts) > 0 {
		combined := ""
		for i, part := range textParts {
			if i > 0 {
				combined += "\n"
			}
			combined += part
		}
		msg.Content = combined
	}

	// Map Anthropic stop_reason to standard finish reasons
	finishReason := mapAnthropicStopReason(ar.StopReason)

	chatResp := &api.ChatResponse{
		ID:           ar.ID,
		Model:        ar.Model,
		FinishReason: finishReason,
		Choices: []api.Choice{
			{
				Index:        0,
				Message:      msg,
				FinishReason: finishReason,
			},
		},
		Usage: api.Usage{
			PromptTokens:     ar.Usage.InputTokens,
			CompletionTokens: ar.Usage.OutputTokens,
			TotalTokens:      ar.Usage.InputTokens + ar.Usage.OutputTokens,
			CachedTokens:     ar.Usage.CacheReadInputTokens,
		},
	}

	return chatResp, nil
}

// NormalizeStreamEvent converts an Anthropic streaming event (*anthropicStreamEvent)
// to a unified *api.ChatStreamEvent. Returns error if event is nil or wrong type.
func (n *AnthropicNormalizer) NormalizeStreamEvent(event any) (*api.ChatStreamEvent, error) {
	if event == nil {
		return nil, fmt.Errorf("AnthropicNormalizer.NormalizeStreamEvent: not yet implemented")
	}

	se, ok := event.(*anthropicStreamEvent)
	if !ok {
		return nil, fmt.Errorf("AnthropicNormalizer.NormalizeStreamEvent: unexpected type %T", event)
	}

	switch se.Type {
	case "message_start":
		evt := &api.ChatStreamEvent{
			Type: api.StreamEventStart,
		}
		if se.Message != nil {
			evt.Response = &api.ChatResponse{
				ID:    se.Message.ID,
				Model: se.Message.Model,
				Usage: api.Usage{
					PromptTokens: se.Message.Usage.InputTokens,
					TotalTokens:  se.Message.Usage.InputTokens,
				},
			}
		}
		return evt, nil

	case "content_block_start":
		evt := &api.ChatStreamEvent{
			Type: api.StreamEventDelta,
		}
		if se.ContentBlock != nil {
			switch se.ContentBlock.Type {
			case "tool_use":
				evt.Type = api.StreamEventToolCall
				evt.Delta = &api.MessageDelta{
					ToolCalls: []api.ToolCallDelta{
						{
							Index: se.Index,
							ID:    se.ContentBlock.ID,
							Type:  "function",
							Function: api.FunctionCallDelta{
								Name: se.ContentBlock.Name,
							},
						},
					},
				}
			case "text":
				// Text block starting — no delta content yet
				evt.Delta = &api.MessageDelta{}
			}
		}
		return evt, nil

	case "content_block_delta":
		evt := &api.ChatStreamEvent{
			Type: api.StreamEventDelta,
		}
		if se.Delta != nil {
			switch se.Delta.Type {
			case "text_delta":
				evt.Delta = &api.MessageDelta{
					Content: se.Delta.Text,
				}
			case "input_json_delta":
				evt.Type = api.StreamEventToolCall
				evt.Delta = &api.MessageDelta{
					ToolCalls: []api.ToolCallDelta{
						{
							Index: se.Index,
							Function: api.FunctionCallDelta{
								Arguments: se.Delta.PartialJSON,
							},
						},
					},
				}
			}
		}
		return evt, nil

	case "content_block_stop":
		// Content block finished — informational, pass through as delta
		return &api.ChatStreamEvent{
			Type: api.StreamEventDelta,
		}, nil

	case "message_delta":
		evt := &api.ChatStreamEvent{
			Type: api.StreamEventDelta,
		}
		if se.Delta != nil && se.Delta.StopReason != "" {
			evt.Response = &api.ChatResponse{
				FinishReason: mapAnthropicStopReason(se.Delta.StopReason),
			}
		}
		if se.Usage != nil {
			if evt.Response == nil {
				evt.Response = &api.ChatResponse{}
			}
			evt.Response.Usage = api.Usage{
				CompletionTokens: se.Usage.OutputTokens,
				TotalTokens:      se.Usage.OutputTokens,
			}
		}
		return evt, nil

	case "message_stop":
		return &api.ChatStreamEvent{
			Type: api.StreamEventDone,
			Done: true,
		}, nil

	case "ping":
		// Heartbeat; return a no-op delta event
		return &api.ChatStreamEvent{
			Type: api.StreamEventDelta,
		}, nil

	case "error":
		errMsg := "unknown error"
		if se.Error != nil {
			errMsg = se.Error.Message
		}
		return &api.ChatStreamEvent{
			Type:  api.StreamEventError,
			Error: fmt.Errorf("anthropic stream error: %s", errMsg),
		}, nil

	default:
		// Unknown event type — treat as no-op
		return &api.ChatStreamEvent{
			Type: api.StreamEventDelta,
		}, nil
	}
}

func (n *AnthropicNormalizer) NormalizeError(err error) *ProviderError {
	return NewProviderError("anthropic_error", err.Error(), ErrorTypeServer, err)
}

// mapAnthropicStopReason maps Anthropic stop_reason values to standard finish reasons.
func mapAnthropicStopReason(reason string) string {
	switch reason {
	case "end_turn":
		return "stop"
	case "max_tokens":
		return "length"
	case "stop_sequence":
		return "stop"
	case "tool_use":
		return "tool_calls"
	default:
		if reason == "" {
			return ""
		}
		return reason
	}
}

func detectMediaType(url string) string {
	// "data:image/" is 11 chars, check len > 11 to have at least one char after prefix
	if len(url) > 11 && url[:11] == "data:image/" {
		for i := 11; i < len(url); i++ {
			if url[i] == ';' {
				return "image/" + url[11:i]
			}
		}
	}
	return "image/jpeg"
}

func extractBase64(url string) string {
	for i := 0; i < len(url)-7; i++ {
		if url[i:i+7] == "base64," {
			return url[i+7:]
		}
	}
	return url
}

func parseJSON(s string) any {
	var result any
	if err := json.Unmarshal([]byte(s), &result); err != nil {
		// If parsing fails, return the raw string wrapped in a map
		return map[string]any{"raw": s}
	}
	return result
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
