package servers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/swarm-ai/swarm/internal/mcp"
	"github.com/swarm-ai/swarm/internal/mcp/server"
)

type HTTPServer struct {
	mcpServer *server.Server
	client    *http.Client
}

func NewHTTPServer() *HTTPServer {
	s := &HTTPServer{
		client: &http.Client{Timeout: 30 * time.Second},
	}
	s.mcpServer = server.NewServer("http", "1.0.0")
	s.registerTools()
	return s
}

func (s *HTTPServer) registerTools() {
	s.mcpServer.RegisterTool(server.NewTool(
		"http_get",
		"Make a GET request",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to request",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "Request headers",
				},
			},
			"required": []string{"url"},
		},
		s.httpGet,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"http_post",
		"Make a POST request",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "URL to request",
				},
				"headers": map[string]interface{}{
					"type":        "object",
					"description": "Request headers",
				},
				"body": map[string]interface{}{
					"type":        "object",
					"description": "Request body (JSON)",
				},
			},
			"required": []string{"url"},
		},
		s.httpPost,
	))
}

func (s *HTTPServer) httpGet(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	if headers, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{Type: "text", Text: fmt.Sprintf("Status: %s\n\n%s", resp.Status, string(body))},
		},
	}, nil
}

func (s *HTTPServer) httpPost(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	url, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url must be a string")
	}

	var bodyReader io.Reader
	if b, ok := args["body"].(map[string]interface{}); ok {
		data, _ := json.Marshal(b)
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bodyReader)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	req.Header.Set("Content-Type", "application/json")
	if headers, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprint(v))
		}
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{Type: "text", Text: fmt.Sprintf("Status: %s\n\n%s", resp.Status, string(respBody))},
		},
	}, nil
}

func (s *HTTPServer) Serve(ctx context.Context, transport server.Transport) error {
	return s.mcpServer.Serve(ctx, transport)
}

func (s *HTTPServer) Server() *server.Server {
	return s.mcpServer
}
