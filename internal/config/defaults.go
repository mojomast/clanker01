package config

import "time"

const (
	DefaultVersion          = "1.0"
	DefaultProjectName      = "swarm-project"
	DefaultProjectRoot      = "."
	DefaultLLMProvider      = "anthropic"
	DefaultLLMModel         = "claude-sonnet-4-20250514"
	DefaultMaxTokens        = 200000
	DefaultMaxRetries       = 3
	DefaultTimeout          = 5 * time.Minute
	DefaultContextMaxTokens = 100000
	DefaultCompressionRatio = 0.3
	DefaultTopK             = 20
	DefaultTUITheme         = "swarm-dark"
	DefaultSplitRatio       = 0.4
	DefaultGRPCPort         = 50051
	DefaultHTTPPort         = 8080
	DefaultSandboxProfile   = "standard"
	DefaultSecretProvider   = "environment"
)

var DefaultConfig = &Config{
	Version: DefaultVersion,
	Project: ProjectConfig{
		Name: DefaultProjectName,
		Root: DefaultProjectRoot,
	},
	LLM: LLMConfig{
		DefaultProvider: DefaultLLMProvider,
		DefaultModel:    DefaultLLMModel,
		Providers: map[string]ProviderConfig{
			"anthropic": {
				Models: []ModelInfo{
					{
						ID:        "claude-sonnet-4-20250514",
						Alias:     "claude-sonnet",
						MaxTokens: DefaultMaxTokens,
					},
					{
						ID:        "claude-3-5-haiku-20241022",
						Alias:     "claude-haiku",
						MaxTokens: DefaultMaxTokens,
					},
				},
			},
		},
		AgentModelMapping: map[string]string{
			"architect": "claude-sonnet",
			"coder":     "claude-sonnet",
			"tester":    "claude-haiku",
			"reviewer":  "claude-haiku",
		},
	},
	MCP: MCPConfig{
		Servers: map[string]MCPServerConfig{
			"filesystem": {
				Type: "stdio",
				Cmd:  "mcp-server-filesystem",
			},
		},
	},
	Agents: AgentsConfig{
		Defaults: AgentDefaults{
			Timeout:    Duration{Duration: DefaultTimeout},
			MaxRetries: DefaultMaxRetries,
		},
		Roles: map[string]Role{
			"architect": {
				MinInstances: 1,
				MaxInstances: 3,
				Model:        "claude-sonnet",
			},
			"coder": {
				MinInstances: 2,
				MaxInstances: 10,
				Model:        "claude-sonnet",
			},
			"tester": {
				MinInstances: 2,
				MaxInstances: 5,
				Model:        "claude-haiku",
			},
			"reviewer": {
				MinInstances: 1,
				MaxInstances: 3,
				Model:        "claude-haiku",
			},
		},
	},
	Skills: SkillsConfig{
		Builtin: []string{"filesystem", "git", "shell", "test-runner"},
	},
	Context: ContextConfig{
		MaxTokens: DefaultContextMaxTokens,
		Compression: CompressionConfig{
			Enabled: true,
			Ratio:   DefaultCompressionRatio,
		},
		Retrieval: RetrievalConfig{
			VectorStore:    "qdrant",
			EmbeddingModel: "text-embedding-3-small",
			TopK:           DefaultTopK,
		},
	},
	TUI: TUIConfig{
		Theme: DefaultTUITheme,
		Layout: TUILayoutConfig{
			SplitRatio: DefaultSplitRatio,
		},
		Keybinds: map[string]string{
			"submit": "Ctrl+Enter",
			"cancel": "Esc",
			"help":   "?",
		},
	},
	Server: ServerConfig{
		Enabled: false,
		GRPC: GRPCConfig{
			Port: DefaultGRPCPort,
		},
		HTTP: HTTPConfig{
			Port: DefaultHTTPPort,
		},
		Auth: AuthConfig{
			Enabled: false,
		},
	},
	Security: SecurityConfig{
		Sandbox: SandboxConfig{
			Enabled: true,
			Profile: DefaultSandboxProfile,
		},
		Audit: AuditConfig{
			Enabled: true,
			Path:    "~/.swarm/logs/audit.jsonl",
		},
		Secrets: SecretsConfig{
			Provider: DefaultSecretProvider,
		},
	},
}

func GetDefaults() *Config {
	return DefaultConfig
}
