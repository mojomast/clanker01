package loader

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestMultiRuntimeLoader(t *testing.T) {
	loader := NewMultiRuntimeLoader()

	// Test loader initialization
	if loader == nil {
		t.Fatal("NewMultiRuntimeLoader returned nil")
	}

	// Test that all loaders are registered
	expectedRuntimes := []string{"go", "python", "nodejs", "wasm", "native"}
	for _, runtime := range expectedRuntimes {
		if _, ok := loader.loaders[runtime]; !ok {
			t.Errorf("Runtime %s not registered", runtime)
		}
	}
}

func TestMultiRuntimeLoaderWithSandbox(t *testing.T) {
	config := &SandboxConfig{
		Enabled:       true,
		Profile:       "restricted",
		MaxMemoryMB:   256,
		MaxCPUSeconds: 10,
		Timeout:       30 * time.Second,
		TempDir:       t.TempDir(),
	}

	loader := NewMultiRuntimeLoaderWithSandbox(config)

	if loader.sandbox == nil {
		t.Fatal("Sandbox not initialized")
	}

	if !loader.sandbox.config.Enabled {
		t.Error("Sandbox not enabled")
	}

	if loader.sandbox.config.Profile != "restricted" {
		t.Errorf("Expected profile 'restricted', got '%s'", loader.sandbox.config.Profile)
	}
}

func TestGoLoader(t *testing.T) {
	l := &GoLoader{buildDir: t.TempDir()}

	// Test loading a binary path
	binaryPath := "/bin/echo"
	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "test-binary",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: binaryPath,
			Tools: []ToolDef{
				{
					Name:        "echo",
					Description: "Echoes input",
				},
			},
		},
	}

	inst, err := l.Load(context.Background(), manifest)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if inst == nil {
		t.Fatal("Instance is nil")
	}

	// Test ProcessSkill wrapper
	ps, ok := inst.(*ProcessSkill)
	if !ok {
		t.Fatal("Not a ProcessSkill")
	}

	if ps.cmd != binaryPath {
		t.Errorf("Expected cmd %s, got %s", binaryPath, ps.cmd)
	}

	// Cleanup
	if err := l.Unload(context.Background(), manifest.Metadata.Name); err != nil {
		t.Errorf("Unload failed: %v", err)
	}
}

func TestPythonLoader(t *testing.T) {
	l := &PythonLoader{venvDir: t.TempDir()}

	// Test venv creation
	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "test-python",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "python",
			Entrypoint: "/tmp/test.py",
			Tools: []ToolDef{
				{
					Name:        "test",
					Description: "Test tool",
				},
			},
		},
	}

	// Skip actual Python tests if not available
	_, err := l.Load(context.Background(), manifest)
	if err != nil {
		// This is expected if Python is not installed
		t.Skipf("Python not available: %v", err)
	}

	// Cleanup
	if err := l.Unload(context.Background(), manifest.Metadata.Name); err != nil {
		t.Errorf("Unload failed: %v", err)
	}
}

func TestNodeLoader(t *testing.T) {
	l := &NodeLoader{nodeModulesDir: t.TempDir()}

	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "test-node",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "nodejs",
			Entrypoint: "/tmp/test.js",
			Tools: []ToolDef{
				{
					Name:        "test",
					Description: "Test tool",
				},
			},
		},
	}

	// Skip actual Node tests if not available
	_, err := l.Load(context.Background(), manifest)
	if err != nil {
		// This is expected if Node.js is not installed
		t.Skipf("Node.js not available: %v", err)
	}
}

