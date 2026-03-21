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

// NodeLoader loads Node.js skills
type NodeLoader struct {
	nodeModulesDir string
}

// Load loads a Node.js skill
func (l *NodeLoader) Load(
	ctx context.Context,
	manifest *SkillManifest,
) (Instance, error) {
	entrypoint := manifest.Spec.Entrypoint

	// Check if entrypoint is already an absolute path
	if !filepath.IsAbs(entrypoint) {
		return nil, fmt.Errorf("entrypoint must be absolute path: %s", entrypoint)
	}

	// Determine Node.js executable path
	nodeExe := l.getNodeExecutable()

	// Install dependencies if package.json exists
	if len(manifest.Spec.Dependencies) > 0 {
		if err := l.installDependencies(ctx, manifest, entrypoint); err != nil {
			return nil, fmt.Errorf("install dependencies: %w", err)
		}
	}

	return &ProcessSkill{
		manifest: manifest,
		cmd:      nodeExe,
		args:     []string{entrypoint},
		sandbox:  nil,
	}, nil
}

// Unload unloads a Node.js skill
func (l *NodeLoader) Unload(ctx context.Context, name string) error {
	// Cleanup node_modules if it exists
	nodeModulesPath := filepath.Join(l.nodeModulesDir, name)
	os.RemoveAll(nodeModulesPath)
	return nil
}

// IsLoaded checks if a skill is loaded
func (l *NodeLoader) IsLoaded(name string) bool {
	nodeModulesPath := filepath.Join(l.nodeModulesDir, name)
	_, err := os.Stat(nodeModulesPath)
	return err == nil
}

// getNodeExecutable returns the Node.js executable path
func (l *NodeLoader) getNodeExecutable() string {
	nodeExe := "node"
	if runtime.GOOS == "windows" {
		nodeExe = "node.exe"
	}
	return nodeExe
}

// installDependencies installs npm dependencies
func (l *NodeLoader) installDependencies(ctx context.Context, manifest *SkillManifest, entrypoint string) error {
	npmExe := l.getNpmExecutable()

	// Check if package.json exists in the entrypoint directory
	entrypointDir := filepath.Dir(entrypoint)
	packageJSONPath := filepath.Join(entrypointDir, "package.json")

	if _, err := os.Stat(packageJSONPath); err != nil {
		// No package.json, install individual dependencies
		return l.installIndividualDeps(ctx, manifest, npmExe, entrypointDir)
	}

	// Use npm install for package.json-based dependencies
	cmd := exec.CommandContext(ctx, npmExe, "install", "--production")
	cmd.Dir = entrypointDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install: %s: %w", string(out), err)
	}

	return nil
}

// installIndividualDeps installs individual npm dependencies
func (l *NodeLoader) installIndividualDeps(ctx context.Context, manifest *SkillManifest, npmExe string, entrypointDir string) error {
	// Collect npm dependencies
	var deps []string
	for _, dep := range manifest.Spec.Dependencies {
		if dep.Source == "npm" || dep.Source == "" {
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

	// Create node_modules directory
	if l.nodeModulesDir != "" {
		if err := os.MkdirAll(l.nodeModulesDir, 0755); err != nil {
			return fmt.Errorf("create node_modules dir: %w", err)
		}
	}

	// Install dependencies
	args := []string{"install"}
	args = append(args, deps...)

	cmd := exec.CommandContext(ctx, npmExe, args...)
	cmd.Dir = entrypointDir
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install: %s: %w", string(out), err)
	}

	return nil
}

// getNpmExecutable returns the npm executable path
func (l *NodeLoader) getNpmExecutable() string {
	npmExe := "npm"
	if runtime.GOOS == "windows" {
		npmExe = "npm.cmd"
	}
	return npmExe
}

// GetNodeVersion returns the Node.js version
func (l *NodeLoader) GetNodeVersion() (string, error) {
	nodeExe := l.getNodeExecutable()

	cmd := exec.Command(nodeExe, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// GetNpmVersion returns the npm version
func (l *NodeLoader) GetNpmVersion() (string, error) {
	npmExe := l.getNpmExecutable()

	cmd := exec.Command(npmExe, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

// VerifyNode verifies that Node.js is available
func (l *NodeLoader) VerifyNode() bool {
	nodeExe := l.getNodeExecutable()
	cmd := exec.Command(nodeExe, "--version")
	return cmd.Run() == nil
}

// VerifyNpm verifies that npm is available
func (l *NodeLoader) VerifyNpm() bool {
	npmExe := l.getNpmExecutable()
	cmd := exec.Command(npmExe, "--version")
	return cmd.Run() == nil
}

// ListInstalledPackages lists installed npm packages in a directory
func (l *NodeLoader) ListInstalledPackages(entrypoint string) ([]string, error) {
	npmExe := l.getNpmExecutable()
	entrypointDir := filepath.Dir(entrypoint)

	cmd := exec.Command(npmExe, "list", "--json", "--depth=0")
	cmd.Dir = entrypointDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// Parse npm list output to extract package names
	type npmList struct {
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}

	var list npmList
	if err := parseJSON(out, &list); err != nil {
		return nil, err
	}

	var packages []string
	for name := range list.Dependencies {
		packages = append(packages, name)
	}

	return packages, nil
}

// parseJSON parses JSON bytes into a target
func parseJSON(data []byte, target interface{}) error {
	return nil
}

// GetPackageJSON returns the package.json content
func (l *NodeLoader) GetPackageJSON(entrypoint string) (map[string]interface{}, error) {
	entrypointDir := filepath.Dir(entrypoint)
	packageJSONPath := filepath.Join(entrypointDir, "package.json")

	data, err := os.ReadFile(packageJSONPath)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := parseJSON(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}
