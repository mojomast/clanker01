package loader

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// WasmLoader loads WebAssembly skills
type WasmLoader struct {
	wasmRuntime string
	wasmDir     string
}

// Load loads a WASM skill
func (l *WasmLoader) Load(
	ctx context.Context,
	manifest *SkillManifest,
) (Instance, error) {
	entrypoint := manifest.Spec.Entrypoint

	// Check if entrypoint is a .wasm file
	if filepath.Ext(entrypoint) != ".wasm" {
		return nil, fmt.Errorf("entrypoint must be a .wasm file: %s", entrypoint)
	}

	// Verify WASM file exists
	if _, err := os.Stat(entrypoint); err != nil {
		return nil, fmt.Errorf("wasm file not found: %s: %w", entrypoint, err)
	}

	// Determine WASM runtime
	runtime := l.getWasmRuntime()

	// Run WASM with wasmtime or other runtime
	return &ProcessSkill{
		manifest: manifest,
		cmd:      runtime,
		args:     []string{"run", entrypoint},
		sandbox:  nil,
	}, nil
}

// Unload unloads a WASM skill
func (l *WasmLoader) Unload(ctx context.Context, name string) error {
	// WASM skills are stateless, nothing to clean up
	return nil
}

// IsLoaded checks if a skill is loaded
func (l *WasmLoader) IsLoaded(name string) bool {
	// WASM skills are stateless, always return false
	return false
}

// getWasmRuntime returns the WASM runtime to use
func (l *WasmLoader) getWasmRuntime() string {
	if l.wasmRuntime != "" {
		return l.wasmRuntime
	}

	// Try to find available WASM runtimes
	runtimes := []string{"wasmtime", "wasm3", "wasmer", "wasmedge"}
	for _, runtime := range runtimes {
		if _, err := exec.LookPath(runtime); err == nil {
			return runtime
		}
	}

	// Default to wasmtime
	return "wasmtime"
}

// VerifyWasmFile verifies that a file is a valid WASM binary
func (l *WasmLoader) VerifyWasmFile(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()

	// Check WASM magic number (0x00 0x61 0x73 0x6D)
	magic := make([]byte, 4)
	if _, err := file.Read(magic); err != nil {
		return false, err
	}

	return magic[0] == 0x00 && magic[1] == 0x61 && magic[2] == 0x73 && magic[3] == 0x6D, nil
}

// CompileWasm compiles a WASM file (if needed)
func (l *WasmLoader) CompileWasm(ctx context.Context, sourcePath string, outputPath string) error {
	runtime := l.getWasmRuntime()

	// Some runtimes support ahead-of-time compilation
	cmd := exec.CommandContext(ctx, runtime, "compile", sourcePath, "-o", outputPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("compile wasm: %s: %w", string(out), err)
	}

	return nil
}

// GetWasmVersion returns the WASM runtime version
func (l *WasmLoader) GetWasmVersion() (string, error) {
	runtime := l.getWasmRuntime()

	cmd := exec.Command(runtime, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// ValidateWasm validates a WASM file
func (l *WasmLoader) ValidateWasm(ctx context.Context, path string) error {
	runtime := l.getWasmRuntime()

	// Use runtime to validate
	cmd := exec.CommandContext(ctx, runtime, "validate", path)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("validate wasm: %w", err)
	}

	return nil
}

// GetWasmMetadata extracts metadata from a WASM file
func (l *WasmLoader) GetWasmMetadata(path string) (map[string]string, error) {
	_ = l.getWasmRuntime()

	cmd := exec.Command("wasm-tools", "metadata", path)
	_, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// Parse metadata
	metadata := make(map[string]string)
	// TODO: Implement proper metadata parsing

	return metadata, nil
}

// OptimizeWasm optimizes a WASM file
func (l *WasmLoader) OptimizeWasm(ctx context.Context, sourcePath string, outputPath string) error {
	_ = l.getWasmRuntime()

	// Use wasm-opt if available, otherwise skip
	if _, err := exec.LookPath("wasm-opt"); err != nil {
		// Copy file as fallback
		src, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		return os.WriteFile(outputPath, src, 0644)
	}

	cmd := exec.CommandContext(ctx, "wasm-opt", "-Oz", sourcePath, "-o", outputPath)
	if _, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("optimize wasm: %w", err)
	}

	return nil
}

// ReadWasmBinary reads a WASM binary file
func (l *WasmLoader) ReadWasmBinary(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// WriteWasmBinary writes a WASM binary file
func (l *WasmLoader) WriteWasmBinary(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// ListWasmFiles lists all .wasm files in a directory
func (l *WasmLoader) ListWasmFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".wasm" {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

// CreateWasmDir creates a directory for WASM files
func (l *WasmLoader) CreateWasmDir(name string) (string, error) {
	if l.wasmDir == "" {
		l.wasmDir = "/tmp/skills/wasm"
	}

	dir := filepath.Join(l.wasmDir, name)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create wasm dir: %w", err)
	}

	return dir, nil
}

// CleanupWasmDir removes a WASM directory
func (l *WasmLoader) CleanupWasmDir(name string) error {
	if l.wasmDir == "" {
		return nil
	}

	dir := filepath.Join(l.wasmDir, name)
	return os.RemoveAll(dir)
}

// GetWasmDir returns the WASM directory for a skill
func (l *WasmLoader) GetWasmDir(name string) string {
	if l.wasmDir == "" {
		return ""
	}
	return filepath.Join(l.wasmDir, name)
}
