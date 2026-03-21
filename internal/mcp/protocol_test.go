package mcp

import (
	"testing"
)

func TestProtocolVersion(t *testing.T) {
	if ProtocolVersion != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %s", ProtocolVersion)
	}
}

func TestErrorCodes(t *testing.T) {
	codes := map[int]string{
		ParseError:       "-32700",
		InvalidRequest:   "-32600",
		MethodNotFound:   "-32601",
		InvalidParams:    "-32602",
		InternalError:    "-32603",
		Unauthorized:     "-32001",
		Forbidden:        "-32002",
		RateLimited:      "-32003",
		Timeout:          "-32004",
		ResourceNotFound: "-32005",
		ToolError:        "-32006",
	}

	for code, expected := range codes {
		if expected == "" {
			t.Errorf("Error code %d should have a description", code)
		}
	}
}

func TestJSONRPCMessage(t *testing.T) {
	msg := JSONRPCMessage{
		JSONRPC: "2.0",
		Method:  "test.method",
	}

	if msg.JSONRPC != "2.0" {
		t.Errorf("Expected JSON-RPC version 2.0, got %s", msg.JSONRPC)
	}

	if msg.Method != "test.method" {
		t.Errorf("Expected method test.method, got %s", msg.Method)
	}
}

func TestInitializeRequest(t *testing.T) {
	req := InitializeRequest{
		ProtocolVersion: ProtocolVersion,
		ClientInfo: Implementation{
			Name:    "test-client",
			Version: "1.0.0",
		},
	}

	if req.ProtocolVersion != ProtocolVersion {
		t.Errorf("Expected protocol version %s, got %s", ProtocolVersion, req.ProtocolVersion)
	}

	if req.ClientInfo.Name != "test-client" {
		t.Errorf("Expected client name test-client, got %s", req.ClientInfo.Name)
	}
}

func TestInitializeResult(t *testing.T) {
	result := InitializeResult{
		ProtocolVersion: ProtocolVersion,
		ServerInfo: Implementation{
			Name:    "test-server",
			Version: "1.0.0",
		},
	}

	if result.ProtocolVersion != ProtocolVersion {
		t.Errorf("Expected protocol version %s, got %s", ProtocolVersion, result.ProtocolVersion)
	}

	if result.ServerInfo.Name != "test-server" {
		t.Errorf("Expected server name test-server, got %s", result.ServerInfo.Name)
	}
}