func TestWasmLoader(t *testing.T) {
	l := &WasmLoader{wasmDir: t.TempDir()}

	// Create a minimal WASM file (magic number + minimal content)
	wasmPath := filepath.Join(t.TempDir(), "test.wasm")
	wasmMagic := []byte{0x00, 0x61, 0x73, 0x6D, 0x01, 0x00, 0x00, 0x00}
	if err := os.WriteFile(wasmPath, wasmMagic, 0644); err != nil {
		t.Fatalf("Failed to create WASM file: %v", err)
	}

	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "test-wasm",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "wasm",
			Entrypoint: wasmPath,
			Tools: []ToolDef{
				{
					Name:        "test",
					Description: "Test tool",
				},
			},
		},
	}

	inst, err := l.Load(context.Background(), manifest)
	if err != nil {
		// This is expected if WASM runtime is not installed
		t.Skipf("WASM runtime not available: %v", err)
	}

	if inst == nil {
		t.Fatal("Instance is nil")
	}

	// Test WASM file verification
	valid, err := l.VerifyWasmFile(wasmPath)
	if err != nil {
		t.Errorf("VerifyWasmFile failed: %v", err)
	}
	if !valid {
		t.Error("WASM file verification failed")
	}

	// Cleanup
	if err := l.Unload(context.Background(), manifest.Metadata.Name); err != nil {
		t.Errorf("Unload failed: %v", err)
	}
}

func TestNativeLoader(t *testing.T) {
	l := &NativeLoader{binDir: t.TempDir()}

	// Test with echo command
	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "test-echo",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "native",
			Entrypoint: "/bin/echo",
			Tools: []ToolDef{
				{
					Name:        "echo",
					Description: "Echoes input",
				},
			},
		},
	}

	inst, err := l.Load(context.Background(), manifest)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if inst == nil {
		t.Fatal("Instance is nil")
	}

	ps, ok := inst.(*ProcessSkill)
	if !ok {
		t.Fatal("Not a ProcessSkill")
	}

	if ps.cmd != "/bin/echo" {
		t.Errorf("Expected cmd /bin/echo, got %s", ps.cmd)
	}

	// Test findExecutable
	path := l.findExecutable("echo")
	if path == "" {
		t.Error("echo not found in PATH")
	}

	// Cleanup
	if err := l.Unload(context.Background(), manifest.Metadata.Name); err != nil {
		t.Errorf("Unload failed: %v", err)
	}
}

func TestSandbox(t *testing.T) {
	config := &SandboxConfig{
		Enabled:       true,
		Profile:       "restricted",
		MaxMemoryMB:   256,
		MaxCPUSeconds: 10,
		Timeout:       5 * time.Second,
		TempDir:       t.TempDir(),
	}

	sandbox := NewSandbox(config)

	if sandbox == nil {
		t.Fatal("NewSandbox returned nil")
	}

	if !sandbox.config.Enabled {
		t.Error("Sandbox not enabled")
	}

	// Test permission checker
	tests := []struct {
		path     string
		mode     string
		expected bool
	}{
		{"/tmp/test.txt", "read", true},
		{"/tmp/test.txt", "write", true},
		{"/etc/passwd", "read", false},
		{"/home/user/test.txt", "read", false},
	}

	for _, tt := range tests {
		result := sandbox.CheckFileAccess(tt.path, tt.mode)
		if result != tt.expected {
			t.Errorf("CheckFileAccess(%s, %s) = %v, want %v", tt.path, tt.mode, result, tt.expected)
		}
	}

	// Test network access
	networkTests := []struct {
		host     string
		expected bool
	}{
		{"example.com", false},
		{"localhost", false},
	}

	for _, tt := range networkTests {
		result := sandbox.CheckNetworkAccess(tt.host)
		if result != tt.expected {
			t.Errorf("CheckNetworkAccess(%s) = %v, want %v", tt.host, result, tt.expected)
		}
	}
}

