package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/swarm-ai/swarm/internal/mcp"
)

type Client struct {
	transport Transport
	requestID int64
	mu        sync.Mutex
}

func NewClient(transport Transport) *Client {
	return &Client{
		transport: transport,
		requestID: 1,
	}
}

func (c *Client) Initialize(ctx context.Context, req *mcp.InitializeRequest) (*mcp.InitializeResult, error) {
	msg := c.buildRequest("initialize", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.InitializeResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal initialize result: %w", err)
	}

	return &result, nil
}

func (c *Client) ListPrompts(ctx context.Context, req *mcp.ListPromptsRequest) (*mcp.ListPromptsResult, error) {
	msg := c.buildRequest("prompts/list", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.ListPromptsResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list prompts result: %w", err)
	}

	return &result, nil
}

func (c *Client) GetPrompt(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
	msg := c.buildRequest("prompts/get", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.GetPromptResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal get prompt result: %w", err)
	}

	return &result, nil
}

func (c *Client) ListResources(ctx context.Context, req *mcp.ListResourcesRequest) (*mcp.ListResourcesResult, error) {
	msg := c.buildRequest("resources/list", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.ListResourcesResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list resources result: %w", err)
	}

	return &result, nil
}

func (c *Client) ReadResource(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
	msg := c.buildRequest("resources/read", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.ReadResourceResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal read resource result: %w", err)
	}

	return &result, nil
}

func (c *Client) ListTools(ctx context.Context, req *mcp.ListToolsRequest) (*mcp.ListToolsResult, error) {
	msg := c.buildRequest("tools/list", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.ListToolsResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal list tools result: %w", err)
	}

	return &result, nil
}

func (c *Client) CallTool(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	msg := c.buildRequest("tools/call", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.CallToolResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal call tool result: %w", err)
	}

	return &result, nil
}

func (c *Client) SetLevel(ctx context.Context, req *mcp.SetLevelRequest) error {
	msg := c.buildRequest("logging/setLevel", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return err
	}

	if respMsg.Error != nil {
		return fmt.Errorf("set level failed: %s", respMsg.Error.Message)
	}

	return nil
}

func (c *Client) Complete(ctx context.Context, req *mcp.CompleteRequest) (*mcp.CompleteResult, error) {
	msg := c.buildRequest("completion/complete", req)

	respMsg, err := c.send(ctx, msg)
	if err != nil {
		return nil, err
	}

	var result mcp.CompleteResult
	if err := json.Unmarshal(respMsg.Result, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal complete result: %w", err)
	}

	return &result, nil
}

func (c *Client) buildRequest(method string, params interface{}) json.RawMessage {
	c.mu.Lock()
	id := c.requestID
	c.requestID++
	c.mu.Unlock()

	req := mcp.JSONRPCMessage{
		JSONRPC: "2.0",
		ID:      json.RawMessage(fmt.Sprintf("%d", id)),
		Method:  method,
	}

	if params != nil {
		paramsBytes, err := json.Marshal(params)
		if err == nil {
			req.Params = json.RawMessage(paramsBytes)
		}
	}

	msgBytes, _ := json.Marshal(req)
	return json.RawMessage(msgBytes)
}

func (c *Client) send(ctx context.Context, msg json.RawMessage) (*mcp.JSONRPCMessage, error) {
	if err := c.transport.Send(ctx, msg); err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	respMsgBytes, err := c.transport.Receive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to receive response: %w", err)
	}

	var respMsg mcp.JSONRPCMessage
	if err := json.Unmarshal(respMsgBytes, &respMsg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if respMsg.Error != nil {
		return &respMsg, fmt.Errorf("server error: %d - %s", respMsg.Error.Code, respMsg.Error.Message)
	}

	return &respMsg, nil
}

func (c *Client) SendNotification(ctx context.Context, method string, params interface{}) error {
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

	if err := c.transport.Send(ctx, json.RawMessage(msgBytes)); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}

	return nil
}
