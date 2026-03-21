package loader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// NativeLoader loads native binary skills
type NativeLoader struct {
	binDir string
}

// Load loads a native skill
func (l *NativeLoader) Load(
	ctx context.Context,
	manifest *SkillManifest,
) (Instance, error) {
	entrypoint := manifest.Spec.Entrypoint

	// Check if entrypoint is an absolute path
	if filepath.IsAbs(entrypoint) {
		// Verify it's a valid executable
		if !l.isExecutable(entrypoint) {
			return nil, fmt.Errorf("not an executable: %s", entrypoint)
		}

		return &ProcessSkill{
			manifest: manifest,
			cmd:      entrypoint,
			sandbox:  nil,
		}, nil
	}

	// Search in PATH and binDir
	path := l.findExecutable(entrypoint)
	if path == "" {
		return nil, fmt.Errorf("executable not found: %s", entrypoint)
	}

	return &ProcessSkill{
		manifest: manifest,
		cmd:      path,
		sandbox:  nil,
	}, nil
}

// Unload unloads a native skill
func (l *NativeLoader) Unload(ctx context.Context, name string) error {
	// Native skills are stateless, nothing to clean up
	return nil
}

// IsLoaded checks if a skill is loaded
func (l *NativeLoader) IsLoaded(name string) bool {
	// Native skills are stateless, always return false
	return false
}

// isExecutable checks if a file is executable
func (l *NativeLoader) isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	// Check if it's a regular file
	if !info.Mode().IsRegular() {
		return false
	}

	// Check if executable bit is set
	return info.Mode().Perm()&0111 != 0
}

// findExecutable searches for an executable in PATH and binDir
func (l *NativeLoader) findExecutable(name string) string {
	// First check binDir
	if l.binDir != "" {
		path := filepath.Join(l.binDir, name)
		if l.isExecutable(path) {
			return path
		}
	}

	// Then check PATH
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	return ""
}

// ListExecutables lists all executables in binDir
func (l *NativeLoader) ListExecutables() ([]string, error) {
	if l.binDir == "" {
		return []string{}, nil
	}

	entries, err := os.ReadDir(l.binDir)
	if err != nil {
		return nil, err
	}

	var executables []string
	for _, entry := range entries {
		if !entry.IsDir() {
			path := filepath.Join(l.binDir, entry.Name())
			if l.isExecutable(path) {
				executables = append(executables, entry.Name())
			}
		}
	}

	return executables, nil
}

// AddExecutable copies an executable to binDir
func (l *NativeLoader) AddExecutable(sourcePath, name string) (string, error) {
	if l.binDir == "" {
		l.binDir = "/tmp/skills/bin"
	}

	if err := os.MkdirAll(l.binDir, 0755); err != nil {
		return "", fmt.Errorf("create bin dir: %w", err)
	}

	destPath := filepath.Join(l.binDir, name)

	// Copy file
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return "", fmt.Errorf("read executable: %w", err)
	}

	if err := os.WriteFile(destPath, data, 0755); err != nil {
		return "", fmt.Errorf("write executable: %w", err)
	}

	return destPath, nil
}

// RemoveExecutable removes an executable from binDir
func (l *NativeLoader) RemoveExecutable(name string) error {
	if l.binDir == "" {
		return nil
	}

	path := filepath.Join(l.binDir, name)
	return os.Remove(path)
}

// GetExecutablePath returns the full path to an executable
func (l *NativeLoader) GetExecutablePath(name string) string {
	// First check binDir
	if l.binDir != "" {
		path := filepath.Join(l.binDir, name)
		if l.isExecutable(path) {
			return path
		}
	}

	// Then check PATH
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	return ""
}

// VerifyExecutable verifies that an executable exists and is runnable
func (l *NativeLoader) VerifyExecutable(name string) bool {
	path := l.findExecutable(name)
	if path == "" {
		return false
	}

	// Try to run the executable with --version or --help
	for _, flag := range []string{"--version", "--help", "-v", "-h"} {
		cmd := exec.Command(path, flag)
		if cmd.Run() == nil {
			return true
		}
	}

	// If no version flag works, just check if it exists
	return l.isExecutable(path)
}

// GetExecutableInfo returns information about an executable
func (l *NativeLoader) GetExecutableInfo(path string) (*ExecutableInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	return &ExecutableInfo{
		Name:    filepath.Base(path),
		Path:    absPath,
		Size:    info.Size(),
		Mode:    info.Mode(),
		ModTime: info.ModTime(),
	}, nil
}

// ExecutableInfo contains information about an executable
type ExecutableInfo struct {
	Name    string
	Path    string
	Size    int64
	Mode    os.FileMode
	ModTime interface{}
}
