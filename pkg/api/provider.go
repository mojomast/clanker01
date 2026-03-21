package api

import (
	"context"
	"time"
)

// LLMProvider is the main interface for all LLM providers
type LLMProvider interface {
	// Identity
	Name() string
	Models() []ModelInfo

	// Chat Operations
	Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
	StreamChat(ctx context.Context, req *ChatRequest) (<-chan ChatStreamEvent, error)

	// Capabilities
	SupportsStreaming() bool
	SupportsFunctionCalling() bool
	SupportsVision() bool
	SupportsAudio() bool
	MaxTokens(model string) int

	// Configuration
	Configure(config *ProviderConfig) error

	// Metrics
	Metrics() *ProviderMetrics
}

// ModelInfo describes a model's capabilities
type ModelInfo struct {
	ID                string
	Alias             string
	MaxTokens         int
	MaxOutputTokens   int
	SupportsVision    bool
	SupportsAudio     bool
	SupportsStreaming bool
	SupportsTools     bool
	SupportsJSON      bool
	InputPricePer1K   float64
	OutputPricePer1K  float64
	ContextWindow     int
}

// ChatRequest is the universal request format
type ChatRequest struct {
	Model          string
	Messages       []Message
	Tools          []Tool
	ToolChoice     *ToolChoice
	Temperature    *float64
	TopP           *float64
	MaxTokens      int
	Stop           []string
	Stream         bool
	ResponseFormat *ResponseFormat
	Metadata       map[string]any
	SystemPrompt   string
}

// Message represents a single message in the conversation
type Message struct {
	Role       string
	Content    any
	ToolCalls  []ToolCall
	ToolCallID string
	Name       string
}

// ContentPart for multimodal messages
type ContentPart struct {
	Type     string
	Text     string
	ImageURL *ImageURL
	Audio    *AudioContent
	Video    *VideoContent
}

type ImageURL struct {
	URL    string
	Detail string
}

type AudioContent struct {
	Data   string
	Format string
}

type VideoContent struct {
	Data   string
	Format string
}

// Tool definition
type Tool struct {
	Type     string
	Function FunctionDef
}

type FunctionDef struct {
	Name        string
	Description string
	Parameters  map[string]any
}

type ToolChoice struct {
	Type     string
	Function *FunctionRef
}

type FunctionRef struct {
	Name string
}

// ToolCall represents a tool call from the assistant
type ToolCall struct {
	ID       string
	Type     string
	Function FunctionCall
}

type FunctionCall struct {
	Name      string
	Arguments string
}

// ChatResponse is the universal response format
type ChatResponse struct {
	ID           string
	Model        string
	Choices      []Choice
	Usage        Usage
	FinishReason string
	Created      int64
}

type Choice struct {
	Index        int
	Message      Message
	FinishReason string
	LogProbs     *LogProbs
}

type LogProbs struct {
	Content []TokenLogProb
}

type TokenLogProb struct {
	Token   string
	LogProb float64
	Bytes   []byte
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CachedTokens     int
}

// ChatStreamEvent for streaming responses
type ChatStreamEvent struct {
	Type     StreamEventType
	Delta    *MessageDelta
	Response *ChatResponse
	Error    error
	Done     bool
}

type StreamEventType string

const (
	StreamEventDelta    StreamEventType = "delta"
	StreamEventDone     StreamEventType = "done"
	StreamEventError    StreamEventType = "error"
	StreamEventToolCall StreamEventType = "tool_call"
)

type MessageDelta struct {
	Role      string
	Content   string
	ToolCalls []ToolCallDelta
}

type ToolCallDelta struct {
	Index    int
	ID       string
	Type     string
	Function FunctionCallDelta
}

type FunctionCallDelta struct {
	Name      string
	Arguments string
}

// ResponseFormat for structured outputs
type ResponseFormat struct {
	Type       string
	JSONSchema map[string]any
}

// ProviderConfig for provider initialization
type ProviderConfig struct {
	APIKey     string
	BaseURL    string
	Headers    map[string]string
	Timeout    time.Duration
	MaxRetries int
	RateLimit  *RateLimitConfig
	Models     []ModelInfo
}

type RateLimitConfig struct {
	RequestsPerMinute int
	TokensPerMinute   int
	RequestsPerDay    int
}

// ProviderMetrics tracks usage
type ProviderMetrics struct {
	TotalRequests     int64
	TotalTokens       int64
	TotalPromptTokens int64
	TotalOutputTokens int64
	TotalCost         float64
	Errors            int64
	AvgLatencyMs      float64
	RateLimitHits     int64
}
