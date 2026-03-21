package git

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/swarm-ai/swarm/internal/skills/loader"
)

type Skill struct {
	manifest *loader.SkillManifest
}

func NewSkill() *Skill {
	manifest := &loader.SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: loader.SkillMetadata{
			Name:        "git",
			Version:     "1.0.0",
			DisplayName: "Git Operations",
			Description: "Clone repositories, check status, commit changes, and manage branches",
			Author:      "SWARM Team",
			License:     "Apache-2.0",
			Tags:        []string{"git", "version-control", "vcs"},
		},
		Spec: loader.SkillSpec{
			Runtime:    "native",
			Entrypoint: "builtin.git",
			Tools: []loader.ToolDef{
				{
					Name:        "clone",
					Description: "Clone a git repository",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"url"},
						"properties": map[string]interface{}{
							"url": map[string]string{
								"type":        "string",
								"description": "Repository URL to clone",
							},
							"path": map[string]string{
								"type":        "string",
								"description": "Destination directory",
							},
							"branch": map[string]string{
								"type":        "string",
								"description": "Specific branch to checkout",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"success": map[string]string{"type": "boolean"},
							"path":    map[string]string{"type": "string"},
						},
					},
				},
				{
					Name:        "status",
					Description: "Get git repository status",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the repository",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"branch":   map[string]string{"type": "string"},
							"modified": map[string]string{"type": "array"},
							"added":    map[string]string{"type": "array"},
							"deleted":  map[string]string{"type": "array"},
						},
					},
				},
				{
					Name:        "commit",
					Description: "Commit changes with a message",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path", "message"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the repository",
							},
							"message": map[string]string{
								"type":        "string",
								"description": "Commit message",
							},
							"files": map[string]interface{}{
								"type":        "array",
								"items":       map[string]string{"type": "string"},
								"description": "Specific files to commit default all",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"success": map[string]string{"type": "boolean"},
							"hash":    map[string]string{"type": "string"},
						},
					},
				},
				{
					Name:        "branch",
					Description: "Create, list, or delete branches",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the repository",
							},
							"name": map[string]string{
								"type":        "string",
								"description": "Branch name (for create/delete)",
							},
							"action": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"list", "create", "delete", "checkout"},
								"default":     "list",
								"description": "Action to perform",
							},
							"checkout": map[string]interface{}{
								"type":        "boolean",
								"default":     false,
								"description": "Checkout the branch after creating (for create action)",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"success":  map[string]string{"type": "boolean"},
							"branches": map[string]string{"type": "array"},
							"current":  map[string]string{"type": "string"},
						},
					},
				},
				{
					Name:        "log",
					Description: "View commit history",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the repository",
							},
							"limit": map[string]interface{}{
								"type":        "integer",
								"default":     10,
								"description": "Maximum number of commits to show",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"commits": map[string]string{"type": "array"},
						},
					},
				},
				{
					Name:        "diff",
					Description: "Show changes between commits or working directory",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the repository",
							},
							"cached": map[string]interface{}{
								"type":        "boolean",
								"default":     false,
								"description": "Show staged changes only",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"diff": map[string]string{"type": "string"},
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
	return &Skill{manifest: manifest}
}

func (s *Skill) Meta() *loader.SkillManifest {
	return s.manifest
}

func (s *Skill) Initialize(ctx context.Context, config *loader.Config) error {
	return nil
}

