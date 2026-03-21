package loader

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Sandbox manages skill execution sandbox
type Sandbox struct {
	config    *SandboxConfig
	checker   *PermissionChecker
	mu        sync.RWMutex
	processes map[int]*os.Process
}

// SandboxConfig defines sandbox configuration
type SandboxConfig struct {
	Enabled       bool
	Profile       string // restricted | standard | elevated
	MaxMemoryMB   int
	MaxCPUSeconds int
	MaxFileSizeMB int
	Timeout       time.Duration
	TempDir       string
}

// PermissionChecker enforces skill permissions
type PermissionChecker struct {
	policies []*SecurityPolicy
}

// SecurityPolicy defines security rules
type SecurityPolicy struct {
	Name          string
	DefaultAction string // allow | deny
	Filesystem    *FilesystemPolicy
	Network       *NetworkPolicy
	Environment   *EnvironmentPolicy
}

// FilesystemPolicy defines filesystem access rules
type FilesystemPolicy struct {
	ReadPaths   []string
	WritePaths  []string
	DeletePaths []string
	DenyPaths   []string
}

// NetworkPolicy defines network access rules
type NetworkPolicy struct {
	Allow        bool
	AllowedHosts []string
	AllowedPorts []int
}

// EnvironmentPolicy defines environment variable access rules
type EnvironmentPolicy struct {
	AllowList []string
	DenyList  []string
}

// NewSandbox creates a new sandbox
func NewSandbox(config *SandboxConfig) *Sandbox {
	if config == nil {
		config = &SandboxConfig{
			Enabled:       false,
			Profile:       "standard",
			MaxMemoryMB:   512,
			MaxCPUSeconds: 30,
			MaxFileSizeMB: 10,
			TempDir:       os.TempDir(),
		}
	}
	return &Sandbox{
		config:    config,
		checker:   NewPermissionChecker(config.Profile),
		processes: make(map[int]*os.Process),
	}
}

// Run executes a command within the sandbox
func (s *Sandbox) Run(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
	if !s.config.Enabled {
		return cmd.CombinedOutput()
	}

	// Apply resource limits
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
		Setpgid:    true,
	}

	// Set up temp directory
	if s.config.TempDir != "" {
		tmp, err := os.MkdirTemp(s.config.TempDir, "skill-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(tmp)
		cmd.Dir = tmp
		cmd.Env = append(os.Environ(), "TMPDIR="+tmp)
	}

	// Timeout
	var cancel context.CancelFunc
	if s.config.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		cmd.Dir = cmd.Dir
		cmd.Env = cmd.Env
		cmd.Stdin = cmd.Stdin
		cmd.Stdout = cmd.Stdout
		cmd.Stderr = cmd.Stderr
		cmd.SysProcAttr = cmd.SysProcAttr
	}

	// Track process
	if cmd.Process != nil {
		s.mu.Lock()
		s.processes[cmd.Process.Pid] = cmd.Process
		s.mu.Unlock()
		defer func() {
			s.mu.Lock()
			delete(s.processes, cmd.Process.Pid)
			s.mu.Unlock()
		}()
	}

	return cmd.CombinedOutput()
}

// StartProcess starts a process in the sandbox
func (s *Sandbox) StartProcess(ctx context.Context, cmd *exec.Cmd) (*os.Process, io.WriteCloser, io.ReadCloser, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, nil, nil, err
	}

	// Configure sandbox if enabled
	if s.config.Enabled {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
			Setpgid:    true,
		}

		if s.config.TempDir != "" {
			tmp, err := os.MkdirTemp(s.config.TempDir, "skill-*")
			if err != nil {
				stdin.Close()
				stdout.Close()
				stderr.Close()
				return nil, nil, nil, err
			}
			cmd.Dir = tmp
			cmd.Env = append(os.Environ(), "TMPDIR="+tmp)
		}

		// Apply resource limits via ulimit
		cmd.Env = append(cmd.Env, fmt.Sprintf("SKILL_MAX_MEMORY=%d", s.config.MaxMemoryMB))
		cmd.Env = append(cmd.Env, fmt.Sprintf("SKILL_MAX_CPU=%d", s.config.MaxCPUSeconds))
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, nil, nil, err
	}

	// Track process
	s.mu.Lock()
	s.processes[cmd.Process.Pid] = cmd.Process
	s.mu.Unlock()

	// Drain stderr to avoid blocking
	go func() {
		io.Copy(io.Discard, stderr)
		stderr.Close()
	}()

	return cmd.Process, stdin, stdout, nil
}

// StopProcess stops a process
func (s *Sandbox) StopProcess(pid int) error {
	s.mu.Lock()
	proc, ok := s.processes[pid]
	if ok {
		delete(s.processes, pid)
	}
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("process not found: %d", pid)
	}

	// Try graceful shutdown first
	proc.Signal(syscall.SIGTERM)

	// Kill after 5 seconds
	time.AfterFunc(5*time.Second, func() {
		proc.Kill()
	})

	return nil
}

// CheckFileAccess checks if file access is allowed
func (s *Sandbox) CheckFileAccess(path string, mode string) bool {
	return s.checker.CheckFileAccess(path, mode)
}

