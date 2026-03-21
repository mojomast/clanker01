package servers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/swarm-ai/swarm/internal/mcp"
	"github.com/swarm-ai/swarm/internal/mcp/server"
)

type FilesystemServer struct {
	mcpServer *server.Server
	root      string
	allowed   []string
}

func NewFilesystemServer(root string, allowedPaths []string) *FilesystemServer {
	s := &FilesystemServer{
		root:    root,
		allowed: allowedPaths,
	}

	s.mcpServer = server.NewServer("filesystem", "1.0.0")
	s.registerTools()
	s.registerResources()

	return s
}

func (s *FilesystemServer) registerTools() {
	s.mcpServer.RegisterTool(server.NewTool(
		"read_file",
		"Read contents of a file",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file",
				},
			},
			"required": []string{"path"},
		},
		s.readFile,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"write_file",
		"Write contents to a file",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the file",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "Content to write",
				},
			},
			"required": []string{"path", "content"},
		},
		s.writeFile,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"list_directory",
		"List contents of a directory",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Path to the directory",
				},
			},
			"required": []string{"path"},
		},
		s.listDirectory,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"search_files",
		"Search for files matching a pattern",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"path": map[string]interface{}{
					"type":        "string",
					"description": "Base path for search",
				},
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern to match",
				},
			},
			"required": []string{"path", "pattern"},
		},
		s.searchFiles,
	))
}

func (s *FilesystemServer) registerResources() {
	s.mcpServer.RegisterResource(server.NewDynamicResource(
		"file://root",
		"Filesystem Root",
		"Root directory of the filesystem server",
		"text/plain",
		func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{
				Contents: []mcp.ResourceContents{
					{
						URI:  uri,
						Text: s.root,
					},
				},
			}, nil
		},
	))
}

func (s *FilesystemServer) isAllowed(path string) bool {
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	// Resolve symlinks to prevent path traversal attacks
	resolved, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return false
	}

	for _, allowed := range s.allowed {
		allowedResolved, err := filepath.EvalSymlinks(allowed)
		if err != nil {
			continue
		}
		if strings.HasPrefix(resolved, allowedResolved) {
			return true
		}
	}
	return false
}

func (s *FilesystemServer) readFile(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	if !s.isAllowed(path) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: "access denied"}},
			IsError: true,
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: string(data)}},
	}, nil
}

func (s *FilesystemServer) writeFile(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	content, ok := args["content"].(string)
	if !ok {
		return nil, fmt.Errorf("content must be a string")
	}

	if !s.isAllowed(path) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: "access denied"}},
			IsError: true,
		}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: "OK"}},
	}, nil
}

func (s *FilesystemServer) listDirectory(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	if !s.isAllowed(path) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: "access denied"}},
			IsError: true,
		}, nil
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	var result strings.Builder
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			result.WriteString(fmt.Sprintf("? %s (error: %v)\n", entry.Name(), err))
			continue
		}
		result.WriteString(fmt.Sprintf("%s %s\n", info.Mode(), entry.Name()))
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: result.String()}},
	}, nil
}

func (s *FilesystemServer) searchFiles(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	path, ok := args["path"].(string)
	if !ok {
		return nil, fmt.Errorf("path must be a string")
	}

	pattern, ok := args["pattern"].(string)
	if !ok {
		return nil, fmt.Errorf("pattern must be a string")
	}

	if !s.isAllowed(path) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: "access denied"}},
			IsError: true,
		}, nil
	}

	matches, err := filepath.Glob(filepath.Join(path, pattern))
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{Type: "text", Text: strings.Join(matches, "\n")},
		},
	}, nil
}

func (s *FilesystemServer) Serve(ctx context.Context, transport server.Transport) error {
	return s.mcpServer.Serve(ctx, transport)
}

func (s *FilesystemServer) Server() *server.Server {
	return s.mcpServer
}
