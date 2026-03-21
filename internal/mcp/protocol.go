package mcp

import (
	"encoding/json"
)

const (
	ProtocolVersion = "2024-11-05"
)

type JSONRPCMessage struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
}

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

const (
	ParseError       = -32700
	InvalidRequest   = -32600
	MethodNotFound   = -32601
	InvalidParams    = -32602
	InternalError    = -32603
	Unauthorized     = -32001
	Forbidden        = -32002
	RateLimited      = -32003
	Timeout          = -32004
	ResourceNotFound = -32005
	ToolError        = -32006
)

type Implementation struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ClientCapabilities struct {
	Roots    *RootsCapability    `json:"roots,omitempty"`
	Sampling *SamplingCapability `json:"sampling,omitempty"`
}

type RootsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type SamplingCapability struct {
}

type ServerCapabilities struct {
	Prompts   *PromptsCapability   `json:"prompts,omitempty"`
	Resources *ResourcesCapability `json:"resources,omitempty"`
	Tools     *ToolsCapability     `json:"tools,omitempty"`
	Logging   *LoggingCapability   `json:"logging,omitempty"`
}

type PromptsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type ResourcesCapability struct {
	Subscribe   bool `json:"subscribe,omitempty"`
	ListChanged bool `json:"listChanged,omitempty"`
}

type ToolsCapability struct {
	ListChanged bool `json:"listChanged,omitempty"`
}

type LoggingCapability struct {
}

type InitializeRequest struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ClientCapabilities `json:"capabilities"`
	ClientInfo      Implementation     `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string             `json:"protocolVersion"`
	Capabilities    ServerCapabilities `json:"capabilities"`
	ServerInfo      Implementation     `json:"serverInfo"`
	Instructions    string             `json:"instructions,omitempty"`
}

type InitializedNotification struct {
}

type ListPromptsRequest struct {
	Cursor *string `json:"cursor,omitempty"`
}

type ListPromptsResult struct {
	NextCursor *string  `json:"nextCursor,omitempty"`
	Prompts    []Prompt `json:"prompts"`
}

type Prompt struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Arguments   []PromptArgument `json:"arguments,omitempty"`
}

type PromptArgument struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type GetPromptRequest struct {
	Name      string            `json:"name"`
	Arguments map[string]string `json:"arguments,omitempty"`
}

type GetPromptResult struct {
	Description string    `json:"description,omitempty"`
	Messages    []Message `json:"messages"`
}

type Message struct {
	Role    string  `json:"role"`
	Content Content `json:"content"`
}

type ListResourcesRequest struct {
	Cursor *string `json:"cursor,omitempty"`
}

type ListResourcesResult struct {
	NextCursor *string    `json:"nextCursor,omitempty"`
	Resources  []Resource `json:"resources"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ReadResourceRequest struct {
	URI string `json:"uri"`
}

type ReadResourceResult struct {
	Contents []ResourceContents `json:"contents"`
}

type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     []byte `json:"blob,omitempty"`
}

type ListToolsRequest struct {
	Cursor *string `json:"cursor,omitempty"`
}

type ListToolsResult struct {
	NextCursor *string `json:"nextCursor,omitempty"`
	Tools      []Tool  `json:"tools"`
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type CallToolRequest struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []Content `json:"content"`
	IsError bool      `json:"isError,omitempty"`
}

type Content struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	Data     string `json:"data,omitempty"`
	MimeType string `json:"mimeType,omitempty"`
}

type SetLevelRequest struct {
	Level string `json:"level"`
}

type CompleteRequest struct {
	Ref      RequestReference  `json:"ref"`
	Argument *CompleteArgument `json:"argument,omitempty"`
}

type RequestReference struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

type CompleteArgument struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type CompleteResult struct {
	Completion struct {
		Values  []string `json:"values,omitempty"`
		HasMore bool     `json:"hasMore,omitempty"`
		Total   *int     `json:"total,omitempty"`
	} `json:"completion"`
}

type ResourceUpdatedNotification struct {
	URI string `json:"uri"`
}

type ResourceListChangedNotification struct {
}

type ToolsListChangedNotification struct {
}

type PromptsListChangedNotification struct {
}

type LogMessage struct {
	Level  string  `json:"level"`
	Data   string  `json:"data"`
	Logger *string `json:"logger,omitempty"`
}