func TestSandboxStandardProfile(t *testing.T) {
	config := &SandboxConfig{
		Enabled: true,
		Profile: "standard",
		TempDir: t.TempDir(),
	}

	sandbox := NewSandbox(config)

	tests := []struct {
		path     string
		mode     string
		expected bool
	}{
		{"/tmp/test.txt", "read", true},
		{"/tmp/test.txt", "write", true},
		{"/home/user/test.txt", "read", true},
		{"/etc/passwd", "read", false},
	}

	for _, tt := range tests {
		result := sandbox.CheckFileAccess(tt.path, tt.mode)
		if result != tt.expected {
			t.Errorf("CheckFileAccess(%s, %s) = %v, want %v", tt.path, tt.mode, result, tt.expected)
		}
	}

	// Standard profile allows network
	result := sandbox.CheckNetworkAccess("example.com")
	if !result {
		t.Error("Standard profile should allow network access")
	}
}

func TestSandboxElevatedProfile(t *testing.T) {
	config := &SandboxConfig{
		Enabled: true,
		Profile: "elevated",
		TempDir: t.TempDir(),
	}

	sandbox := NewSandbox(config)

	// Elevated profile allows most access
	tests := []struct {
		path     string
		mode     string
		expected bool
	}{
		{"/tmp/test.txt", "read", true},
		{"/tmp/test.txt", "write", true},
		{"/home/user/test.txt", "read", true},
		{"/home/user/test.txt", "write", true},
		{"/etc/passwd", "read", true},
	}

	for _, tt := range tests {
		result := sandbox.CheckFileAccess(tt.path, tt.mode)
		if result != tt.expected {
			t.Errorf("CheckFileAccess(%s, %s) = %v, want %v", tt.path, tt.mode, result, tt.expected)
		}
	}
}

func TestProcessSkill(t *testing.T) {
	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "Test skill",
		},
		Spec: SkillSpec{
			Tools: []ToolDef{
				{
					Name:        "test_tool",
					Description: "Test tool",
				},
			},
		},
	}

	ps := NewProcessSkill(manifest, "/bin/echo", []string{}, nil)

	// Test Meta
	meta := ps.Meta()
	if meta != manifest {
		t.Error("Meta returned wrong manifest")
	}

	// Test Tools
	tools := ps.Tools()
	if len(tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(tools))
	}

	if tools[0].Function.Name != "test_tool" {
		t.Errorf("Expected tool name 'test_tool', got '%s'", tools[0].Function.Name)
	}
}

func TestTypes(t *testing.T) {
	// Test RPCResponse getters
	resp := &RPCResponse{
		Result: map[string]interface{}{
			"success": true,
			"error":   "test error",
			"data":    "test data",
		},
	}

	if !resp.GetBool("success") {
		t.Error("GetBool failed")
	}

	if resp.GetString("error") != "test error" {
		t.Error("GetString failed")
	}

	if resp.Get("data") != "test data" {
		t.Error("Get failed")
	}

	// Test RPCError
	err := &RPCError{
		Code:    -32601,
		Message: "Method not found",
	}

	if err.Code != -32601 {
		t.Error("RPCError.Code failed")
	}

	if err.Message != "Method not found" {
		t.Error("RPCError.Message failed")
	}
}

func TestPermissionChecker(t *testing.T) {
	_ = NewPermissionChecker("restricted")

	// Test file access patterns
	patterns := []struct {
		path     string
		pattern  string
		expected bool
	}{
		{"/tmp/test.txt", "/tmp/**", true},
		{"/tmp/subdir/test.txt", "/tmp/**", true},
		{"/var/tmp/test.txt", "/tmp/**", false},
	}

	for _, tt := range patterns {
		result := matchGlob(tt.path, tt.pattern)
		if result != tt.expected {
			t.Errorf("matchGlob(%s, %s) = %v, want %v", tt.path, tt.pattern, result, tt.expected)
		}
	}

	// Test $HOME expansion separately
	home := os.Getenv("HOME")
	if home != "" {
		result := matchGlob(home+"/user/test.txt", "$HOME/**")
		if !result {
			t.Errorf("matchGlob(%s/user/test.txt, $HOME/**) = false, want true", home)
		}
	}
}

