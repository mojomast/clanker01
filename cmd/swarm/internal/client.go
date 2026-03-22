package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is an HTTP client for the SWARM REST API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewClient creates a new SWARM REST API client.
// The baseURL must use http:// or https:// scheme.
func NewClient(baseURL, token string) (*Client, error) {
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		return nil, fmt.Errorf("invalid base URL %q: must start with http:// or https://", baseURL)
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

// sanitizePathParam escapes a user-provided string for safe inclusion
// in a URL path segment, preventing path traversal attacks.
func sanitizePathParam(s string) string {
	return url.PathEscape(s)
}

// --- Client-side response types (mirrors server-side JSON shapes) ---

// APIAgentResponse is the client-side representation of a single agent
// returned by the REST API.
type APIAgentResponse struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Name      string                 `json:"name"`
	Status    string                 `json:"status"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	Config    map[string]interface{} `json:"config"`
	Metrics   map[string]interface{} `json:"metrics"`
}

// APIAgentListResponse is the wrapper for the agent list endpoint.
type APIAgentListResponse struct {
	Count  int                `json:"count"`
	Agents []APIAgentResponse `json:"agents"`
}

// APICreateAgentRequest is the request body for creating an agent.
type APICreateAgentRequest struct {
	Type   string                 `json:"type"`
	Name   string                 `json:"name"`
	Model  string                 `json:"model,omitempty"`
	Skills []string               `json:"skills,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
}

// APISkillToolInfo represents a tool provided by a skill.
type APISkillToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Returns     map[string]interface{} `json:"returns"`
}

// APISkillResponse is the client-side representation of a skill
// returned by the REST API.
type APISkillResponse struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Status      string                 `json:"status"`
	LoadedAt    time.Time              `json:"loaded_at"`
	Config      map[string]interface{} `json:"config"`
	Tools       []APISkillToolInfo     `json:"tools"`
}

// APISkillListResponse is the wrapper for the skill list endpoint.
type APISkillListResponse struct {
	Count  int                `json:"count"`
	Skills []APISkillResponse `json:"skills"`
}

// APIInstallSkillRequest is the request body for installing/loading a skill.
type APIInstallSkillRequest struct {
	Name    string                 `json:"name"`
	Version string                 `json:"version,omitempty"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enable  bool                   `json:"enable"`
}

// APIErrorResponse represents an error returned by the API.
type APIErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// --- Internal HTTP helpers ---

// doRequest performs an HTTP request against the SWARM API, adding
// base URL prefix and authentication headers.
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	return resp, nil
}

// doJSON performs an HTTP request and decodes the JSON response into result.
// If result is nil, only the status code is checked.
func (c *Client) doJSON(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	resp, err := c.doRequest(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// For 204 No Content, there's nothing to decode.
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	// Read the full body for error reporting or decoding (capped at 10 MB).
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		var apiErr APIErrorResponse
		if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
			return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, apiErr.Error)
		}
		return fmt.Errorf("API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// --- Health ---

// Ping checks connectivity to the SWARM server by hitting the health endpoint.
func (c *Client) Ping(ctx context.Context) error {
	var health map[string]interface{}
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/health", nil, &health); err != nil {
		return err
	}
	status, _ := health["status"].(string)
	if status != "healthy" {
		return fmt.Errorf("server reported unhealthy status: %s", status)
	}
	return nil
}

// --- Agent methods ---

// ListAgents retrieves all agents from the SWARM server.
func (c *Client) ListAgents(ctx context.Context) ([]APIAgentResponse, error) {
	var resp APIAgentListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/agents", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Agents, nil
}

// CreateAgent creates a new agent on the SWARM server.
func (c *Client) CreateAgent(ctx context.Context, req *APICreateAgentRequest) (*APIAgentResponse, error) {
	var resp APIAgentResponse
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/agents", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAgent retrieves a single agent by ID.
func (c *Client) GetAgent(ctx context.Context, id string) (*APIAgentResponse, error) {
	var resp APIAgentResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/agents/"+sanitizePathParam(id), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// DeleteAgent removes an agent by ID.
func (c *Client) DeleteAgent(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/agents/"+sanitizePathParam(id), nil, nil)
}

// StartAgent starts an agent by ID.
func (c *Client) StartAgent(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/agents/"+sanitizePathParam(id)+"/start", nil, nil)
}

// StopAgent stops an agent by ID.
func (c *Client) StopAgent(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/agents/"+sanitizePathParam(id)+"/stop", nil, nil)
}

// --- Skill methods ---

// ListSkills retrieves all skills from the SWARM server.
func (c *Client) ListSkills(ctx context.Context) ([]APISkillResponse, error) {
	var resp APISkillListResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/skills", nil, &resp); err != nil {
		return nil, err
	}
	return resp.Skills, nil
}

// GetSkill retrieves a single skill by name/ID.
func (c *Client) GetSkill(ctx context.Context, name string) (*APISkillResponse, error) {
	var resp APISkillResponse
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/skills/"+sanitizePathParam(name), nil, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// InstallSkill installs/loads a skill on the SWARM server.
func (c *Client) InstallSkill(ctx context.Context, req *APIInstallSkillRequest) error {
	return c.doJSON(ctx, http.MethodPost, "/api/v1/skills", req, nil)
}

// RemoveSkill unloads/removes a skill by name/ID.
func (c *Client) RemoveSkill(ctx context.Context, name string) error {
	return c.doJSON(ctx, http.MethodDelete, "/api/v1/skills/"+sanitizePathParam(name), nil, nil)
}
