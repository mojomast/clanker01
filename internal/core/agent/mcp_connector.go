package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/pkg/api"
)

type MCPConnector struct {
	mu      sync.RWMutex
	servers map[string]MCPServer
}

type MCPServer interface {
	GetTools() []api.Tool
	CallTool(ctx context.Context, toolCall api.ToolCall) (string, error)
}

func NewMCPConnector() *MCPConnector {
	return &MCPConnector{
		servers: make(map[string]MCPServer),
	}
}

func (m *MCPConnector) RegisterServer(name string, server MCPServer) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.servers[name] = server
}

func (m *MCPConnector) UnregisterServer(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.servers, name)
}

func (m *MCPConnector) GetAvailableTools() []api.Tool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var tools []api.Tool
	for _, server := range m.servers {
		tools = append(tools, server.GetTools()...)
	}
	return tools
}

func (m *MCPConnector) ExecuteTool(ctx context.Context, toolCall api.ToolCall) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, server := range m.servers {
		tools := server.GetTools()
		for _, tool := range tools {
			if tool.Function.Name == toolCall.Function.Name {
				return server.CallTool(ctx, toolCall)
			}
		}
	}
	return "", fmt.Errorf("tool not found: %s", toolCall.Function.Name)
}
