package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/internal/mcp"
)

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]interface{}
	Handler     ToolHandler
}

type ToolHandler func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error)

type ToolRegistry struct {
	tools map[string]*Tool
	mu    sync.RWMutex
}

func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]*Tool),
	}
}

func (r *ToolRegistry) Register(tool *Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name]; exists {
		return fmt.Errorf("tool %s already registered", tool.Name)
	}

	r.tools[tool.Name] = tool
	return nil
}

func (r *ToolRegistry) Get(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

func (r *ToolRegistry) List() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		list = append(list, tool)
	}
	return list
}

func (r *ToolRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.tools, name)
}

func (s *Server) HandleListTools(req *mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	tools := s.toolRegistry.List()

	result := &mcp.ListToolsResult{
		Tools: make([]mcp.Tool, 0, len(tools)),
	}

	for _, tool := range tools {
		result.Tools = append(result.Tools, mcp.Tool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	return result, nil
}

func (s *Server) HandleCallTool(req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	tool, ok := s.toolRegistry.Get(req.Name)
	if !ok {
		return nil, fmt.Errorf("tool not found: %s", req.Name)
	}

	result, err := tool.Handler(s.ctx, req.Arguments)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				{
					Type: "text",
					Text: err.Error(),
				},
			},
			IsError: true,
		}, nil
	}

	return result, nil
}

func NewTool(name, description string, schema map[string]interface{}, handler ToolHandler) *Tool {
	return &Tool{
		Name:        name,
		Description: description,
		InputSchema: schema,
		Handler:     handler,
	}
}
