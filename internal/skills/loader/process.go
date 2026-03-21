package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// ProcessSkill wraps a skill process that communicates via JSON-RPC over stdio
type ProcessSkill struct {
	manifest *SkillManifest
	cmd      string
	args     []string
	process  *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	mu       sync.Mutex
	sandbox  *Sandbox
	config   *Config
}

// NewProcessSkill creates a new process-based skill
func NewProcessSkill(manifest *SkillManifest, cmd string, args []string, sandbox *Sandbox) *ProcessSkill {
	return &ProcessSkill{
		manifest: manifest,
		cmd:      cmd,
		args:     args,
		sandbox:  sandbox,
	}
}

// Meta returns the skill manifest
func (s *ProcessSkill) Meta() *SkillManifest {
	return s.manifest
}

// Initialize starts the process and sends the initialize request
func (s *ProcessSkill) Initialize(ctx context.Context, config *Config) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config

	cmd := exec.CommandContext(ctx, s.cmd, s.args...)
	if config != nil && config.Workspace != "" {
		cmd.Dir = config.Workspace
	}

	var err error
	s.process = cmd

	// Use sandbox if enabled
	if s.sandbox != nil && s.sandbox.config != nil && s.sandbox.config.Enabled {
		_, s.stdin, s.stdout, err = s.sandbox.StartProcess(ctx, cmd)
		if err != nil {
			return fmt.Errorf("start process in sandbox: %w", err)
		}
	} else {
		s.stdin, err = cmd.StdinPipe()
		if err != nil {
			return err
		}

		s.stdout, err = cmd.StdoutPipe()
		if err != nil {
			s.stdin.Close()
			return err
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			s.stdin.Close()
			s.stdout.Close()
			return err
		}

		if err := cmd.Start(); err != nil {
			s.stdin.Close()
			s.stdout.Close()
			stderr.Close()
			return err
		}

		// Drain stderr to avoid blocking
		go func() {
			io.Copy(io.Discard, stderr)
			stderr.Close()
		}()
	}

	// Send initialize request
	initParams := make(map[string]interface{})
	if config != nil {
		initParams["workspace"] = config.Workspace
		initParams["config"] = config.Settings
	}

	_, err = s.sendRequest("initialize", initParams)
	if err != nil {
		s.Shutdown(ctx)
		return fmt.Errorf("initialize skill: %w", err)
	}

	return nil
}

// Shutdown stops the process
func (s *ProcessSkill) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Send shutdown request
	s.sendRequest("shutdown", nil)

	// Close pipes
	if s.stdin != nil {
		s.stdin.Close()
		s.stdin = nil
	}
	if s.stdout != nil {
		s.stdout.Close()
		s.stdout = nil
	}

	// Kill process
	if s.process != nil && s.process.Process != nil {
		pid := s.process.Process.Pid
		if s.sandbox != nil {
			s.sandbox.StopProcess(pid)
		} else {
			s.process.Process.Signal(syscall.SIGTERM)

			// Kill after 5 seconds
			time.AfterFunc(5*time.Second, func() {
				s.process.Process.Kill()
			})
		}
		s.process = nil
	}

	return nil
}

// Tools returns the list of tools from the manifest
func (s *ProcessSkill) Tools() []Tool {
	var tools []Tool
	for _, t := range s.manifest.Spec.Tools {
		tools = append(tools, Tool{
			Type: "function",
			Function: FunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return tools
}

// Execute executes a tool
func (s *ProcessSkill) Execute(
	ctx context.Context,
	toolName string,
	args map[string]interface{},
) (*Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	start := time.Now()

	resp, err := s.sendRequest("execute", map[string]interface{}{
		"tool": toolName,
		"args": args,
	})
	if err != nil {
		return &Result{Success: false, Error: err.Error()}, nil
	}

	return &Result{
		Success:  resp.GetBool("success"),
		Data:     resp.Get("data"),
		Error:    resp.GetString("error"),
		Duration: time.Since(start),
	}, nil
}

// sendRequest sends a JSON-RPC request and waits for response
func (s *ProcessSkill) sendRequest(method string, params interface{}) (*RPCResponse, error) {
	if s.stdin == nil || s.stdout == nil {
		return nil, fmt.Errorf("process not running")
	}

	req := &RPCRequest{
		JSONRPC: "2.0",
		ID:      generateID(),
		Method:  method,
		Params:  params,
	}

	encoder := json.NewEncoder(s.stdin)
	if err := encoder.Encode(req); err != nil {
		return nil, fmt.Errorf("encode request: %w", err)
	}

	var resp RPCResponse
	decoder := json.NewDecoder(s.stdout)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("%d: %s", resp.Error.Code, resp.Error.Message)
	}

	return &resp, nil
}

// generateID generates a unique JSON-RPC request ID
func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
