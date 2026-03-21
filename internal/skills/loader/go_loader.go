package loader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// GoLoader loads Go skills
type GoLoader struct {
	buildDir string
}

// Load loads a Go skill
func (l *GoLoader) Load(
	ctx context.Context,
	manifest *SkillManifest,
) (Instance, error) {
	entrypoint := manifest.Spec.Entrypoint

	// If entrypoint is a binary path, use directly
	if filepath.IsAbs(entrypoint) || isBinary(entrypoint) {
		return &ProcessSkill{
			manifest: manifest,
			cmd:      entrypoint,
			sandbox:  nil, // Will be set by MultiRuntimeLoader
		}, nil
	}

	// Otherwise build from source
	buildDir := filepath.Join(l.buildDir, manifest.Metadata.Name)
	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return nil, fmt.Errorf("create build dir: %w", err)
	}

	// Build the skill
	cmd := exec.CommandContext(ctx, "go", "build", "-o", "skill", ".")
	cmd.Dir = filepath.Dir(entrypoint)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("build: %s: %w", string(out), err)
	}

	binaryPath := filepath.Join(filepath.Dir(entrypoint), "skill")

	return &ProcessSkill{
		manifest: manifest,
		cmd:      binaryPath,
		sandbox:  nil,
	}, nil
}

// Unload unloads a Go skill
func (l *GoLoader) Unload(ctx context.Context, name string) error {
	// Cleanup build artifacts
	buildDir := filepath.Join(l.buildDir, name)
	os.RemoveAll(buildDir)
	return nil
}

// IsLoaded checks if a skill is loaded
func (l *GoLoader) IsLoaded(name string) bool {
	// Go skills are always process-based, so this is a no-op
	return false
}

// isBinary checks if a file is a binary
func isBinary(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if executable
	return info.Mode().Perm()&0111 != 0
}
