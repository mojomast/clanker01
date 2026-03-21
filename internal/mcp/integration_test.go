package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/swarm-ai/swarm/internal/mcp"
	"github.com/swarm-ai/swarm/internal/mcp/client"
	"github.com/swarm-ai/swarm/internal/mcp/server"
	"github.com/swarm-ai/swarm/internal/mcp/servers"
)

type pipeTransport struct {
	clientToServer chan json.RawMessage
	serverToClient chan json.RawMessage
}

func newPipeTransport() *pipeTransport {
	return &pipeTransport{
		clientToServer: make(chan json.RawMessage, 100),
		serverToClient: make(chan json.RawMessage, 100),
	}
}

func (p *pipeTransport) Send(ctx context.Context, msg json.RawMessage) error {
	p.clientToServer <- msg
	return nil
}

func (p *pipeTransport) Receive(ctx context.Context) (json.RawMessage, error) {
	select {
	case msg := <-p.serverToClient:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (p *pipeTransport) Close() error {
	close(p.clientToServer)
	close(p.serverToClient)
	return nil
}

type serverPipeTransport struct {
	clientToServer chan json.RawMessage
	serverToClient chan json.RawMessage
}

func newServerPipeTransport(clientToServer, serverToClient chan json.RawMessage) *serverPipeTransport {
	return &serverPipeTransport{
		clientToServer: clientToServer,
		serverToClient: serverToClient,
	}
}

func (s *serverPipeTransport) Send(ctx context.Context, msg json.RawMessage) error {
	s.serverToClient <- msg
	return nil
}

func (s *serverPipeTransport) Receive(ctx context.Context) (json.RawMessage, error) {
	select {
	case msg := <-s.clientToServer:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (s *serverPipeTransport) Close() error {
	return nil
}

func TestClientServerIntegration(t *testing.T) {
	pipe := newPipeTransport()

	srv := server.NewServer("test-server", "1.0.0")
	srv.RegisterTool(server.NewTool(
		"add",
		"Add two numbers",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"a": map[string]interface{}{"type": "number"},
				"b": map[string]interface{}{"type": "number"},
			},
		},
		func(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)
			result := a + b
			return &mcp.CallToolResult{
				Content: []mcp.Content{
					{Type: "text", Text: fmt.Sprintf("%.0f", result)},
				},
			}, nil
		},
	))

	serverTransport := newServerPipeTransport(pipe.clientToServer, pipe.serverToClient)

	serverCtx, cancelServer := context.WithCancel(context.Background())
	defer cancelServer()

	go func() {
		if err := srv.Serve(serverCtx, serverTransport); err != nil {
			t.Logf("Server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	clt := client.NewClient(pipe)

	clientCtx := context.Background()

	initResult, err := clt.Initialize(clientCtx, &mcp.InitializeRequest{
		ProtocolVersion: mcp.ProtocolVersion,
		ClientInfo: mcp.Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	})

	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if initResult.ServerInfo.Name != "test-server" {
		t.Errorf("Expected server name 'test-server', got '%s'", initResult.ServerInfo.Name)
	}

	toolsResult, err := clt.ListTools(clientCtx, &mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(toolsResult.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(toolsResult.Tools))
	}

	callResult, err := clt.CallTool(clientCtx, &mcp.CallToolRequest{
		Name: "add",
		Arguments: map[string]interface{}{
			"a": 3.0,
			"b": 4.0,
		},
	})

	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}

	if len(callResult.Content) == 0 {
		t.Fatal("Expected non-empty content")
	}

	if callResult.Content[0].Text != "7" {
		t.Errorf("Expected '7', got '%s'", callResult.Content[0].Text)
	}

	time.Sleep(100 * time.Millisecond)
}

func TestFilesystemServer(t *testing.T) {
	fs := servers.NewFilesystemServer("/tmp", []string{"/tmp"})

	if fs == nil {
		t.Fatal("Expected non-nil filesystem server")
	}

	if fs.Server() == nil {
		t.Fatal("Expected non-nil server")
	}
}

func TestGitServer(t *testing.T) {
	git := servers.NewGitServer("/tmp")

	if git == nil {
		t.Fatal("Expected non-nil git server")
	}

	if git.Server() == nil {
		t.Fatal("Expected non-nil server")
	}
}

func TestMemoryServer(t *testing.T) {
	mem := servers.NewMemoryServer()

	if mem == nil {
		t.Fatal("Expected non-nil memory server")
	}

	if mem.Server() == nil {
		t.Fatal("Expected non-nil server")
	}
}

func TestHTTPServer(t *testing.T) {
	http := servers.NewHTTPServer()

	if http == nil {
		t.Fatal("Expected non-nil http server")
	}

	if http.Server() == nil {
		t.Fatal("Expected non-nil server")
	}
}