func (s *Skill) Shutdown(ctx context.Context) error {
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
	case "clone":
		return s.clone(args)
	case "status":
		return s.status(args)
	case "commit":
		return s.commit(args)
	case "branch":
		return s.branch(args)
	case "log":
		return s.log(args)
	case "diff":
		return s.diff(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (s *Skill) gitCommand(args map[string]interface{}, gitArgs ...string) (string, error) {
	path, ok := args["path"].(string)
	if !ok {
		return "", fmt.Errorf("path is required")
	}

	cmd := exec.Command("git", gitArgs...)
	cmd.Dir = path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w", string(output), err)
	}

	return string(output), nil
}

func (s *Skill) clone(args map[string]interface{}) (*loader.Result, error) {
	url, ok := args["url"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "url is required"}, nil
	}

	path := filepath.Base(url)
	if p, ok := args["path"].(string); ok && p != "" {
		path = p
	}

	gitArgs := []string{"clone", url, path}
	if branch, ok := args["branch"].(string); ok && branch != "" {
		gitArgs = []string{"clone", "-b", branch, url, path}
	}

	cmd := exec.Command("git", gitArgs...)
	if err := cmd.Run(); err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	absPath, _ := filepath.Abs(path)

	result := map[string]interface{}{
		"success": true,
		"path":    absPath,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) status(args map[string]interface{}) (*loader.Result, error) {
	output, err := s.gitCommand(args, "status", "--porcelain")
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	branchOutput, _ := s.gitCommand(args, "rev-parse", "--abbrev-ref", "HEAD")
	branch := strings.TrimSpace(branchOutput)

	var modified, added, deleted []string
	for _, line := range strings.Split(output, "\n") {
		if line == "" {
			continue
		}
		status := line[:2]
		file := line[3:]

		switch status {
		case " M", "MM":
			modified = append(modified, file)
		case "??":
			added = append(added, file)
		case "D ", "MD":
			deleted = append(deleted, file)
		}
	}

	result := map[string]interface{}{
		"branch":   branch,
		"modified": modified,
		"added":    added,
		"deleted":  deleted,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) commit(args map[string]interface{}) (*loader.Result, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	message, ok := args["message"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "message is required"}, nil
	}

	var cmd *exec.Cmd
	if files, ok := args["files"].([]interface{}); ok && len(files) > 0 {
		filePaths := make([]string, len(files))
		for i, f := range files {
			filePaths[i] = f.(string)
		}
		cmd = exec.Command("git", append([]string{"commit", "-m", message}, filePaths...)...)
	} else {
		cmd = exec.Command("git", "commit", "-am", message)
	}
	cmd.Dir = path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &loader.Result{Success: false, Error: string(output)}, nil
	}

	hashCmd := exec.Command("git", "rev-parse", "HEAD")
	hashCmd.Dir = path
	hashOutput, _ := hashCmd.CombinedOutput()
	hash := strings.TrimSpace(string(hashOutput))

	result := map[string]interface{}{
		"success": true,
		"hash":    hash,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) branch(args map[string]interface{}) (*loader.Result, error) {
	if _, ok := args["path"].(string); !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	action := "list"
	if a, ok := args["action"].(string); ok {
		action = a
	}

	switch action {
	case "list":
		return s.listBranches(args)
	case "create":
		return s.createBranch(args)
	case "delete":
		return s.deleteBranch(args)
	case "checkout":
		return s.checkoutBranch(args)
	default:
		return &loader.Result{Success: false, Error: fmt.Sprintf("unknown action: %s", action)}, nil
	}
}

func (s *Skill) listBranches(args map[string]interface{}) (*loader.Result, error) {
	output, err := s.gitCommand(args, "branch", "--format=%(refname:short)")
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	branches := strings.Split(strings.TrimSpace(output), "\n")
	if output == "" {
		branches = []string{}
	}

	currentOutput, _ := s.gitCommand(args, "rev-parse", "--abbrev-ref", "HEAD")
	current := strings.TrimSpace(currentOutput)

	result := map[string]interface{}{
		"success":  true,
		"branches": branches,
		"current":  current,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) createBranch(args map[string]interface{}) (*loader.Result, error) {
	name, ok := args["name"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "name is required"}, nil
	}

	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	cmd := exec.Command("git", "branch", name)
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	if checkout, ok := args["checkout"].(bool); ok && checkout {
		cmd = exec.Command("git", "checkout", name)
		cmd.Dir = path
		if err := cmd.Run(); err != nil {
			return &loader.Result{Success: false, Error: err.Error()}, nil
		}
	}

	result := map[string]interface{}{
		"success": true,
		"name":    name,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) deleteBranch(args map[string]interface{}) (*loader.Result, error) {
	name, ok := args["name"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "name is required"}, nil
	}

	_, err := s.gitCommand(args, "branch", "-D", name)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	result := map[string]interface{}{
		"success": true,
		"name":    name,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) checkoutBranch(args map[string]interface{}) (*loader.Result, error) {
	name, ok := args["name"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "name is required"}, nil
	}

	_, err := s.gitCommand(args, "checkout", name)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	result := map[string]interface{}{
		"success": true,
		"name":    name,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) log(args map[string]interface{}) (*loader.Result, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	cmd := exec.Command("git", "log", "--format=%H|%an|%ae|%s", "-n", fmt.Sprintf("%d", limit))
	cmd.Dir = path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &loader.Result{Success: false, Error: string(output)}, nil
	}

	var commits []map[string]string
	for _, line := range strings.Split(string(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) == 4 {
			commits = append(commits, map[string]string{
				"hash":    parts[0],
				"author":  parts[1],
				"email":   parts[2],
				"message": parts[3],
			})
		}
	}

	result := map[string]interface{}{
		"commits": commits,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) diff(args map[string]interface{}) (*loader.Result, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	cached := false
	if c, ok := args["cached"].(bool); ok {
		cached = c
	}

	var cmd *exec.Cmd
	if cached {
		cmd = exec.Command("git", "diff", "--cached")
	} else {
		cmd = exec.Command("git", "diff")
	}
	cmd.Dir = path

	output, err := cmd.CombinedOutput()
	if err != nil {
		return &loader.Result{Success: false, Error: string(output)}, nil
	}

	result := map[string]interface{}{
		"diff": string(output),
	}

	return &loader.Result{Success: true, Data: result}, nil
}
