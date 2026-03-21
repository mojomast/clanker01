package providers

import (
	"encoding/json"
	"fmt"

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

// TODO: NormalizeResponse needs real implementation to parse Anthropic API response
// format into the standard ChatResponse struct.
func (n *AnthropicNormalizer) NormalizeResponse(resp any) (*api.ChatResponse, error) {
	return nil, fmt.Errorf("AnthropicNormalizer.NormalizeResponse not yet implemented")
}

// TODO: NormalizeStreamEvent needs real implementation to parse Anthropic streaming
// event format into the standard ChatStreamEvent struct.
func (n *AnthropicNormalizer) NormalizeStreamEvent(event any) (*api.ChatStreamEvent, error) {
	return nil, fmt.Errorf("AnthropicNormalizer.NormalizeStreamEvent not yet implemented")
}

func (n *AnthropicNormalizer) NormalizeError(err error) *ProviderError {
	return NewProviderError("anthropic_error", err.Error(), ErrorTypeServer, err)
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
