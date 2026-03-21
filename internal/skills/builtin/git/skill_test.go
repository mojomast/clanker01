package git

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/swarm-ai/swarm/internal/skills/loader"
)

func TestNewSkill(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	assert.NotNil(t, skill)
	assert.NotNil(t, skill.Meta())
	assert.Equal(t, "git", skill.Meta().Metadata.Name)
	assert.Equal(t, "1.0.0", skill.Meta().Metadata.Version)
}

func TestSkill_Meta(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	manifest := skill.Meta()

	assert.Equal(t, "swarm.ai/v1", manifest.APIVersion)
	assert.Equal(t, "Skill", manifest.Kind)
	assert.Equal(t, "git", manifest.Metadata.Name)
	assert.Equal(t, "1.0.0", manifest.Metadata.Version)
	assert.Equal(t, "Git Operations", manifest.Metadata.DisplayName)
	assert.NotEmpty(t, manifest.Metadata.Description)
	assert.Len(t, manifest.Spec.Tools, 6)
}

func TestSkill_Initialize(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	err := skill.Initialize(context.Background(), &loader.Config{})
	assert.NoError(t, err)
}

func TestSkill_Shutdown(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	err := skill.Shutdown(context.Background())
	assert.NoError(t, err)
}

func TestSkill_Tools(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	tools := skill.Tools()

	assert.Len(t, tools, 6)

	toolNames := make([]string, len(tools))
	for i, tool := range tools {
		toolNames[i] = tool.Function.Name
	}

	assert.Contains(t, toolNames, "clone")
	assert.Contains(t, toolNames, "status")
	assert.Contains(t, toolNames, "commit")
	assert.Contains(t, toolNames, "branch")
	assert.Contains(t, toolNames, "log")
	assert.Contains(t, toolNames, "diff")
}

func setupTestRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	err := cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, ".gitkeep")
	err = os.WriteFile(testFile, []byte(""), 0644)
	require.NoError(t, err)

	cmd = exec.Command("git", "add", ".")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tmpDir
	err = cmd.Run()
	require.NoError(t, err)

	return tmpDir
}

func TestSkill_Status(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	testFile := filepath.Join(repoPath, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "status", map[string]interface{}{
		"path": repoPath,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data := result.Data.(map[string]interface{})
	assert.Contains(t, []string{"main", "master"}, data["branch"])
}

func TestSkill_Commit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	testFile := filepath.Join(repoPath, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "commit", map[string]interface{}{
		"path":    repoPath,
		"message": "Add test file",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Data.(map[string]interface{})["hash"])
}

func TestSkill_Commit_WithFiles(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	testFile1 := filepath.Join(repoPath, "test1.txt")
	testFile2 := filepath.Join(repoPath, "test2.txt")
	err := os.WriteFile(testFile1, []byte("test content 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("test content 2"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test1.txt", "test2.txt")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "commit", map[string]interface{}{
		"path":    repoPath,
		"message": "Commit specific files",
		"files":   []interface{}{"test1.txt", "test2.txt"},
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestSkill_Branch_List(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	result, err := skill.Execute(context.Background(), "branch", map[string]interface{}{
		"path":   repoPath,
		"action": "list",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	data := result.Data.(map[string]interface{})
	assert.Contains(t, []string{"main", "master"}, data["current"])
	assert.NotEmpty(t, data["branches"])
}

func TestSkill_Branch_Create(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	result, err := skill.Execute(context.Background(), "branch", map[string]interface{}{
		"path":   repoPath,
		"action": "create",
		"name":   "test-branch",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "test-branch", result.Data.(map[string]interface{})["name"])
}

func TestSkill_Branch_Create_WithCheckout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	result, err := skill.Execute(context.Background(), "branch", map[string]interface{}{
		"path":     repoPath,
		"action":   "create",
		"name":     "test-branch-2",
		"checkout": true,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestSkill_Branch_Checkout(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	_, err := skill.Execute(context.Background(), "branch", map[string]interface{}{
		"path":   repoPath,
		"action": "create",
		"name":   "test-branch",
	})
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "branch", map[string]interface{}{
		"path":   repoPath,
		"action": "checkout",
		"name":   "test-branch",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestSkill_Branch_Delete(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	_, err := skill.Execute(context.Background(), "branch", map[string]interface{}{
		"path":   repoPath,
		"action": "create",
		"name":   "test-branch",
	})
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "branch", map[string]interface{}{
		"path":   repoPath,
		"action": "delete",
		"name":   "test-branch",
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
}

func TestSkill_Log(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	testFile := filepath.Join(repoPath, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	_, err = skill.Execute(context.Background(), "commit", map[string]interface{}{
		"path":    repoPath,
		"message": "Initial commit",
	})
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "log", map[string]interface{}{
		"path":  repoPath,
		"limit": 10.0,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)

	commits := result.Data.(map[string]interface{})["commits"].([]map[string]string)
	assert.Len(t, commits, 1)
	assert.Equal(t, "Initial commit", commits[0]["message"])
}

func TestSkill_Diff(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()
	repoPath := setupTestRepo(t)

	testFile := filepath.Join(repoPath, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	cmd = exec.Command("git", "commit", "-m", "Add test file")
	cmd.Dir = repoPath
	err = cmd.Run()
	require.NoError(t, err)

	err = os.WriteFile(testFile, []byte("test content modified"), 0644)
	require.NoError(t, err)

	result, err := skill.Execute(context.Background(), "diff", map[string]interface{}{
		"path": repoPath,
	})
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.Data.(map[string]interface{})["diff"])
}

func TestSkill_GitCommand_MissingPath(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()

	result, err := skill.Execute(context.Background(), "status", map[string]interface{}{})
	require.NoError(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Error, "path is required")
}

func TestSkill_Execute_UnknownTool(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found, skipping tests")
	}

	skill := NewSkill()

	_, err := skill.Execute(context.Background(), "unknown_tool", map[string]interface{}{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
}
