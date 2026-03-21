package servers

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/internal/mcp"
	"github.com/swarm-ai/swarm/internal/mcp/server"
)

type MemoryServer struct {
	mcpServer *server.Server
	data      map[string]interface{}
	mu        sync.RWMutex
}

func NewMemoryServer() *MemoryServer {
	s := &MemoryServer{
		data: make(map[string]interface{}),
	}
	s.mcpServer = server.NewServer("memory", "1.0.0")
	s.registerTools()
	s.registerResources()
	return s
}

func (s *MemoryServer) registerTools() {
	s.mcpServer.RegisterTool(server.NewTool(
		"memory_set",
		"Set a value in memory",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Key to set",
				},
				"value": map[string]interface{}{
					"type":        "any",
					"description": "Value to store",
				},
			},
			"required": []string{"key", "value"},
		},
		s.memorySet,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"memory_get",
		"Get a value from memory",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Key to get",
				},
			},
			"required": []string{"key"},
		},
		s.memoryGet,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"memory_delete",
		"Delete a value from memory",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"key": map[string]interface{}{
					"type":        "string",
					"description": "Key to delete",
				},
			},
			"required": []string{"key"},
		},
		s.memoryDelete,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"memory_list",
		"List all keys in memory",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		s.memoryList,
	))
}

func (s *MemoryServer) registerResources() {
	s.mcpServer.RegisterResource(server.NewDynamicResource(
		"memory://all",
		"All Memory",
		"All keys and values in memory",
		"application/json",
		func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
			s.mu.RLock()
			defer s.mu.RUnlock()

			data, _ := json.MarshalIndent(s.data, "", "  ")
			return &mcp.ReadResourceResult{
				Contents: []mcp.ResourceContents{
					{
						URI:      uri,
						MimeType: "application/json",
						Text:     string(data),
					},
				},
			}, nil
		},
	))
}

func (s *MemoryServer) memorySet(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	key, ok := args["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key must be a string")
	}
	value := args["value"]

	s.mu.Lock()
	s.data[key] = value
	s.mu.Unlock()

	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: "OK"}},
	}, nil
}

func (s *MemoryServer) memoryGet(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	key, ok := args["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key must be a string")
	}

	s.mu.RLock()
	value, ok := s.data[key]
	s.mu.RUnlock()

	if !ok {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: "key not found"}},
			IsError: true,
		}, nil
	}

	data, _ := json.MarshalIndent(value, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MemoryServer) memoryDelete(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	key, ok := args["key"].(string)
	if !ok {
		return nil, fmt.Errorf("key must be a string")
	}

	s.mu.Lock()
	delete(s.data, key)
	s.mu.Unlock()

	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: "OK"}},
	}, nil
}

func (s *MemoryServer) memoryList(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys := make([]string, 0, len(s.data))
	for k := range s.data {
		keys = append(keys, k)
	}

	data, _ := json.MarshalIndent(keys, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *MemoryServer) Serve(ctx context.Context, transport server.Transport) error {
	return s.mcpServer.Serve(ctx, transport)
}

func (s *MemoryServer) Server() *server.Server {
	return s.mcpServer
}
