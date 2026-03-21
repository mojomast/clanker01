package providers

import (
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

func (n *OpenAINormalizer) NormalizeResponse(resp any) (*api.ChatResponse, error) {
	return &api.ChatResponse{}, nil
}

func (n *OpenAINormalizer) NormalizeStreamEvent(event any) (*api.ChatStreamEvent, error) {
	return &api.ChatStreamEvent{}, nil
}

func (n *OpenAINormalizer) NormalizeError(err error) *ProviderError {
	return NewProviderError("openai_error", err.Error(), ErrorTypeServer, err)
}
