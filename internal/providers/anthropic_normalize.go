package providers

import (
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

func (n *AnthropicNormalizer) NormalizeResponse(resp any) (*api.ChatResponse, error) {
	return &api.ChatResponse{}, nil
}

func (n *AnthropicNormalizer) NormalizeStreamEvent(event any) (*api.ChatStreamEvent, error) {
	return &api.ChatStreamEvent{}, nil
}

func (n *AnthropicNormalizer) NormalizeError(err error) *ProviderError {
	return NewProviderError("anthropic_error", err.Error(), ErrorTypeServer, err)
}

func detectMediaType(url string) string {
	if len(url) > 10 && url[:10] == "data:image/" {
		for i := 10; i < len(url); i++ {
			if url[i] == ';' {
				return "image/" + url[10:i]
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
	return map[string]any{}
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
