package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	defaults := GetDefaults()

	require.NotNil(t, defaults)
	assert.Equal(t, "1.0", defaults.Version)
	assert.Equal(t, "swarm-project", defaults.Project.Name)
	assert.Equal(t, ".", defaults.Project.Root)
	assert.Equal(t, "anthropic", defaults.LLM.DefaultProvider)
	assert.Equal(t, "claude-sonnet-4-20250514", defaults.LLM.DefaultModel)
}

func TestLoadDefault(t *testing.T) {
	testConfig := `
version: "1.0"
project:
  name: test-project
  root: /tmp/test
llm:
  default_provider: anthropic
  default_model: claude-sonnet-4-20250514
  providers:
    anthropic:
      api_key: test-key
      models:
        - id: claude-sonnet-4-20250514
          alias: claude-sonnet
          max_tokens: 200000
mcp:
  servers: {}
agents:
  defaults:
    timeout: 300s
    max_retries: 3
  roles: {}
skills:
  builtin: []
  external: []
context:
  max_tokens: 100000
  compression:
    enabled: true
    ratio: 0.3
  retrieval:
    vector_store: qdrant
    embedding_model: text-embedding-3-small
    top_k: 20
tui:
  theme: swarm-dark
  layout:
    split_ratio: 0.4
  keybinds: {}
server:
  enabled: false
  grpc:
    port: 50051
  http:
    port: 8080
  auth:
    enabled: false
security:
  sandbox:
    enabled: true
    profile: standard
  audit:
    enabled: true
    path: ~/.swarm/logs/audit.jsonl
  secrets:
    provider: environment
`

	config, err := LoadFromBytes([]byte(testConfig), "yaml")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "1.0", config.Version)
	assert.Equal(t, "test-project", config.Project.Name)
	assert.Equal(t, "/tmp/test", config.Project.Root)
	assert.Equal(t, "anthropic", config.LLM.DefaultProvider)
	assert.Equal(t, "claude-sonnet-4-20250514", config.LLM.DefaultModel)
}

func TestLoadFromBytesYAML(t *testing.T) {
	yamlConfig := `
version: "2.0"
project:
  name: yaml-test
  root: .
llm:
  default_provider: openai
  default_model: gpt-4o
  providers:
    openai:
      api_key: openai-key
      models:
        - id: gpt-4o
          max_tokens: 128000
mcp:
  servers: {}
agents:
  defaults:
    timeout: 300s
    max_retries: 3
  roles: {}
skills:
  builtin: []
  external: []
context:
  max_tokens: 100000
  compression:
    enabled: true
    ratio: 0.3
  retrieval:
    vector_store: qdrant
    embedding_model: text-embedding-3-small
    top_k: 20
tui:
  theme: swarm-dark
  layout:
    split_ratio: 0.4
  keybinds: {}
server:
  enabled: false
  grpc:
    port: 50051
  http:
    port: 8080
  auth:
    enabled: false
security:
  sandbox:
    enabled: true
    profile: standard
  audit:
    enabled: true
    path: ~/.swarm/logs/audit.jsonl
  secrets:
    provider: environment
`

	config, err := LoadFromBytes([]byte(yamlConfig), "yaml")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "2.0", config.Version)
	assert.Equal(t, "yaml-test", config.Project.Name)
	assert.Equal(t, "openai", config.LLM.DefaultProvider)
	assert.Equal(t, "gpt-4o", config.LLM.DefaultModel)
}

func TestLoadFromBytesJSON(t *testing.T) {
	jsonConfig := `{
  "version": "3.0",
  "project": {
    "name": "json-test",
    "root": "."
  },
  "llm": {
    "default_provider": "ollama",
    "default_model": "llama3.2",
    "providers": {
      "ollama": {
        "base_url": "http://localhost:11434",
        "models": [
          {
            "id": "llama3.2",
            "max_tokens": 128000
          }
        ]
      }
    }
  },
  "mcp": {
    "servers": {}
  },
  "agents": {
    "defaults": {
      "timeout": "300s",
      "max_retries": 3
    },
    "roles": {}
  },
  "skills": {
    "builtin": [],
    "external": []
  },
  "context": {
    "max_tokens": 100000,
    "compression": {
      "enabled": true,
      "ratio": 0.3
    },
    "retrieval": {
      "vector_store": "qdrant",
      "embedding_model": "text-embedding-3-small",
      "top_k": 20
    }
  },
  "tui": {
    "theme": "swarm-dark",
    "layout": {
      "split_ratio": 0.4
    },
    "keybinds": {}
  },
  "server": {
    "enabled": false,
    "grpc": {
      "port": 50051
    },
    "http": {
      "port": 8080
    },
    "auth": {
      "enabled": false
    }
  },
  "security": {
    "sandbox": {
      "enabled": true,
      "profile": "standard"
    },
    "audit": {
      "enabled": true,
      "path": "~/.swarm/logs/audit.jsonl"
    },
    "secrets": {
      "provider": "environment"
    }
  }
}`

	config, err := LoadFromBytes([]byte(jsonConfig), "json")
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, "3.0", config.Version)
	assert.Equal(t, "json-test", config.Project.Name)
	assert.Equal(t, "ollama", config.LLM.DefaultProvider)
	assert.Equal(t, "llama3.2", config.LLM.DefaultModel)
	assert.Equal(t, "http://localhost:11434", config.LLM.Providers["ollama"].BaseURL)
}

