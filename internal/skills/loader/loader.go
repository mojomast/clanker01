package loader

import (
	"context"
	"fmt"
	"sync"
)

// Loader defines the interface for loading skills
type Loader interface {
	Load(ctx context.Context, manifest *SkillManifest) (Instance, error)
	Unload(ctx context.Context, name string) error
	IsLoaded(name string) bool
}

// Instance defines the interface for a loaded skill instance
type Instance interface {
	Meta() *SkillManifest
	Initialize(ctx context.Context, config *Config) error
	Shutdown(ctx context.Context) error
	Tools() []Tool
	Execute(ctx context.Context, toolName string, args map[string]interface{}) (*Result, error)
}

// Tool represents a tool exposed by a skill
type Tool struct {
	Type     string
	Function FunctionDef
}

// FunctionDef defines a function tool
type FunctionDef struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
}

// MultiRuntimeLoader manages multiple runtime loaders
type MultiRuntimeLoader struct {
	loaders map[string]Loader
	mu      sync.RWMutex
	loaded  map[string]Instance
	sandbox *Sandbox
}

// NewMultiRuntimeLoader creates a new multi-runtime loader
func NewMultiRuntimeLoader() *MultiRuntimeLoader {
	l := &MultiRuntimeLoader{
		loaders: make(map[string]Loader),
		loaded:  make(map[string]Instance),
		sandbox: NewSandbox(nil),
	}
	l.loaders["go"] = &GoLoader{buildDir: "/tmp/skills/build"}
	l.loaders["python"] = &PythonLoader{venvDir: "/tmp/skills/venvs"}
	l.loaders["nodejs"] = &NodeLoader{}
	l.loaders["wasm"] = &WasmLoader{}
	l.loaders["native"] = &NativeLoader{}
	return l
}

// NewMultiRuntimeLoaderWithSandbox creates a new multi-runtime loader with custom sandbox
func NewMultiRuntimeLoaderWithSandbox(sandboxConfig *SandboxConfig) *MultiRuntimeLoader {
	l := &MultiRuntimeLoader{
		loaders: make(map[string]Loader),
		loaded:  make(map[string]Instance),
		sandbox: NewSandbox(sandboxConfig),
	}
	l.loaders["go"] = &GoLoader{buildDir: "/tmp/skills/build"}
	l.loaders["python"] = &PythonLoader{venvDir: "/tmp/skills/venvs"}
	l.loaders["nodejs"] = &NodeLoader{}
	l.loaders["wasm"] = &WasmLoader{}
	l.loaders["native"] = &NativeLoader{}
	return l
}

// Load loads a skill from its manifest
func (l *MultiRuntimeLoader) Load(
	ctx context.Context,
	manifest *SkillManifest,
) (Instance, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	key := manifest.Metadata.Name
	if inst, ok := l.loaded[key]; ok {
		return inst, nil
	}

	loader, ok := l.loaders[manifest.Spec.Runtime]
	if !ok {
		return nil, fmt.Errorf("unsupported runtime: %s", manifest.Spec.Runtime)
	}

	inst, err := loader.Load(ctx, manifest)
	if err != nil {
		return nil, fmt.Errorf("load skill %s: %w", key, err)
	}

	l.loaded[key] = inst
	return inst, nil
}

// Unload unloads a skill
func (l *MultiRuntimeLoader) Unload(ctx context.Context, name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	inst, ok := l.loaded[name]
	if !ok {
		return nil
	}

	if err := inst.Shutdown(ctx); err != nil {
		return err
	}

	delete(l.loaded, name)
	return nil
}

// IsLoaded checks if a skill is loaded
func (l *MultiRuntimeLoader) IsLoaded(name string) bool {
	l.mu.RLock()
	defer l.mu.RUnlock()
	_, ok := l.loaded[name]
	return ok
}

// ListLoaded returns a list of loaded skill names
func (l *MultiRuntimeLoader) ListLoaded() []string {
	l.mu.RLock()
	defer l.mu.RUnlock()

	names := make([]string, 0, len(l.loaded))
	for name := range l.loaded {
		names = append(names, name)
	}
	return names
}

// GetLoadedInstance returns a loaded skill instance
func (l *MultiRuntimeLoader) GetLoadedInstance(name string) (Instance, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	inst, ok := l.loaded[name]
	return inst, ok
}
