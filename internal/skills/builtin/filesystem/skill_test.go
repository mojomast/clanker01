package filesystem

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-ai/swarm/internal/skills/loader"
)

func TestNewSkill(t *testing.T) {
	skill := NewSkill()
	assert.NotNil(t, skill)
	assert.NotNil(t, skill.Meta())
	assert.Equal(t, "filesystem", skill.Meta().Metadata.Name)
	assert.Equal(t, "1.0.0", skill.Meta().Metadata.Version)
}

func TestSkill_Meta(t *testing.T) {
	skill := NewSkill()
	manifest := skill.Meta()

	assert.Equal(t, "swarm.ai/v1", manifest.APIVersion)
	assert.Equal(t, "Skill", manifest.Kind)
	assert.Equal(t, "filesystem", manifest.Metadata.Name)
	assert.Equal(t, "1.0.0", manifest.Metadata.Version)
	assert.Equal(t, "Filesystem Operations", manifest.Metadata.DisplayName)
	assert.NotEmpty(t, manifest.Metadata.Description)
	assert.Len(t, manifest.Spec.Tools, 5)
}

func TestSkill_Initialize(t *testing.T) {
	skill := NewSkill()
	err := skill.Initialize(context.Background(), &loader.Config{})
	assert.NoError(t, err)
}

func TestSkill_Shutdown(t *testing.T) {
	skill := NewSkill()
	err := skill.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestSkill_Tools(t *testing.T) {
	skill := NewSkill()
	tools := skill.Tools()

	assert.Len(t, tools, 5)

	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	assert.Contains(t, toolNames, "read_file")
	assert.Contains(t, toolNames, "write_file")
	assert.Contains(t, toolNames, "list_directory")
	assert.Contains(t, toolNames, "search_files")
	assert.Contains(t, toolNames, "search_content")
}

func TestSkill_ReadFile(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Hello, World!\nLine 2\nLine 3"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "read_file", map[string]interface{}{
		"path": testFile,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, content, result.Data.(map[string]interface{})["content"])
	assert.Equal(t, 3, result.Data.(map[string]interface{})["lines"])
}

func TestSkill_ReadFileWithOffsetLimit(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	err := os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "read_file", map[string]interface{}{
		"path":   testFile,
		"offset": 2.0,
		"limit":  2.0,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	resultContent := result.Data.(map[string]interface{})["content"].(string)
	expected := "Line 2\nLine 3"
	assert.Equal(t, expected, resultContent)
	assert.Equal(t, 2, result.Data.(map[string]interface{})["lines"])
}

func TestSkill_ReadFile_NotFound(t *testing.T) {
	skill := NewSkill()

	result, err := skill.Execute(context.Background(), "read_file", map[string]interface{}{
		"path": "/nonexistent/file.txt",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.NotEmpty(t, result.Error)
}

func TestSkill_ReadFile_MissingPath(t *testing.T) {
	skill := NewSkill()

	result, err := skill.Execute(context.Background(), "read_file", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path is required")
}

func TestSkill_WriteFile(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "write_test.txt")
	content := "Test content for writing"

	result, err := skill.Execute(context.Background(), "write_file", map[string]interface{}{
		"path":    testFile,
		"content": content,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	readContent, err := os.ReadFile(testFile)
	require.NoError(t, err)
	assert.Equal(t, content, string(readContent))
}

func TestSkill_WriteFile_MissingPath(t *testing.T) {
	skill := NewSkill()

	result, err := skill.Execute(context.Background(), "write_file", map[string]interface{}{
		"content": "test",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path is required")
}

func TestSkill_WriteFile_MissingContent(t *testing.T) {
	skill := NewSkill()

	result, err := skill.Execute(context.Background(), "write_file", map[string]interface{}{
		"path": "/tmp/test.txt",
	})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "content is required")
}

func TestSkill_ListDirectory(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	err = os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "list_directory", map[string]interface{}{
		"path": tmpDir,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data := result.Data.(map[string]interface{})
	files := data["files"].([]string)
	directories := data["directories"].([]string)

	assert.Len(t, files, 2)
	assert.Len(t, directories, 1)
}

func TestSkill_ListDirectory_Recursive(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	subdir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subdir, 0755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(subdir, "file2.txt"), []byte("content"), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "list_directory", map[string]interface{}{
		"path":      tmpDir,
		"recursive": true,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data := result.Data.(map[string]interface{})
	files := data["files"].([]string)
	assert.Len(t, files, 2)
}

func TestSkill_SearchFiles(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "test1.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "test2.txt"), []byte("content"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "other.log"), []byte("content"), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "search_files", map[string]interface{}{
		"pattern": "*.txt",
		"path":    tmpDir,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	matches := result.Data.(map[string]interface{})["matches"].([]string)
	assert.Len(t, matches, 2)
}

func TestSkill_SearchContent(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "test1.txt"), []byte("Hello World\nGoodbye"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "test2.txt"), []byte("No match here"), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "search_content", map[string]interface{}{
		"pattern": "Hello",
		"path":    tmpDir,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	matches := result.Data.(map[string]interface{})["matches"].([]map[string]interface{})
	assert.Len(t, matches, 1)
	assert.Equal(t, 1, matches[0]["line"])
	assert.True(t, strings.Contains(matches[0]["content"].(string), "Hello"))
}

func TestSkill_SearchContent_WithInclude(t *testing.T) {
	skill := NewSkill()

	tmpDir := t.TempDir()

	err := os.WriteFile(filepath.Join(tmpDir, "test1.txt"), []byte("Hello World"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tmpDir, "test2.go"), []byte("Hello Go"), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "search_content", map[string]interface{}{
		"pattern": "Hello",
		"path":    tmpDir,
		"include": "*.go",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	matches := result.Data.(map[string]interface{})["matches"].([]map[string]interface{})
	assert.Len(t, matches, 1)
	assert.Contains(t, matches[0]["file"].(string), "test2.go")
}

func TestSkill_Execute_UnknownTool(t *testing.T) {
	skill := NewSkill()

	_, err := skill.Execute(context.Background(), "unknown_tool", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