func TestLoadFromBytesUnsupportedFormat(t *testing.T) {
	_, err := LoadFromBytes([]byte("test"), "xml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestValidateValidConfig(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Name: "test",
			Root: "/tmp",
		},
		LLM: LLMConfig{
			DefaultProvider: "anthropic",
			DefaultModel:    "claude-sonnet-4-20250514",
			Providers: map[string]ProviderConfig{
				"anthropic": {
					APIKey: "test-key",
					Models: []ModelInfo{
						{
							ID:        "claude-sonnet-4-20250514",
							MaxTokens: 200000,
						},
					},
				},
			},
		},
		MCP: MCPConfig{
			Servers: map[string]MCPServerConfig{},
		},
		Agents: AgentsConfig{
			Defaults: AgentDefaults{
				Timeout:    Duration{Duration: 300 * time.Second},
				MaxRetries: 3,
			},
			Roles: map[string]Role{},
		},
		Skills: SkillsConfig{
			Builtin:  []string{"filesystem"},
			External: []ExternalSkill{},
		},
		Context: ContextConfig{
			MaxTokens: 100000,
			Compression: CompressionConfig{
				Enabled: true,
				Ratio:   0.3,
			},
			Retrieval: RetrievalConfig{
				VectorStore:    "qdrant",
				EmbeddingModel: "text-embedding-3-small",
				TopK:           20,
			},
		},
		TUI: TUIConfig{
			Theme: "swarm-dark",
			Layout: TUILayoutConfig{
				SplitRatio: 0.4,
			},
			Keybinds: map[string]string{},
		},
		Server: ServerConfig{
			Enabled: false,
			GRPC:    GRPCConfig{Port: 50051},
			HTTP:    HTTPConfig{Port: 8080},
			Auth:    AuthConfig{Enabled: false},
		},
		Security: SecurityConfig{
			Sandbox: SandboxConfig{
				Enabled: true,
				Profile: "standard",
			},
			Audit: AuditConfig{
				Enabled: true,
				Path:    "/tmp/audit.jsonl",
			},
			Secrets: SecretsConfig{
				Provider: "environment",
			},
		},
	}

	err := Validate(config)
	assert.NoError(t, err)
}

func TestValidateMissingVersion(t *testing.T) {
	config := &Config{
		Version: "",
		Project: ProjectConfig{
			Name: "test",
			Root: "/tmp",
		},
		LLM: LLMConfig{
			DefaultProvider: "anthropic",
			DefaultModel:    "claude-sonnet-4-20250514",
			Providers: map[string]ProviderConfig{
				"anthropic": {
					APIKey: "test-key",
					Models: []ModelInfo{
						{
							ID:        "claude-sonnet-4-20250514",
							MaxTokens: 200000,
						},
					},
				},
			},
		},
		MCP:      MCPConfig{Servers: map[string]MCPServerConfig{}},
		Agents:   AgentsConfig{Defaults: AgentDefaults{Timeout: Duration{Duration: 300 * time.Second}, MaxRetries: 3}, Roles: map[string]Role{}},
		Skills:   SkillsConfig{Builtin: []string{}, External: []ExternalSkill{}},
		Context:  ContextConfig{MaxTokens: 100000, Compression: CompressionConfig{Enabled: true, Ratio: 0.3}, Retrieval: RetrievalConfig{VectorStore: "qdrant", EmbeddingModel: "text-embedding-3-small", TopK: 20}},
		TUI:      TUIConfig{Theme: "swarm-dark", Layout: TUILayoutConfig{SplitRatio: 0.4}, Keybinds: map[string]string{}},
		Server:   ServerConfig{Enabled: false, GRPC: GRPCConfig{Port: 50051}, HTTP: HTTPConfig{Port: 8080}, Auth: AuthConfig{Enabled: false}},
		Security: SecurityConfig{Sandbox: SandboxConfig{Enabled: true, Profile: "standard"}, Audit: AuditConfig{Enabled: true, Path: "/tmp/audit.jsonl"}, Secrets: SecretsConfig{Provider: "environment"}},
	}

	err := Validate(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "version")
}

