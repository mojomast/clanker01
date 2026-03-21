package servers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/swarm-ai/swarm/internal/mcp"
	"github.com/swarm-ai/swarm/internal/mcp/server"
)

type GitServer struct {
	mcpServer *server.Server
	repoPath  string
}

func NewGitServer(repoPath string) *GitServer {
	s := &GitServer{repoPath: repoPath}
	s.mcpServer = server.NewServer("git", "1.0.0")
	s.registerTools()
	return s
}

func (s *GitServer) registerTools() {
	s.mcpServer.RegisterTool(server.NewTool(
		"git_status",
		"Show working tree status",
		map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
		s.gitStatus,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"git_log",
		"Show commit logs",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"n": map[string]interface{}{
					"type":        "number",
					"description": "Number of commits to show",
				},
				"oneline": map[string]interface{}{
					"type":        "boolean",
					"description": "Show one line per commit",
				},
			},
		},
		s.gitLog,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"git_diff",
		"Show changes between commits",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"cached": map[string]interface{}{
					"type":        "boolean",
					"description": "Show staged changes",
				},
				"file": map[string]interface{}{
					"type":        "string",
					"description": "Specific file to diff",
				},
			},
		},
		s.gitDiff,
	))

	s.mcpServer.RegisterTool(server.NewTool(
		"git_branch",
		"List branches",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"all": map[string]interface{}{
					"type":        "boolean",
					"description": "List all branches including remote",
				},
			},
		},
		s.gitBranch,
	))
}

func (s *GitServer) runGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = s.repoPath
	output, err := cmd.CombinedOutput()
	return string(output), err
}

func (s *GitServer) gitStatus(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	output, err := s.runGit("status", "--porcelain")
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: output}},
	}, nil
}

func (s *GitServer) gitLog(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	cmdArgs := []string{"log"}

	if n, ok := args["n"].(float64); ok {
		cmdArgs = append(cmdArgs, fmt.Sprintf("-n%d", int(n)))
	}

	if oneline, ok := args["oneline"].(bool); ok && oneline {
		cmdArgs = append(cmdArgs, "--oneline")
	}

	output, err := s.runGit(cmdArgs...)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: output}},
	}, nil
}

func (s *GitServer) gitDiff(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	cmdArgs := []string{"diff"}

	if cached, ok := args["cached"].(bool); ok && cached {
		cmdArgs = append(cmdArgs, "--cached")
	}

	if file, ok := args["file"].(string); ok {
		cmdArgs = append(cmdArgs, "--", file)
	}

	output, err := s.runGit(cmdArgs...)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{{Type: "text", Text: output}},
	}, nil
}

func (s *GitServer) gitBranch(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	cmdArgs := []string{"branch"}

	if all, ok := args["all"].(bool); ok && all {
		cmdArgs = append(cmdArgs, "-a")
	}

	output, err := s.runGit(cmdArgs...)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	branches := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			branches = append(branches, trimmed)
		}
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			{Type: "text", Text: strings.Join(branches, "\n")},
		},
	}, nil
}

func (s *GitServer) Serve(ctx context.Context, transport server.Transport) error {
	return s.mcpServer.Serve(ctx, transport)
}

func (s *GitServer) Server() *server.Server {
	return s.mcpServer
}
