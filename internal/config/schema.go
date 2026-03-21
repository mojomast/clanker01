package config

import (
	"encoding/json"
	"fmt"
	"time"
)

type Config struct {
	Version  string         `yaml:"version" json:"version"`
	Project  ProjectConfig  `yaml:"project" json:"project"`
	LLM      LLMConfig      `yaml:"llm" json:"llm"`
	MCP      MCPConfig      `yaml:"mcp" json:"mcp"`
	Agents   AgentsConfig   `yaml:"agents" json:"agents"`
	Skills   SkillsConfig   `yaml:"skills" json:"skills"`
	Context  ContextConfig  `yaml:"context" json:"context"`
	TUI      TUIConfig      `yaml:"tui" json:"tui"`
	Server   ServerConfig   `yaml:"server" json:"server"`
	Security SecurityConfig `yaml:"security" json:"security"`
}

type ProjectConfig struct {
	Name string `yaml:"name" json:"name" validate:"required"`
	Root string `yaml:"root" json:"root" validate:"required"`
}

type LLMConfig struct {
	DefaultProvider   string                    `yaml:"default_provider" json:"default_provider" validate:"required"`
	DefaultModel      string                    `yaml:"default_model" json:"default_model" validate:"required"`
	Providers         map[string]ProviderConfig `yaml:"providers" json:"providers"`
	AgentModelMapping map[string]string         `yaml:"agent_model_mapping" json:"agent_model_mapping"`
}

type ProviderConfig struct {
	APIKey  string         `yaml:"api_key" json:"api_key" validate:"required_without=BaseURL"`
	BaseURL string         `yaml:"base_url" json:"base_url"`
	Models  []ModelInfo    `yaml:"models" json:"models"`
	Options map[string]any `yaml:"options,omitempty" json:"options,omitempty"`
}

type ModelInfo struct {
	ID        string `yaml:"id" json:"id" validate:"required"`
	Alias     string `yaml:"alias" json:"alias"`
	MaxTokens int    `yaml:"max_tokens" json:"max_tokens" validate:"required,min=1"`
}

type MCPConfig struct {
	Servers map[string]MCPServerConfig `yaml:"servers" json:"servers"`
}

type MCPServerConfig struct {
	Type   string            `yaml:"type" json:"type" validate:"required,oneof=stdio http websocket"`
	Cmd    string            `yaml:"command" json:"command" validate:"required_if=Type stdio"`
	Args   []string          `yaml:"args" json:"args"`
	Env    map[string]string `yaml:"env" json:"env"`
	URL    string            `yaml:"url" json:"url" validate:"required_if=Type http,required_if=Type websocket"`
	Stdin  string            `yaml:"stdin" json:"stdin"`
	Stdout string            `yaml:"stdout" json:"stdout"`
}

type AgentsConfig struct {
	Defaults AgentDefaults   `yaml:"defaults" json:"defaults"`
	Roles    map[string]Role `yaml:"roles" json:"roles"`
}

type AgentDefaults struct {
	Timeout    Duration `yaml:"timeout" json:"timeout"`
	MaxRetries int      `yaml:"max_retries" json:"max_retries"`
}

type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch value := v.(type) {
	case float64:
		d.Duration = time.Duration(value)
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported duration type: %T", v)
	}
	return nil
}

func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var v interface{}
	if err := unmarshal(&v); err != nil {
		return err
	}
	switch value := v.(type) {
	case int:
		d.Duration = time.Duration(value)
	case float64:
		d.Duration = time.Duration(value)
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("unsupported duration type: %T", v)
	}
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Duration.String())
}

func (d Duration) MarshalYAML() (interface{}, error) {
	return d.Duration.String(), nil
}

type Role struct {
	MinInstances int    `yaml:"min_instances" json:"min_instances" validate:"min=0"`
	MaxInstances int    `yaml:"max_instances" json:"max_instances" validate:"gtefield=MinInstances"`
	Model        string `yaml:"model" json:"model"`
}

type SkillsConfig struct {
	Builtin  []string        `yaml:"builtin" json:"builtin"`
	External []ExternalSkill `yaml:"external" json:"external"`
}

type ExternalSkill struct {
	Name    string            `yaml:"name" json:"name" validate:"required"`
	Version string            `yaml:"version" json:"version"`
	Config  map[string]string `yaml:"config" json:"config"`
}

type ContextConfig struct {
	MaxTokens   int               `yaml:"max_tokens" json:"max_tokens"`
	Compression CompressionConfig `yaml:"compression" json:"compression"`
	Retrieval   RetrievalConfig   `yaml:"retrieval" json:"retrieval"`
}

type CompressionConfig struct {
	Enabled bool    `yaml:"enabled" json:"enabled"`
	Ratio   float64 `yaml:"ratio" json:"ratio" validate:"gte=0,lte=1"`
}

type RetrievalConfig struct {
	VectorStore    string `yaml:"vector_store" json:"vector_store"`
	EmbeddingModel string `yaml:"embedding_model" json:"embedding_model"`
	TopK           int    `yaml:"top_k" json:"top_k" validate:"min=1"`
}

type TUIConfig struct {
	Theme    string            `yaml:"theme" json:"theme" validate:"required"`
	Layout   TUILayoutConfig   `yaml:"layout" json:"layout"`
	Keybinds map[string]string `yaml:"keybinds" json:"keybinds"`
}

type TUILayoutConfig struct {
	SplitRatio float64 `yaml:"split_ratio" json:"split_ratio" validate:"gte=0,lte=1"`
}

type ServerConfig struct {
	Enabled bool       `yaml:"enabled" json:"enabled"`
	GRPC    GRPCConfig `yaml:"grpc" json:"grpc"`
	HTTP    HTTPConfig `yaml:"http" json:"http"`
	Auth    AuthConfig `yaml:"auth" json:"auth"`
}

type GRPCConfig struct {
	Port int `yaml:"port" json:"port" validate:"min=1,max=65535"`
}

type HTTPConfig struct {
	Port int `yaml:"port" json:"port" validate:"min=1,max=65535"`
}

type AuthConfig struct {
	Enabled   bool   `yaml:"enabled" json:"enabled"`
	JWTSecret string `yaml:"jwt_secret" json:"jwt_secret" validate:"required_if=Enabled true"`
}

type SecurityConfig struct {
	Sandbox SandboxConfig `yaml:"sandbox" json:"sandbox"`
	Audit   AuditConfig   `yaml:"audit" json:"audit"`
	Secrets SecretsConfig `yaml:"secrets" json:"secrets"`
}

type SandboxConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Profile string `yaml:"profile" json:"profile" validate:"required_if=Enabled true,oneof=standard strict permissive"`
}

type AuditConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	Path    string `yaml:"path" json:"path" validate:"required_if=Enabled true"`
}

type SecretsConfig struct {
	Provider string `yaml:"provider" json:"provider" validate:"oneof=environment vault file"`
}