func TestValidateMissingProjectName(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Project: ProjectConfig{
			Name: "",
			Root: "/tmp",
		},
		LLM: LLMConfig{
			DefaultProvider: "anthropic",
			DefaultModel:    "claude-sonnet-4-20250514",
			Providers: map[string]ProviderConfig{
				"anthropic": {
					APIKey: "test-key",
					Models: []ModelInfo{
						{
							ID:        "claude-sonnet-4-20250514",
							MaxTokens: 200000,
						},
					},
				},
			},
		},
		MCP:      MCPConfig{Servers: map[string]MCPServerConfig{}},
		Agents:   AgentsConfig{Defaults: AgentDefaults{Timeout: Duration{Duration: 300 * time.Second}, MaxRetries: 3}, Roles: map[string]Role{}},
		Skills:   SkillsConfig{Builtin: []string{}, External: []ExternalSkill{}},
		Context:  ContextConfig{MaxTokens: 100000, Compression: CompressionConfig{Enabled: true, Ratio: 0.3}, Retrieval: RetrievalConfig{VectorStore: "qdrant", EmbeddingModel: "text-embedding-3-small", TopK: 20}},
		TUI:      TUIConfig{Theme: "swarm-dark", Layout: TUILayoutConfig{SplitRatio: 0.4}, Keybinds: map[string]string{}},
		Server:   ServerConfig{Enabled: false, GRPC: GRPCConfig{Port: 50051}, HTTP: HTTPConfig{Port: 8080}, Auth: AuthConfig{Enabled: false}},
		Security: SecurityConfig{Sandbox: SandboxConfig{Enabled: true, Profile: "standard"}, Audit: AuditConfig{Enabled: true, Path: "/tmp/audit.jsonl"}, Secrets: SecretsConfig{Provider: "environment"}},
	}

	err := Validate(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project.name")
}

func TestValidateInvalidMaxTokens(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Project: ProjectConfig{Name: "test", Root: "/tmp"},
		LLM: LLMConfig{
			DefaultProvider: "anthropic",
			DefaultModel:    "claude-sonnet-4-20250514",
			Providers: map[string]ProviderConfig{
				"anthropic": {
					APIKey: "test-key",
					Models: []ModelInfo{
						{
							ID:        "claude-sonnet-4-20250514",
							MaxTokens: -1,
						},
					},
				},
			},
		},
		MCP:      MCPConfig{Servers: map[string]MCPServerConfig{}},
		Agents:   AgentsConfig{Defaults: AgentDefaults{Timeout: Duration{Duration: 300 * time.Second}, MaxRetries: 3}, Roles: map[string]Role{}},
		Skills:   SkillsConfig{Builtin: []string{}, External: []ExternalSkill{}},
		Context:  ContextConfig{MaxTokens: 100000, Compression: CompressionConfig{Enabled: true, Ratio: 0.3}, Retrieval: RetrievalConfig{VectorStore: "qdrant", EmbeddingModel: "text-embedding-3-small", TopK: 20}},
		TUI:      TUIConfig{Theme: "swarm-dark", Layout: TUILayoutConfig{SplitRatio: 0.4}, Keybinds: map[string]string{}},
		Server:   ServerConfig{Enabled: false, GRPC: GRPCConfig{Port: 50051}, HTTP: HTTPConfig{Port: 8080}, Auth: AuthConfig{Enabled: false}},
		Security: SecurityConfig{Sandbox: SandboxConfig{Enabled: true, Profile: "standard"}, Audit: AuditConfig{Enabled: true, Path: "/tmp/audit.jsonl"}, Secrets: SecretsConfig{Provider: "environment"}},
	}

	err := Validate(config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "max_tokens")
}

