package loader

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// PythonLoader loads Python skills
type PythonLoader struct {
	venvDir string
}

// Load loads a Python skill
func (l *PythonLoader) Load(
	ctx context.Context,
	manifest *SkillManifest,
) (Instance, error) {
	entrypoint := manifest.Spec.Entrypoint

	// Check if entrypoint is already an absolute path
	if !filepath.IsAbs(entrypoint) {
		return nil, fmt.Errorf("entrypoint must be absolute path: %s", entrypoint)
	}

	// Create or get virtual environment
	venvPath, err := l.ensureVenv(ctx, manifest)
	if err != nil {
		return nil, fmt.Errorf("setup venv: %w", err)
	}

	// Determine Python executable path
	pythonExe := l.getPythonExecutable(venvPath)

	// Install dependencies if specified
	if len(manifest.Spec.Dependencies) > 0 {
		if err := l.installDependencies(ctx, manifest, venvPath); err != nil {
			return nil, fmt.Errorf("install dependencies: %w", err)
		}
	}

	return &ProcessSkill{
		manifest: manifest,
		cmd:      pythonExe,
		args:     []string{entrypoint},
		sandbox:  nil,
	}, nil
}

// Unload unloads a Python skill
func (l *PythonLoader) Unload(ctx context.Context, name string) error {
	// Cleanup venv
	venvPath := filepath.Join(l.venvDir, name)
	os.RemoveAll(venvPath)
	return nil
}

// IsLoaded checks if a skill is loaded
func (l *PythonLoader) IsLoaded(name string) bool {
	venvPath := filepath.Join(l.venvDir, name)
	_, err := os.Stat(venvPath)
	return err == nil
}

// ensureVenv creates or retrieves a virtual environment
func (l *PythonLoader) ensureVenv(ctx context.Context, manifest *SkillManifest) (string, error) {
	if err := os.MkdirAll(l.venvDir, 0755); err != nil {
		return "", fmt.Errorf("create venv dir: %w", err)
	}

	venvPath := filepath.Join(l.venvDir, manifest.Metadata.Name)

	// Check if venv already exists
	if _, err := os.Stat(venvPath); err == nil {
		return venvPath, nil
	}

	// Create venv
	pythonCmd := "python3"
	if runtime.GOOS == "windows" {
		pythonCmd = "python"
	}

	cmd := exec.CommandContext(ctx, pythonCmd, "-m", "venv", venvPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("create venv: %s: %w", string(out), err)
	}

	return venvPath, nil
}

// getPythonExecutable returns the Python executable path in a venv
func (l *PythonLoader) getPythonExecutable(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "python.exe")
	}
	return filepath.Join(venvPath, "bin", "python")
}

// installDependencies installs Python dependencies
func (l *PythonLoader) installDependencies(ctx context.Context, manifest *SkillManifest, venvPath string) error {
	pipExe := l.getPipExecutable(venvPath)

	// Collect Python dependencies
	var deps []string
	for _, dep := range manifest.Spec.Dependencies {
		if dep.Source == "pypi" || dep.Source == "" {
			if dep.Version != "" {
				deps = append(deps, fmt.Sprintf("%s%s", dep.Name, dep.Version))
			} else {
				deps = append(deps, dep.Name)
			}
		}
	}

	if len(deps) == 0 {
		return nil
	}

	// Install dependencies
	args := []string{"install"}
	args = append(args, deps...)

	cmd := exec.CommandContext(ctx, pipExe, args...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pip install: %s: %w", string(out), err)
	}

	return nil
}

// getPipExecutable returns the pip executable path in a venv
func (l *PythonLoader) getPipExecutable(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Scripts", "pip.exe")
	}
	return filepath.Join(venvPath, "bin", "pip")
}

// GetVenvPath returns the venv path for a skill
func (l *PythonLoader) GetVenvPath(name string) string {
	return filepath.Join(l.venvDir, name)
}

// ListVenvs returns a list of all virtual environments
func (l *PythonLoader) ListVenvs() ([]string, error) {
	entries, err := os.ReadDir(l.venvDir)
	if err != nil {
		return nil, err
	}

	var venvs []string
	for _, entry := range entries {
		if entry.IsDir() {
			venvs = append(venvs, entry.Name())
		}
	}

	return venvs, nil
}

// RemoveVenv removes a virtual environment
func (l *PythonLoader) RemoveVenv(name string) error {
	venvPath := filepath.Join(l.venvDir, name)
	return os.RemoveAll(venvPath)
}

// VerifyVenv verifies that a virtual environment is valid
func (l *PythonLoader) VerifyVenv(name string) bool {
	venvPath := filepath.Join(l.venvDir, name)
	pythonExe := l.getPythonExecutable(venvPath)

	if _, err := os.Stat(pythonExe); err != nil {
		return false
	}

	// Try to run python --version
	cmd := exec.Command(pythonExe, "--version")
	return cmd.Run() == nil
}

// GetVenvPythonVersion returns the Python version in a venv
func (l *PythonLoader) GetVenvPythonVersion(name string) (string, error) {
	venvPath := filepath.Join(l.venvDir, name)
	pythonExe := l.getPythonExecutable(venvPath)

	cmd := exec.Command(pythonExe, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}
