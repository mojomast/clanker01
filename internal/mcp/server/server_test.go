package server

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/swarm-ai/swarm/internal/mcp"
)

type mockServerTransport struct {
	sendQueue    chan json.RawMessage
	receiveQueue chan json.RawMessage
	closed       bool
}

func newMockServerTransport() *mockServerTransport {
	return &mockServerTransport{
		sendQueue:    make(chan json.RawMessage, 100),
		receiveQueue: make(chan json.RawMessage, 100),
	}
}

func (m *mockServerTransport) Send(ctx context.Context, msg json.RawMessage) error {
	if m.closed {
		return nil
	}
	m.sendQueue <- msg
	return nil
}

func (m *mockServerTransport) Receive(ctx context.Context) (json.RawMessage, error) {
	if m.closed {
		return nil, nil
	}
	select {
	case msg := <-m.receiveQueue:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (m *mockServerTransport) Close() error {
	m.closed = true
	close(m.sendQueue)
	close(m.receiveQueue)
	return nil
}

func (m *mockServerTransport) sendRequest(method string, params interface{}) error {
	req := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
	}
	if params != nil {
		req["params"] = params
	}
	data, _ := json.Marshal(req)
	m.receiveQueue <- json.RawMessage(data)
	return nil
}

func (m *mockServerTransport) readResponse() (map[string]interface{}, error) {
	select {
	case msg := <-m.sendQueue:
		var resp map[string]interface{}
		if err := json.Unmarshal(msg, &resp); err != nil {
			return nil, err
		}
		return resp, nil
	case <-time.After(100 * time.Millisecond):
		return nil, nil
	}
}

func TestNewServer(t *testing.T) {
	s := NewServer("test-server", "1.0.0")

	if s == nil {
		t.Fatal("Expected non-nil server")
	}
}

func TestServerInitialize(t *testing.T) {
	s := NewServer("test-server", "1.0.0")
	transport := newMockServerTransport()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.Serve(ctx, transport)
	}()

	time.Sleep(10 * time.Millisecond)

	transport.sendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})

	time.Sleep(10 * time.Millisecond)

	resp, err := transport.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if resp == nil {
		t.Fatal("Expected non-nil response")
	}

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if serverInfo, ok := result["serverInfo"].(map[string]interface{}); ok {
			if name, ok := serverInfo["name"].(string); ok && name != "test-server" {
				t.Errorf("Expected server name 'test-server', got '%s'", name)
			}
		}
	}

	cancel()
	time.Sleep(10 * time.Millisecond)
	<-done
}

func TestServerToolRegistration(t *testing.T) {
	s := NewServer("test-server", "1.0.0")

	tool := NewTool(
		"test_tool",
		"A test tool",
		map[string]interface{}{
			"type": "object",
		},
		func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					{Type: "text", Text: "ok"},
				},
			}, nil
		},
	)

	err := s.RegisterTool(tool)
	if err != nil {
		t.Fatalf("Failed to register tool: %v", err)
	}

	transport := newMockServerTransport()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.Serve(ctx, transport)
	}()

	time.Sleep(10 * time.Millisecond)

	transport.sendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})

	time.Sleep(10 * time.Millisecond)
	transport.readResponse()

	transport.sendRequest("initialized", nil)
	time.Sleep(10 * time.Millisecond)

	transport.sendRequest("tools/list", nil)
	time.Sleep(10 * time.Millisecond)

	resp, err := transport.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if tools, ok := result["tools"].([]interface{}); ok {
			if len(tools) != 1 {
				t.Fatalf("Expected 1 tool, got %d", len(tools))
			}
		}
	}

	cancel()
	time.Sleep(10 * time.Millisecond)
	<-done
}

func TestServerToolExecution(t *testing.T) {
	s := NewServer("test-server", "1.0.0")

	s.RegisterTool(NewTool(
		"echo_tool",
		"Echos the input",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"message": map[string]interface{}{
					"type": "string",
				},
			},
		},
		func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
			msg, _ := args["message"].(string)
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					{Type: "text", Text: msg},
				},
			}, nil
		},
	))

	transport := newMockServerTransport()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.Serve(ctx, transport)
	}()

	time.Sleep(10 * time.Millisecond)

	transport.sendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})

	time.Sleep(10 * time.Millisecond)
	transport.readResponse()

	transport.sendRequest("initialized", nil)
	time.Sleep(10 * time.Millisecond)

	transport.sendRequest("tools/call", map[string]interface{}{
		"name": "echo_tool",
		"arguments": map[string]interface{}{
			"message": "hello world",
		},
	})

	time.Sleep(10 * time.Millisecond)

	resp, err := transport.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if content, ok := result["content"].([]interface{}); ok && len(content) > 0 {
			if c, ok := content[0].(map[string]interface{}); ok {
				if text, ok := c["text"].(string); ok {
					if text != "hello world" {
						t.Errorf("Expected 'hello world', got '%s'", text)
					}
				}
			}
		}
	}

	cancel()
	time.Sleep(10 * time.Millisecond)
	<-done
}

func TestResourceRegistration(t *testing.T) {
	s := NewServer("test-server", "1.0.0")

	resource := NewTextResource(
		"test://resource",
		"Test Resource",
		"A test resource",
		"Hello, world!",
	)

	err := s.RegisterResource(resource)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}

	transport := newMockServerTransport()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.Serve(ctx, transport)
	}()

	time.Sleep(10 * time.Millisecond)

	transport.sendRequest("initialize", map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "test-client",
			"version": "1.0.0",
		},
	})

	time.Sleep(10 * time.Millisecond)
	transport.readResponse()

	transport.sendRequest("initialized", nil)
	time.Sleep(10 * time.Millisecond)

	transport.sendRequest("resources/list", nil)
	time.Sleep(10 * time.Millisecond)

	resp, err := transport.readResponse()
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	if result, ok := resp["result"].(map[string]interface{}); ok {
		if resources, ok := result["resources"].([]interface{}); ok {
			if len(resources) != 1 {
				t.Fatalf("Expected 1 resource, got %d", len(resources))
			}
		}
	}

	cancel()
	time.Sleep(10 * time.Millisecond)
	<-done
}
