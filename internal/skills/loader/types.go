package loader

import (
	"time"
)

// SkillManifest represents a loaded skill manifest
type SkillManifest struct {
	APIVersion string
	Kind       string
	Metadata   SkillMetadata
	Spec       SkillSpec
	Source     string
	FilePath   string
}

// SkillMetadata contains skill metadata
type SkillMetadata struct {
	Name        string
	Version     string
	DisplayName string
	Description string
	Author      string
	License     string
	Tags        []string
	Icon        string
	Homepage    string
	Repository  string
	Deprecated  bool
}

// SkillSpec contains skill specifications
type SkillSpec struct {
	Runtime      string
	Entrypoint   string
	Triggers     []SkillTrigger
	Tools        []ToolDef
	Prompts      SkillPrompts
	Dependencies []Dependency
	Resources    ResourceLimits
	Permissions  SkillPermissions
	Composition  SkillComposition
	Config       SkillConfig
}

// SkillTrigger defines when a skill should be activated
type SkillTrigger struct {
	Type       string
	Patterns   []string
	Confidence float64
	Events     []string
}

// ToolDef defines a tool exposed by the skill
type ToolDef struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Returns     map[string]interface{}
}

// SkillPrompts contains prompt templates
type SkillPrompts struct {
	System           string
	Examples         []map[string]interface{}
	ContextInjection ContextInjection
}

// ContextInjection defines where to inject context
type ContextInjection struct {
	Before string
	After  string
}

// Dependency defines a skill or package dependency
type Dependency struct {
	Name     string
	Version  string
	Source   string
	Optional bool
	Skill    string
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	CPU         string
	Memory      string
	Timeout     time.Duration
	TempStorage string
}

// SkillPermissions defines access permissions
type SkillPermissions struct {
	Filesystem  FilesystemPermissions
	Network     NetworkPermissions
	Environment EnvironmentPermissions
}

// FilesystemPermissions defines file access
type FilesystemPermissions struct {
	Read   []string
	Write  []string
	Delete []string
}

// NetworkPermissions defines network access
type NetworkPermissions struct {
	Allow        bool
	AllowedHosts []string
}

// EnvironmentPermissions defines environment variable access
type EnvironmentPermissions struct {
	Allow []string
}

// SkillComposition defines skill relationships
type SkillComposition struct {
	CompatibleWith []string
	ConflictsWith  []string
	Enhances       []Enhancement
}

// Enhancement defines a skill enhancement
type Enhancement struct {
	Skill      string
	MinVersion string
}

// SkillConfig defines configuration schema
type SkillConfig struct {
	Schema map[string]interface{}
	UI     map[string]interface{}
}

// SkillConfig contains runtime configuration for a skill
type Config struct {
	Settings  map[string]interface{}
	Workspace string
}

// SkillResult contains the result of skill execution
type Result struct {
	Success    bool
	Data       interface{}
	Error      string
	Duration   time.Duration
	TokensUsed int
}

// SkillMatch represents a search match result
type Match struct {
	SkillID string
	Score   float64
	Context map[string]interface{}
}

// RPCRequest represents a JSON-RPC request
type RPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// RPCResponse represents a JSON-RPC response
type RPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      string      `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// GetBool gets a boolean value from result
func (r *RPCResponse) GetBool(key string) bool {
	if m, ok := r.Result.(map[string]interface{}); ok {
		if val, ok := m[key].(bool); ok {
			return val
		}
	}
	return false
}

// GetString gets a string value from result
func (r *RPCResponse) GetString(key string) string {
	if m, ok := r.Result.(map[string]interface{}); ok {
		if val, ok := m[key].(string); ok {
			return val
		}
	}
	return ""
}

// Get gets any value from result
func (r *RPCResponse) Get(key string) interface{} {
	if m, ok := r.Result.(map[string]interface{}); ok {
		return m[key]
	}
	return nil
}
