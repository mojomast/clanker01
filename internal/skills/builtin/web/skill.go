package web

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/swarm-ai/swarm/internal/skills/loader"
)

type Skill struct {
	manifest *loader.SkillManifest
	client   *http.Client
}

func NewSkill() *Skill {
	manifest := &loader.SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: loader.SkillMetadata{
			Name:        "web",
			Version:     "1.0.0",
			DisplayName: "Web Operations",
			Description: "Fetch web content, make HTTP requests, and manage headers",
			Author:      "SWARM Team",
			License:     "Apache-2.0",
			Tags:        []string{"web", "http", "api"},
		},
		Spec: loader.SkillSpec{
			Runtime:    "native",
			Entrypoint: "builtin.web",
			Tools: []loader.ToolDef{
				{
					Name:        "fetch",
					Description: "Fetch content from a URL",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"url"},
						"properties": map[string]interface{}{
							"url": map[string]string{
								"type":        "string",
								"description": "URL to fetch",
							},
							"method": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
								"default":     "GET",
								"description": "HTTP method",
							},
							"headers": map[string]string{
								"type":        "object",
								"description": "Request headers",
							},
							"body": map[string]string{
								"type":        "string",
								"description": "Request body for POST/PUT/PATCH",
							},
							"timeout": map[string]interface{}{
								"type":        "integer",
								"default":     30,
								"description": "Request timeout in seconds",
							},
							"format": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"text", "json", "html"},
								"default":     "text",
								"description": "Response format",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"status":      map[string]string{"type": "integer"},
							"status_text": map[string]string{"type": "string"},
							"headers":     map[string]string{"type": "object"},
							"content":     map[string]string{"type": "string"},
							"size":        map[string]string{"type": "integer"},
						},
					},
				},
				{
					Name:        "request",
					Description: "Make an HTTP request with full control",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"url"},
						"properties": map[string]interface{}{
							"url": map[string]string{
								"type":        "string",
								"description": "URL to request",
							},
							"method": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
								"default":     "GET",
								"description": "HTTP method",
							},
							"headers": map[string]string{
								"type":        "object",
								"description": "Request headers",
							},
							"body": map[string]string{
								"type":        "string",
								"description": "Request body",
							},
							"query": map[string]string{
								"type":        "object",
								"description": "Query parameters",
							},
							"timeout": map[string]interface{}{
								"type":        "integer",
								"default":     30,
								"description": "Request timeout in seconds",
							},
							"follow_redirects": map[string]interface{}{
								"type":        "boolean",
								"default":     true,
								"description": "Follow HTTP redirects",
							},
							"verify_ssl": map[string]interface{}{
								"type":        "boolean",
								"default":     true,
								"description": "Verify SSL certificates",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"status":      map[string]string{"type": "integer"},
							"status_text": map[string]string{"type": "string"},
							"headers":     map[string]string{"type": "object"},
							"content":     map[string]string{"type": "string"},
							"size":        map[string]string{"type": "integer"},
							"elapsed":     map[string]string{"type": "number"},
						},
					},
				},
				{
					Name:        "download",
					Description: "Download a file from a URL",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"url", "path"},
						"properties": map[string]interface{}{
							"url": map[string]string{
								"type":        "string",
								"description": "URL to download from",
							},
							"path": map[string]string{
								"type":        "string",
								"description": "Local path to save the file",
							},
							"timeout": map[string]interface{}{
								"type":        "integer",
								"default":     60,
								"description": "Download timeout in seconds",
							},
							"headers": map[string]string{
								"type":        "object",
								"description": "Request headers",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"success": map[string]string{"type": "boolean"},
							"path":    map[string]string{"type": "string"},
							"size":    map[string]string{"type": "integer"},
						},
					},
				},
				{
					Name:        "headers",
					Description: "Parse and manipulate HTTP headers",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"action"},
						"properties": map[string]interface{}{
							"action": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"parse", "format", "validate"},
								"description": "Action to perform",
							},
							"headers": map[string]string{
								"type":        "object",
								"description": "Headers to process",
							},
							"raw": map[string]string{
								"type":        "string",
								"description": "Raw header string",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"result": map[string]string{"type": "object"},
							"valid":  map[string]string{"type": "boolean"},
							"errors": map[string]string{"type": "array"},
						},
					},
				},
			},
			Config: loader.SkillConfig{
				Schema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"default_timeout": map[string]interface{}{
							"type":    "integer",
							"default": 30,
						},
						"user_agent": map[string]string{
							"type":    "string",
							"default": "SWARM/1.0",
						},
						"max_redirects": map[string]interface{}{
							"type":    "integer",
							"default": 10,
						},
					},
				},
			},
			Permissions: loader.SkillPermissions{
				Filesystem: loader.FilesystemPermissions{
					Read:   []string{"**"},
					Write:  []string{"**"},
					Delete: []string{},
				},
				Network: loader.NetworkPermissions{
					Allow:        true,
					AllowedHosts: []string{"*"},
				},
			},
		},
	}

	return &Skill{
		manifest: manifest,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *Skill) Meta() *loader.SkillManifest {
	return s.manifest
}