func TestSkillManifest(t *testing.T) {
	manifest := &SkillManifest{
		APIVersion: "swarm.ai/v1",
		Kind:       "Skill",
		Metadata: SkillMetadata{
			Name:        "test-skill",
			Version:     "1.0.0",
			Description: "Test skill",
			Author:      "Test Author",
			License:     "MIT",
			Tags:        []string{"test", "example"},
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "/bin/echo",
			Tools: []ToolDef{
				{
					Name:        "echo",
					Description: "Echoes input",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"text": map[string]string{"type": "string"},
						},
					},
				},
			},
			Permissions: SkillPermissions{
				Filesystem: FilesystemPermissions{
					Read:  []string{"**"},
					Write: []string{"/tmp/**"},
				},
			},
		},
	}

	// Verify manifest structure
	if manifest.Metadata.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", manifest.Metadata.Name)
	}

	if len(manifest.Spec.Tools) != 1 {
		t.Fatalf("Expected 1 tool, got %d", len(manifest.Spec.Tools))
	}

	if manifest.Spec.Runtime != "go" {
		t.Errorf("Expected runtime 'go', got '%s'", manifest.Spec.Runtime)
	}
}

func TestMultiRuntimeLoaderLoad(t *testing.T) {
	loader := NewMultiRuntimeLoader()

	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "test-skill",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "native",
			Entrypoint: "/bin/echo",
			Tools: []ToolDef{
				{
					Name:        "echo",
					Description: "Echoes input",
				},
			},
		},
	}

	// Test loading
	inst, err := loader.Load(context.Background(), manifest)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if inst == nil {
		t.Fatal("Instance is nil")
	}

	// Test IsLoaded
	if !loader.IsLoaded("test-skill") {
		t.Error("Skill not marked as loaded")
	}

	// Test loading again (should return cached instance)
	inst2, err := loader.Load(context.Background(), manifest)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if inst != inst2 {
		t.Error("Load returned different instance for cached skill")
	}

	// Test ListLoaded
	loaded := loader.ListLoaded()
	if len(loaded) != 1 {
		t.Fatalf("Expected 1 loaded skill, got %d", len(loaded))
	}

	if loaded[0] != "test-skill" {
		t.Errorf("Expected 'test-skill', got '%s'", loaded[0])
	}

	// Test GetLoadedInstance
	gotInst, ok := loader.GetLoadedInstance("test-skill")
	if !ok {
		t.Error("GetLoadedInstance returned false")
	}

	if gotInst != inst {
		t.Error("GetLoadedInstance returned wrong instance")
	}

	// Test Unload
	if err := loader.Unload(context.Background(), "test-skill"); err != nil {
		t.Errorf("Unload failed: %v", err)
	}

	// Verify unloaded
	if loader.IsLoaded("test-skill") {
		t.Error("Skill still marked as loaded")
	}
}

func TestMultiRuntimeLoaderUnsupportedRuntime(t *testing.T) {
	loader := NewMultiRuntimeLoader()

	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "test-skill",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "unsupported",
			Entrypoint: "/bin/echo",
		},
	}

	_, err := loader.Load(context.Background(), manifest)
	if err == nil {
		t.Error("Expected error for unsupported runtime")
	}
}

func BenchmarkGoLoaderLoad(b *testing.B) {
	l := &GoLoader{buildDir: b.TempDir()}
	manifest := &SkillManifest{
		Metadata: SkillMetadata{
			Name:    "bench-skill",
			Version: "1.0.0",
		},
		Spec: SkillSpec{
			Runtime:    "go",
			Entrypoint: "/bin/echo",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := l.Load(context.Background(), manifest)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSandboxCheckFileAccess(b *testing.B) {
	sandbox := NewSandbox(&SandboxConfig{
		Enabled: true,
		Profile: "standard",
		TempDir: b.TempDir(),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sandbox.CheckFileAccess("/tmp/test.txt", "read")
	}
}