// CheckNetworkAccess checks if network access is allowed
func (s *Sandbox) CheckNetworkAccess(host string) bool {
	return s.checker.CheckNetworkAccess(host)
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(profile string) *PermissionChecker {
	c := &PermissionChecker{
		policies: make([]*SecurityPolicy, 0),
	}

	// Add default policy based on profile
	policy := &SecurityPolicy{
		Name:          "default",
		DefaultAction: "deny",
	}

	switch profile {
	case "restricted":
		policy.Filesystem = &FilesystemPolicy{
			ReadPaths:   []string{"/tmp/**", "/var/tmp/**"},
			WritePaths:  []string{"/tmp/**", "/var/tmp/**"},
			DeletePaths: []string{},
			DenyPaths:   []string{"/home/**", "/etc/**", "/root/**", "/usr/**"},
		}
		policy.Network = &NetworkPolicy{
			Allow:        false,
			AllowedHosts: []string{},
		}
	case "standard":
		policy.DefaultAction = "allow"
		policy.Filesystem = &FilesystemPolicy{
			ReadPaths:   []string{"**"},
			WritePaths:  []string{"/tmp/**", "/var/tmp/**", "$HOME/**"},
			DeletePaths: []string{"/tmp/**", "/var/tmp/**"},
			DenyPaths:   []string{"/etc/**", "/root/**", "/usr/bin/**"},
		}
		policy.Network = &NetworkPolicy{
			Allow:        true,
			AllowedHosts: []string{},
		}
	case "elevated":
		policy.DefaultAction = "allow"
		policy.Filesystem = &FilesystemPolicy{
			ReadPaths:   []string{"**"},
			WritePaths:  []string{"**"},
			DeletePaths: []string{"/tmp/**", "/var/tmp/**"},
			DenyPaths:   []string{"/boot/**", "/dev/**"},
		}
		policy.Network = &NetworkPolicy{
			Allow:        true,
			AllowedHosts: []string{"*"},
		}
	}

	c.policies = append(c.policies, policy)
	return c
}

// CheckFileAccess checks if file access is allowed
func (c *PermissionChecker) CheckFileAccess(path string, mode string) bool {
	for _, policy := range c.policies {
		if policy.Filesystem == nil {
			continue
		}

		// Check deny list first
		for _, deny := range policy.Filesystem.DenyPaths {
			if matchGlob(path, deny) {
				return false
			}
		}

		var allowed []string
		switch mode {
		case "read":
			allowed = policy.Filesystem.ReadPaths
		case "write":
			allowed = policy.Filesystem.WritePaths
		case "delete":
			allowed = policy.Filesystem.DeletePaths
		}

		for _, allow := range allowed {
			if matchGlob(path, allow) {
				return true
			}
		}

		if policy.DefaultAction == "deny" {
			return false
		}
	}
	return true
}

// CheckNetworkAccess checks if network access is allowed
func (c *PermissionChecker) CheckNetworkAccess(host string) bool {
	for _, policy := range c.policies {
		if policy.Network == nil {
			continue
		}
		if !policy.Network.Allow {
			return false
		}
		// If no specific hosts are listed, allow all
		if len(policy.Network.AllowedHosts) == 0 {
			return true
		}
		for _, allowed := range policy.Network.AllowedHosts {
			if matchGlob(host, allowed) {
				return true
			}
		}
	}
	return false
}

// matchGlob matches a path against a glob pattern
func matchGlob(path, pattern string) bool {
	if pattern == "**" || pattern == "*/*" {
		return true
	}

	// Expand environment variables in pattern
	pattern = os.ExpandEnv(pattern)

	// Handle ** pattern
	if pattern == "**" {
		return true
	}

	// Handle pattern ending with /**
	if len(pattern) >= 3 && pattern[len(pattern)-3:] == "/**" {
		prefix := pattern[:len(pattern)-3]
		return strings.HasPrefix(path, prefix) || strings.HasPrefix(path+"/", prefix+"/")
	}

	// Handle pattern starting with /**
	if len(pattern) >= 3 && pattern[:3] == "/**" {
		suffix := pattern[3:]
		if suffix == "" || suffix == "/" {
			return true
		}
		return strings.HasSuffix(path, suffix) || strings.HasSuffix(path, suffix+"/") || strings.Contains(path, suffix+"/")
	}

	// Handle pattern with ** in the middle
	if strings.Contains(pattern, "/**/") {
		parts := strings.Split(pattern, "/**/")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]
			if prefix == "" {
				return strings.HasSuffix(path, suffix) || strings.Contains(path, "/"+suffix)
			}
			if suffix == "" {
				return strings.HasPrefix(path, prefix) || strings.HasPrefix(path, prefix+"/")
			}
			return (strings.HasPrefix(path, prefix) || strings.HasPrefix(path, prefix+"/")) &&
				(strings.HasSuffix(path, suffix) || strings.Contains(path, "/"+suffix))
		}
	}

	// Use filepath.Match for simple patterns
	matched, err := filepath.Match(pattern, path)
	if err == nil {
		return matched
	}

	return false
}

func pathHasSuffix(path, suffix string) bool {
	if suffix == "" {
		return true
	}
	if suffix[0] == '/' {
		suffix = suffix[1:]
	}
	if suffix == "" {
		return true
	}

	pathParts := splitPath(path)
	suffixParts := splitPath(suffix)

	if len(suffixParts) > len(pathParts) {
		return false
	}

	for i := range suffixParts {
		if suffixParts[len(suffixParts)-1-i] != pathParts[len(pathParts)-1-i] {
			return false
		}
	}

	return true
}

func splitPath(path string) []string {
	parts := filepath.SplitList(path)
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" && part != string(filepath.Separator) {
			result = append(result, part)
		}
	}
	return result
}
