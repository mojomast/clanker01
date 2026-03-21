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

// applyDefaults fills in zero-value fields in the given config from DefaultConfig.
func applyDefaults(config *Config) {
	if config.Version == "" {
		config.Version = DefaultConfig.Version
	}
	if config.Project.Name == "" {
		config.Project.Name = DefaultConfig.Project.Name
	}
	if config.Project.Root == "" {
		config.Project.Root = DefaultConfig.Project.Root
	}
	if config.LLM.DefaultProvider == "" {
		config.LLM.DefaultProvider = DefaultConfig.LLM.DefaultProvider
	}
	if config.LLM.DefaultModel == "" {
		config.LLM.DefaultModel = DefaultConfig.LLM.DefaultModel
	}
	if config.LLM.Providers == nil {
		config.LLM.Providers = DefaultConfig.LLM.Providers
	}
	if config.LLM.AgentModelMapping == nil {
		config.LLM.AgentModelMapping = DefaultConfig.LLM.AgentModelMapping
	}
	if config.MCP.Servers == nil {
		config.MCP.Servers = DefaultConfig.MCP.Servers
	}
	if config.Agents.Defaults.Timeout.Duration == 0 {
		config.Agents.Defaults.Timeout = DefaultConfig.Agents.Defaults.Timeout
	}
	if config.Agents.Defaults.MaxRetries == 0 {
		config.Agents.Defaults.MaxRetries = DefaultConfig.Agents.Defaults.MaxRetries
	}
	if config.Agents.Roles == nil {
		config.Agents.Roles = DefaultConfig.Agents.Roles
	}
	if config.Context.MaxTokens == 0 {
		config.Context.MaxTokens = DefaultConfig.Context.MaxTokens
	}
	if config.Context.Compression.Ratio == 0 {
		config.Context.Compression.Ratio = DefaultConfig.Context.Compression.Ratio
	}
	if config.Context.Retrieval.TopK == 0 {
		config.Context.Retrieval.TopK = DefaultConfig.Context.Retrieval.TopK
	}
	if config.Context.Retrieval.VectorStore == "" {
		config.Context.Retrieval.VectorStore = DefaultConfig.Context.Retrieval.VectorStore
	}
	if config.Context.Retrieval.EmbeddingModel == "" {
		config.Context.Retrieval.EmbeddingModel = DefaultConfig.Context.Retrieval.EmbeddingModel
	}
	if config.TUI.Theme == "" {
		config.TUI.Theme = DefaultConfig.TUI.Theme
	}
	if config.TUI.Layout.SplitRatio == 0 {
		config.TUI.Layout.SplitRatio = DefaultConfig.TUI.Layout.SplitRatio
	}
	if config.TUI.Keybinds == nil {
		config.TUI.Keybinds = DefaultConfig.TUI.Keybinds
	}
	if config.Server.GRPC.Port == 0 {
		config.Server.GRPC.Port = DefaultConfig.Server.GRPC.Port
	}
	if config.Server.HTTP.Port == 0 {
		config.Server.HTTP.Port = DefaultConfig.Server.HTTP.Port
	}
	if config.Security.Sandbox.Profile == "" {
		config.Security.Sandbox.Profile = DefaultConfig.Security.Sandbox.Profile
	}
	if config.Security.Audit.Path == "" {
		config.Security.Audit.Path = DefaultConfig.Security.Audit.Path
	}
	if config.Security.Secrets.Provider == "" {
		config.Security.Secrets.Provider = DefaultConfig.Security.Secrets.Provider
	}
}
