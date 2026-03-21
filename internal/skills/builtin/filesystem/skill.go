package filesystem

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
			Name:        "filesystem",
			Version:     "1.0.0",
			DisplayName: "Filesystem Operations",
			Description: "Read, write, list, and search files and directories",
			Author:      "SWARM Team",
			License:     "Apache-2.0",
			Tags:        []string{"filesystem", "io", "files"},
		},
		Spec: loader.SkillSpec{
			Runtime:    "native",
			Entrypoint: "builtin.filesystem",
			Tools: []loader.ToolDef{
				{
					Name:        "read_file",
					Description: "Read the contents of a file",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the file",
							},
							"offset": map[string]interface{}{
								"type":        "integer",
								"description": "Line number to start reading from (1-indexed)",
								"default":     0,
							},
							"limit": map[string]interface{}{
								"type":        "integer",
								"description": "Maximum number of lines to read",
								"default":     0,
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"content": map[string]string{"type": "string"},
							"lines":   map[string]string{"type": "integer"},
							"path":    map[string]string{"type": "string"},
						},
					},
				},
				{
					Name:        "write_file",
					Description: "Write content to a file",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path", "content"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the file",
							},
							"content": map[string]string{
								"type":        "string",
								"description": "Content to write",
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
					Name:        "list_directory",
					Description: "List contents of a directory",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"path"},
						"properties": map[string]interface{}{
							"path": map[string]string{
								"type":        "string",
								"description": "Path to the directory",
							},
							"recursive": map[string]interface{}{
								"type":        "boolean",
								"default":     false,
								"description": "List recursively",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"files":       map[string]string{"type": "array"},
							"directories": map[string]string{"type": "array"},
						},
					},
				},
				{
					Name:        "search_files",
					Description: "Search for files matching a pattern",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"pattern"},
						"properties": map[string]interface{}{
							"pattern": map[string]string{
								"type":        "string",
								"description": "Glob pattern to match",
							},
							"path": map[string]string{
								"type":        "string",
								"description": "Directory to search in",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"matches": map[string]string{"type": "array"},
						},
					},
				},
				{
					Name:        "search_content",
					Description: "Search for content within files",
					Parameters: map[string]interface{}{
						"type":     "object",
						"required": []string{"pattern"},
						"properties": map[string]interface{}{
							"pattern": map[string]string{
								"type":        "string",
								"description": "Regex pattern to search for",
							},
							"path": map[string]string{
								"type":        "string",
								"description": "Directory to search in",
							},
							"include": map[string]string{
								"type":        "string",
								"description": "File pattern to include (e.g., *.go)",
							},
						},
					},
					Returns: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"matches": map[string]string{"type": "array"},
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
					Allow: false,
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
	case "read_file":
		return s.readFile(args)
	case "write_file":
		return s.writeFile(args)
	case "list_directory":
		return s.listDirectory(args)
	case "search_files":
		return s.searchFiles(args)
	case "search_content":
		return s.searchContent(args)
	default:
		return nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}

func (s *Skill) readFile(args map[string]interface{}) (*loader.Result, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	offset := 0
	if o, ok := args["offset"].(float64); ok {
		offset = int(o)
	}

	limit := 0
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	lines := strings.Split(string(content), "\n")

	if offset > 0 {
		if offset > len(lines) {
			offset = len(lines)
		}
		lines = lines[offset-1:]
	}

	if limit > 0 {
		if limit > len(lines) {
			limit = len(lines)
		}
		lines = lines[:limit]
	}

	result := map[string]interface{}{
		"content": strings.Join(lines, "\n"),
		"lines":   len(lines),
		"path":    path,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) writeFile(args map[string]interface{}) (*loader.Result, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	content, ok := args["content"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "content is required"}, nil
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return &loader.Result{Success: false, Error: fmt.Sprintf("stat after write: %s", err.Error())}, nil
	}

	result := map[string]interface{}{
		"success": true,
		"path":    path,
		"size":    info.Size(),
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) listDirectory(args map[string]interface{}) (*loader.Result, error) {
	path, ok := args["path"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "path is required"}, nil
	}

	recursive := false
	if r, ok := args["recursive"].(bool); ok {
		recursive = r
	}

	var files, directories []string

	if recursive {
		filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() && p != path {
				directories = append(directories, p+"/")
			} else if !d.IsDir() {
				files = append(files, p)
			}
			return nil
		})
	} else {
		entries, err := os.ReadDir(path)
		if err != nil {
			return &loader.Result{Success: false, Error: err.Error()}, nil
		}
		for _, entry := range entries {
			fullPath := filepath.Join(path, entry.Name())
			if entry.IsDir() {
				directories = append(directories, fullPath+"/")
			} else {
				files = append(files, fullPath)
			}
		}
	}

	result := map[string]interface{}{
		"files":       files,
		"directories": directories,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) searchFiles(args map[string]interface{}) (*loader.Result, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "pattern is required"}, nil
	}

	searchPath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		searchPath = p
	}

	var matches []string

	matches, err := filepath.Glob(filepath.Join(searchPath, pattern))
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	result := map[string]interface{}{
		"matches": matches,
	}

	return &loader.Result{Success: true, Data: result}, nil
}

func (s *Skill) searchContent(args map[string]interface{}) (*loader.Result, error) {
	pattern, ok := args["pattern"].(string)
	if !ok {
		return &loader.Result{Success: false, Error: "pattern is required"}, nil
	}

	searchPath := "."
	if p, ok := args["path"].(string); ok && p != "" {
		searchPath = p
	}

	include := ""
	if i, ok := args["include"].(string); ok {
		include = i
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return &loader.Result{Success: false, Error: err.Error()}, nil
	}

	var matches []map[string]interface{}

	filepath.WalkDir(searchPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if include != "" {
			matched, err := filepath.Match(include, filepath.Base(path))
			if err != nil || !matched {
				return nil
			}
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			if re.MatchString(line) {
				matches = append(matches, map[string]interface{}{
					"file":    path,
					"line":    i + 1,
					"content": line,
				})
			}
		}

		return nil
	})

	result := map[string]interface{}{
		"matches": matches,
	}

	return &loader.Result{Success: true, Data: result}, nil
}