func TestGetProviderConfig(t *testing.T) {
	config := &Config{
		LLM: LLMConfig{
			Providers: map[string]ProviderConfig{
				"anthropic": {
					APIKey: "test-key",
					Models: []ModelInfo{
						{
							ID:        "claude-sonnet-4-20250514",
							MaxTokens: 200000,
						},
					},
				},
			},
		},
	}

	provider, err := GetProviderConfig(config, "anthropic")
	require.NoError(t, err)
	assert.Equal(t, "test-key", provider.APIKey)

	_, err = GetProviderConfig(config, "openai")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetModelConfig(t *testing.T) {
	config := &Config{
		LLM: LLMConfig{
			Providers: map[string]ProviderConfig{
				"anthropic": {
					Models: []ModelInfo{
						{
							ID:        "claude-sonnet-4-20250514",
							Alias:     "claude-sonnet",
							MaxTokens: 200000,
						},
					},
				},
			},
		},
	}

	model, err := GetModelConfig(config, "anthropic", "claude-sonnet-4-20250514")
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-20250514", model.ID)

	model, err = GetModelConfig(config, "anthropic", "claude-sonnet")
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-20250514", model.ID)

	_, err = GetModelConfig(config, "anthropic", "gpt-4")
	assert.Error(t, err)
}

func TestGetAgentModel(t *testing.T) {
	config := &Config{
		LLM: LLMConfig{
			DefaultModel: "claude-sonnet-4-20250514",
			AgentModelMapping: map[string]string{
				"architect": "claude-sonnet",
				"coder":     "claude-sonnet",
			},
		},
	}

	model, err := GetAgentModel(config, "architect")
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet", model)

	model, err = GetAgentModel(config, "reviewer")
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-20250514", model)
}

func TestGetProviderForModel(t *testing.T) {
	config := &Config{
		LLM: LLMConfig{
			Providers: map[string]ProviderConfig{
				"anthropic": {
					Models: []ModelInfo{
						{
							ID:        "claude-sonnet-4-20250514",
							Alias:     "claude-sonnet",
							MaxTokens: 200000,
						},
					},
				},
				"openai": {
					Models: []ModelInfo{
						{
							ID:        "gpt-4o",
							MaxTokens: 128000,
						},
					},
				},
			},
		},
	}

	provider, err := GetProviderForModel(config, "claude-sonnet-4-20250514")
	require.NoError(t, err)
	assert.Equal(t, "anthropic", provider)

	provider, err = GetProviderForModel(config, "claude-sonnet")
	require.NoError(t, err)
	assert.Equal(t, "anthropic", provider)

	provider, err = GetProviderForModel(config, "gpt-4o")
	require.NoError(t, err)
	assert.Equal(t, "openai", provider)

	_, err = GetProviderForModel(config, "unknown-model")
	assert.Error(t, err)
}

func TestManager(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "swarm.yaml")

	configContent := `
version: "1.0"
project:
  name: test-project
  root: /tmp/test
llm:
  default_provider: anthropic
  default_model: claude-sonnet-4-20250514
  providers:
    anthropic:
      api_key: test-key
      models:
        - id: claude-sonnet-4-20250514
          max_tokens: 200000
mcp:
  servers: {}
agents:
  defaults:
    timeout: 300s
    max_retries: 3
  roles: {}
skills:
  builtin: []
  external: []
context:
  max_tokens: 100000
  compression:
    enabled: true
    ratio: 0.3
  retrieval:
    vector_store: qdrant
    embedding_model: text-embedding-3-small
    top_k: 20
tui:
  theme: swarm-dark
  layout:
    split_ratio: 0.4
  keybinds: {}
server:
  enabled: false
  grpc:
    port: 50051
  http:
    port: 8080
  auth:
    enabled: false
security:
  sandbox:
    enabled: true
    profile: standard
  audit:
    enabled: true
    path: ~/.swarm/logs/audit.jsonl
  secrets:
    provider: environment
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	mgr, err := NewManager(&ManagerOptions{
		ConfigPath: configPath,
	})
	require.NoError(t, err)
	require.NotNil(t, mgr)

	defer mgr.Close()

	config := mgr.Get()
	assert.NotNil(t, config)
	assert.Equal(t, "1.0", config.Version)
	assert.Equal(t, "test-project", config.Project.Name)

	provider, err := mgr.GetProviderConfig("anthropic")
	require.NoError(t, err)
	assert.Equal(t, "test-key", provider.APIKey)

	model, err := mgr.GetModelConfig("anthropic", "claude-sonnet-4-20250514")
	require.NoError(t, err)
	assert.Equal(t, "claude-sonnet-4-20250514", model.ID)

	assert.Equal(t, "test-project", mgr.GetProjectName())
	assert.Equal(t, "/tmp/test", mgr.GetProjectRoot())
	assert.Equal(t, "anthropic", mgr.GetDefaultProvider())
	assert.Equal(t, "claude-sonnet-4-20250514", mgr.GetDefaultModel())
	assert.False(t, mgr.IsServerEnabled())
	assert.Equal(t, 50051, mgr.GetGRPCPort())
	assert.Equal(t, 8080, mgr.GetHTTPPort())
}

func TestManagerUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "swarm.yaml")

	configContent := `
version: "1.0"
project:
  name: test-project
  root: /tmp/test
llm:
  default_provider: anthropic
  default_model: claude-sonnet-4-20250514
  providers:
    anthropic:
      api_key: test-key
      models:
        - id: claude-sonnet-4-20250514
          max_tokens: 200000
mcp:
  servers: {}
agents:
  defaults:
    timeout: 300s
    max_retries: 3
  roles: {}
skills:
  builtin: []
  external: []
context:
  max_tokens: 100000
  compression:
    enabled: true
    ratio: 0.3
  retrieval:
    vector_store: qdrant
    embedding_model: text-embedding-3-small
    top_k: 20
tui:
  theme: swarm-dark
  layout:
    split_ratio: 0.4
  keybinds: {}
server:
  enabled: false
  grpc:
    port: 50051
  http:
    port: 8080
  auth:
    enabled: false
security:
  sandbox:
    enabled: true
    profile: standard
  audit:
    enabled: true
    path: ~/.swarm/logs/audit.jsonl
  secrets:
    provider: environment
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	mgr, err := NewManager(&ManagerOptions{
		ConfigPath: configPath,
	})
	require.NoError(t, err)

	defer mgr.Close()

	err = mgr.Update(func(cfg *Config) error {
		cfg.Project.Name = "updated-project"
		return nil
	})
	require.NoError(t, err)

	config := mgr.Get()
	assert.Equal(t, "updated-project", config.Project.Name)
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("SWARM_ANTHROPIC_API_KEY", "env-api-key")
	os.Setenv("PROJECT_ROOT", "/env/root")
	os.Setenv("JWT_SECRET", "env-jwt-secret")
	defer func() {
		os.Unsetenv("SWARM_ANTHROPIC_API_KEY")
		os.Unsetenv("PROJECT_ROOT")
		os.Unsetenv("JWT_SECRET")
	}()

	testConfig := `
version: "1.0"
project:
  name: test-project
  root: /tmp/test
llm:
  default_provider: anthropic
  default_model: claude-sonnet-4-20250514
  providers:
    anthropic:
      api_key: file-api-key
      models:
        - id: claude-sonnet-4-20250514
          max_tokens: 200000
mcp:
  servers: {}
agents:
  defaults:
    timeout: 300s
    max_retries: 3
  roles: {}
skills:
  builtin: []
  external: []
context:
  max_tokens: 100000
  compression:
    enabled: true
    ratio: 0.3
  retrieval:
    vector_store: qdrant
    embedding_model: text-embedding-3-small
    top_k: 20
tui:
  theme: swarm-dark
  layout:
    split_ratio: 0.4
  keybinds: {}
server:
  enabled: true
  grpc:
    port: 50051
  http:
    port: 8080
  auth:
    enabled: true
    jwt_secret: file-jwt-secret
security:
  sandbox:
    enabled: true
    profile: standard
  audit:
    enabled: true
    path: ~/.swarm/logs/audit.jsonl
  secrets:
    provider: environment
`

	config, err := LoadFromBytes([]byte(testConfig), "yaml")
	require.NoError(t, err)

	assert.Equal(t, "env-api-key", config.LLM.Providers["anthropic"].APIKey)
	assert.Equal(t, "/env/root", config.Project.Root)
	assert.Equal(t, "env-jwt-secret", config.Server.Auth.JWTSecret)
}

func TestValidateProviderConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  ProviderConfig
		wantErr bool
	}{
		{
			name: "valid provider",
			config: ProviderConfig{
				APIKey: "test-key",
				Models: []ModelInfo{
					{
						ID:        "claude-sonnet-4-20250514",
						MaxTokens: 200000,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid provider with base URL",
			config: ProviderConfig{
				BaseURL: "http://localhost:11434",
				Models: []ModelInfo{
					{
						ID:        "llama3.2",
						MaxTokens: 128000,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid - no API key or base URL",
			config: ProviderConfig{
				Models: []ModelInfo{
					{
						ID:        "claude-sonnet-4-20250514",
						MaxTokens: 200000,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid - no models",
			config: ProviderConfig{
				APIKey: "test-key",
				Models: []ModelInfo{},
			},
			wantErr: true,
		},
		{
			name: "invalid - model has no ID",
			config: ProviderConfig{
				APIKey: "test-key",
				Models: []ModelInfo{
					{
						MaxTokens: 200000,
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid - model has invalid max tokens",
			config: ProviderConfig{
				APIKey: "test-key",
				Models: []ModelInfo{
					{
						ID:        "claude-sonnet-4-20250514",
						MaxTokens: -1,
					},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateProviderConfig(&tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
