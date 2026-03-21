package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/swarm-ai/swarm/internal/mcp"
)

type Server struct {
	name             string
	version          string
	capabilities     mcp.ServerCapabilities
	promptRegistry   *PromptRegistry
	resourceRegistry *ResourceRegistry
	toolRegistry     *ToolRegistry
	transport        Transport
	ctx              context.Context
	initialized      bool
}

type Transport interface {
	Send(ctx context.Context, msg json.RawMessage) error
	Receive(ctx context.Context) (json.RawMessage, error)
	Close() error
}

func NewServer(name, version string) *Server {
	return &Server{
		name:    name,
		version: version,
		capabilities: mcp.ServerCapabilities{
			Prompts: &mcp.PromptsCapability{
				ListChanged: true,
			},
			Resources: &mcp.ResourcesCapability{
				Subscribe:   true,
				ListChanged: true,
			},
			Tools: &mcp.ToolsCapability{
				ListChanged: true,
			},
			Logging: &mcp.LoggingCapability{},
		},
		promptRegistry:   NewPromptRegistry(),
		resourceRegistry: NewResourceRegistry(),
		toolRegistry:     NewToolRegistry(),
	}
}

func (s *Server) Serve(ctx context.Context, transport Transport) error {
	s.ctx = ctx
	s.transport = transport

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			msg, err := transport.Receive(ctx)
			if err != nil {
				return fmt.Errorf("receive error: %w", err)
			}

			go s.handleMessage(msg)
		}
	}
}

func (s *Server) handleMessage(msg json.RawMessage) {
	var base struct {
		JSONRPC string          `json:"jsonrpc"`
		ID      json.RawMessage `json:"id,omitempty"`
		Method  string          `json:"method,omitempty"`
	}

	if err := json.Unmarshal(msg, &base); err != nil {
		s.sendError(nil, mcp.ParseError, "invalid JSON-RPC message")
		return
	}

	if base.Method == "" {
		s.sendError(base.ID, mcp.InvalidRequest, "method is required")
		return
	}

	var result interface{}
	var err error

	switch base.Method {
	case "initialize":
		result, err = s.handleInitialize(msg)
	case "initialized":
		s.initialized = true
		return
	case "prompts/list":
		result, err = s.HandleListPrompts(&mcp.ListPromptsRequest{})
	case "prompts/get":
		var req mcp.GetPromptRequest
		if params, ok := extractParamsFromMessage(msg); ok {
			if e := json.Unmarshal(params, &req); e == nil {
				result, err = s.HandleGetPrompt(&req)
			} else {
				err = e
			}
		}
	case "resources/list":
		result, err = s.HandleListResources(&mcp.ListResourcesRequest{})
	case "resources/read":
		var req mcp.ReadResourceRequest
		if params, ok := extractParamsFromMessage(msg); ok {
			if e := json.Unmarshal(params, &req); e == nil {
				result, err = s.HandleReadResource(&req)
			} else {
				err = e
			}
		}
	case "tools/list":
		result, err = s.HandleListTools(&mcp.ListToolsRequest{})
	case "tools/call":
		var req mcp.CallToolRequest
		if params, ok := extractParamsFromMessage(msg); ok {
			if e := json.Unmarshal(params, &req); e == nil {
				result, err = s.HandleCallTool(&req)
			} else {
				err = e
			}
		}
	case "logging/setLevel":
		var req mcp.SetLevelRequest
		if params, ok := extractParamsFromMessage(msg); ok {
			if e := json.Unmarshal(params, &req); e == nil {
				s.handleSetLevel(&req)
			} else {
				err = e
			}
		}
	default:
		s.sendError(base.ID, mcp.MethodNotFound, fmt.Sprintf("unknown method: %s", base.Method))
		return
	}

	if err != nil {
		s.sendError(base.ID, mcp.InternalError, err.Error())
		return
	}

	if base.ID != nil {
		s.sendSuccess(base.ID, result)
	}
}

func extractParamsFromMessage(msg json.RawMessage) (json.RawMessage, bool) {
	var parsed map[string]json.RawMessage
	if err := json.Unmarshal(msg, &parsed); err != nil {
		return nil, false
	}
	params, ok := parsed["params"]
	if !ok {
		return nil, false
	}
	return params, true
}

func (s *Server) handleInitialize(msg json.RawMessage) (*mcp.InitializeResult, error) {
	var req mcp.InitializeRequest
	if err := json.Unmarshal(msg, &req); err != nil {
		return nil, fmt.Errorf("invalid initialize request: %w", err)
	}

	log.Printf("Client connected: %s %s", req.ClientInfo.Name, req.ClientInfo.Version)

	return &mcp.InitializeResult{
		ProtocolVersion: mcp.ProtocolVersion,
		Capabilities:    s.capabilities,
		ServerInfo: mcp.Implementation{
			Name:    s.name,
			Version: s.version,
		},
	}, nil
}

func (s *Server) handleSetLevel(req *mcp.SetLevelRequest) {
	log.Printf("Log level set to: %s", req.Level)
}

func (s *Server) sendSuccess(id json.RawMessage, result interface{}) {
	resp := mcp.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
	}

	if result != nil {
		resultBytes, _ := json.Marshal(result)
		resp.Result = json.RawMessage(resultBytes)
	}

	msgBytes, _ := json.Marshal(resp)
	s.transport.Send(s.ctx, json.RawMessage(msgBytes))
}

func (s *Server) sendError(id json.RawMessage, code int, message string) {
	resp := mcp.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      id,
		Error: &mcp.Error{
			Code:    code,
			Message: message,
		},
	}

	msgBytes, _ := json.Marshal(resp)
	s.transport.Send(s.ctx, json.RawMessage(msgBytes))
}

func (s *Server) RegisterTool(tool *Tool) error {
	return s.toolRegistry.Register(tool)
}

func (s *Server) RegisterResource(resource *Resource) error {
	return s.resourceRegistry.Register(resource)
}

func (s *Server) RegisterPrompt(prompt *Prompt) error {
	return s.promptRegistry.Register(prompt)
}

func (s *Server) SendNotification(method string, params interface{}) error {
	if !s.initialized {
		return fmt.Errorf("server not initialized")
	}

	req := mcp.JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  method,
	}

	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err == nil {
			req.Params = json.RawMessage(paramsBytes)
		}
	}

	msgBytes, _ := json.Marshal(req)
	return s.transport.Send(s.ctx, json.RawMessage(msgBytes))
}

func (s *Server) SendLogMessage(level, data string) {
	logMsg := mcp.LogMessage{
		Level: level,
		Data:  data,
	}
	s.SendNotification("notifications/message", logMsg)
}

func (s *Server) NotifyResourceListChanged() {
	s.SendNotification("notifications/resources/list_changed", nil)
}

func (s *Server) NotifyToolListChanged() {
	s.SendNotification("notifications/tools/list_changed", nil)
}

func (s *Server) NotifyPromptListChanged() {
	s.SendNotification("notifications/prompts/list_changed", nil)
}