func (s *Skill) Initialize(ctx context.Context, config *loader.Config) error {
	if config != nil && config.Settings != nil {
		if timeout, ok := config.Settings["default_timeout"].(float64); ok {
			s.client.Timeout = time.Duration(timeout) * time.Second
		}
	}
	return nil
}

func (s *Skill) Shutdown(ctx context.Context) error {
	s.client.CloseIdleConnections()
	return nil
}

func (s *Skill) Tools() []loader.Tool {
	var tools []loader.Tool
	for _, t := range s.manifest.Spec.Tools {
		tools = append(tools, loader.Tool{
			Type: "function",
			Function: loader.FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return tools
}

func (s *Skill) Execute(ctx context.Context, toolName string, args map[string]interface{}) (*loader.Result, error) {
	switch toolName {
	case "fetch":
		return s.fetch(ctx, args)
	case "request":
		return s.request(ctx, args)
	case "download":
		return s.download(ctx, args)
	case "headers":
		return s.headers(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (s *Skill) buildRequest(ctx context.Context, args map[string]interface{}) (*http.Request, error) {
	urlStr, ok := args["url"].(string)
	if !ok {
		return nil, fmt.Errorf("url is required")
	}

	method := "GET"
	if m, ok := args["method"].(string); ok && m != "" {
		method = m
	}

	var body io.Reader
	if b, ok := args["body"].(string); ok {
		body = strings.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}

	if headers, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	if query, ok := args["query"].(map[string]interface{}); ok {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, fmt.Sprintf("%v", v))
		}
		req.URL.RawQuery = q.Encode()
	}

	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "SWARM/1.0")
	}

	return req, nil
}

func (s *Skill) fetch(ctx context.Context, args map[string]interface{}) (*loader.Result, error) {
	req, err := s.buildRequest(ctx, args)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	timeout := 30
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	result := map[string]interface{}{
		"status":      resp.StatusCode,
		"status_text": resp.Status,
		"content":     string(content),
		"size":        len(content),
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}
	result["headers"] = headers

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) request(ctx context.Context, args map[string]interface{}) (*loader.Result, error) {
	req, err := s.buildRequest(ctx, args)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	timeout := 30
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	if follow, ok := args["follow_redirects"].(bool); ok {
		client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			if !follow {
				return http.ErrUseLastResponse
			}
			return nil
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	result := map[string]interface{}{
		"status":      resp.StatusCode,
		"status_text": resp.Status,
		"content":     string(content),
		"size":        len(content),
		"elapsed":     time.Since(start).Seconds(),
	}

	headers := make(map[string]string)
	for k, v := range resp.Header {
		headers[k] = strings.Join(v, ", ")
	}
	result["headers"] = headers

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) download(ctx context.Context, args map[string]interface{}) (*loader.Result, error) {
	urlStr, ok := args["url"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "url is required"}, nil
	}

	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	// Sanitize path to prevent path traversal attacks.
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		cwd, err := os.Getwd()
		if err != nil {
			return &loader.Result{Success: false, Error: "failed to determine working directory"}, nil
		}
		path = filepath.Join(cwd, path)
	}
	// Reject any path that still contains ".." components after cleaning.
	if strings.Contains(path, "..") {
		return &loader.Result{Success: false, Error: "path contains invalid traversal components"}, nil
	}
	// Ensure the resolved path is within the current working directory.
	cwd, err := os.Getwd()
	if err != nil {
		return &loader.Result{Success: false, Error: "failed to determine working directory"}, nil
	}
	cwdPrefix := filepath.Clean(cwd) + string(filepath.Separator)
	if !strings.HasPrefix(path, cwdPrefix) && path != filepath.Clean(cwd) {
		return &loader.Result{Success: false, Error: "path must be within the current working directory"}, nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	if headers, ok := args["headers"].(map[string]interface{}); ok {
		for k, v := range headers {
			req.Header.Set(k, fmt.Sprintf("%v", v))
		}
	}

	timeout := 60
	if t, ok := args["timeout"].(float64); ok {
		timeout = int(t)
	}

	client := &http.Client{Timeout: time.Duration(timeout) * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return &loader.Result{Success: false, Error: fmt.Sprintf("HTTP %d: %s", resp.StatusCode, resp.Status)}, nil
	}

	out, err := os.Create(path)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}
	defer out.Close()

	size, err := io.Copy(out, resp.Body)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	result := map[string]interface{}{
		"success": true,
		"path":    path,
		"size":    size,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) headers(args map[string]interface{}) (*loader.Result, error) {
	action, ok := args["action"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "action is required"}, nil
	}

	switch action {
	case "parse":
		return s.parseHeaders(args)
	case "format":
		return s.formatHeaders(args)
	case "validate":
		return s.validateHeaders(args)
	default:
		return &loader.Result{Success: false, Error: fmt.Sprintf("unknown action: %s", action)}, nil
	}
}

func (s *Skill) parseHeaders(args map[string]interface{}) (*loader.Result, error) {
	raw, ok := args["raw"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "raw is required for parse action"}, nil
	}

	lines := strings.Split(raw, "\n")
	headers := make(map[string]interface{})

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			headers[key] = value
		}
	}

	result := map[string]interface{}{
		"result": headers,
		"valid":  true,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) formatHeaders(args map[string]interface{}) (*loader.Result, error) {
	headers, ok := args["headers"].(map[string]interface{})
	if !ok {
		return &loader.Result{Success: false, Error: "headers is required for format action"}, nil
	}

	var lines []string
	for k, v := range headers {
		lines = append(lines, fmt.Sprintf("%s: %v", k, v))
	}

	result := map[string]interface{}{
		"result": strings.Join(lines, "\r\n"),
		"valid":  true,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) validateHeaders(args map[string]interface{}) (*loader.Result, error) {
	headers, ok := args["headers"].(map[string]interface{})
	if !ok {
		return &loader.Result{Success: false, Error: "headers is required for validate action"}, nil
	}

	var errors []string

	for k, v := range headers {
		if k == "" {
			errors = append(errors, "header name cannot be empty")
		}

		if strings.ContainsAny(k, " \t\r\n") {
			errors = append(errors, fmt.Sprintf("header name '%s' contains invalid characters", k))
		}

		if strVal, ok := v.(string); ok && strings.Contains(strVal, "\r\n") {
			errors = append(errors, fmt.Sprintf("header value for '%s' contains invalid newline characters", k))
		}

		_, err := url.Parse(fmt.Sprintf("%v", v))
		if err != nil {
			errors = append(errors, fmt.Sprintf("header value for '%s' is not valid: %v", k, err))
		}
	}

	result := map[string]interface{}{
		"valid":  len(errors) == 0,
		"errors": errors,
	}

	return &loader.Result{Success: true, Data: result}, nil
}
