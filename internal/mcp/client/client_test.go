package client

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/swarm-ai/swarm/internal/mcp"
)

type mockTransport struct {
	sendQueue    chan json.RawMessage
	receiveQueue chan json.RawMessage
	closed       bool
}

func newMockTransport() *mockTransport {
	return &mockTransport{
		sendQueue:    make(chan json.RawMessage, 100),
		receiveQueue: make(chan json.RawMessage, 100),
	}
}

func (m *mockTransport) Send(ctx context.Context, msg json.RawMessage) error {
	if m.closed {
		return nil
	}
	m.sendQueue <- msg
	return nil
}

func (m *mockTransport) Receive(ctx context.Context) (json.RawMessage, error) {
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

func (m *mockTransport) Close() error {
	m.closed = true
	close(m.sendQueue)
	close(m.receiveQueue)
	return nil
}

func (m *mockTransport) sendResponse(result interface{}) error {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"result":  result,
	}
	data, _ := json.Marshal(resp)
	m.receiveQueue <- json.RawMessage(data)
	return nil
}

func (m *mockTransport) sendErrorResponse(code int, message string) error {
	resp := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}
	data, _ := json.Marshal(resp)
	m.receiveQueue <- json.RawMessage(data)
	return nil
}

func TestNewClient(t *testing.T) {
	transport := newMockTransport()
	client := NewClient(transport)

	if client == nil {
		t.Fatal("Expected non-nil client")
	}

	if client.transport != transport {
		t.Error("Expected transport to be set")
	}
}

func TestClientInitialize(t *testing.T) {
	transport := newMockTransport()
	client := NewClient(transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		transport.sendResponse(map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "test-server",
				"version": "1.0.0",
			},
		})
	}()

	result, err := client.Initialize(ctx, &mcp.InitializeRequest{
		ProtocolVersion: "2024-11-05",
		ClientInfo: mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if result.ServerInfo.Name != "test-server" {
		t.Errorf("Expected server name test-server, got %s", result.ServerInfo.Name)
	}
}

func TestClientListTools(t *testing.T) {
	transport := newMockTransport()
	client := NewClient(transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		transport.sendResponse(map[string]interface{}{
			"tools": []map[string]interface{}{
				{
					"name":        "test_tool",
					"description": "A test tool",
					"inputSchema": map[string]interface{}{
						"type": "object",
					},
				},
			},
		})
	}()

	result, err := client.ListTools(ctx, &mcp.ListToolsRequest{})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(result.Tools))
	}

	if result.Tools[0].Name != "test_tool" {
		t.Errorf("Expected tool name test_tool, got %s", result.Tools[0].Name)
	}
}

func TestClientCallTool(t *testing.T) {
	transport := newMockTransport()
	client := NewClient(transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		transport.sendResponse(map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": "Tool executed successfully",
				},
			},
			"isError": false,
		})
	}()

	result, err := client.CallTool(ctx, &mcp.CallToolRequest{
		Name: "test_tool",
		Arguments: map[string]interface{}{
			"param1": "value1",
		},
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(result.Content) != 1 {
		t.Fatalf("Expected 1 content item, got %d", len(result.Content))
	}

	if result.Content[0].Text != "Tool executed successfully" {
		t.Errorf("Expected 'Tool executed successfully', got %s", result.Content[0].Text)
	}
}

func TestClientError(t *testing.T) {
	transport := newMockTransport()
	client := NewClient(transport)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	go func() {
		time.Sleep(10 * time.Millisecond)
		transport.sendErrorResponse(-32601, "Method not found")
	}()

	_, err := client.ListTools(ctx, &mcp.ListToolsRequest{})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	expectedErrMsg := "server error: -32601 - Method not found"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrMsg, err.Error())
	}
}

func TestSendNotification(t *testing.T) {
	transport := newMockTransport()
	client := NewClient(transport)

	ctx := context.Background()

	err := client.SendNotification(ctx, "test/notification", map[string]interface{}{
		"key": "value",
	})

	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	select {
	case msg := <-transport.sendQueue:
		var parsed map[string]interface{}
		if err := json.Unmarshal(msg, &parsed); err != nil {
			t.Fatalf("Failed to parse message: %v", err)
		}

		if parsed["method"] != "test/notification" {
			t.Errorf("Expected method 'test/notification', got %v", parsed["method"])
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Timeout waiting for message to be sent")
	}
}
