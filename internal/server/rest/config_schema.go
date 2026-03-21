package rest

// configSchemaDefinition is a static JSON-schema-like description of the
// Config struct.  It is returned by the GET /api/v1/config/schema endpoint so
// that clients (UIs, CLI tools) can generate forms or validate payloads
// without hard-coding knowledge of the Go types.
var configSchemaDefinition = map[string]interface{}{
	"$schema":     "http://json-schema.org/draft-07/schema#",
	"title":       "Swarm Configuration",
	"description": "Configuration schema for the SWARM AI agent orchestration system",
	"type":        "object",
	"properties": map[string]interface{}{
		"version": map[string]interface{}{
			"type":        "string",
			"description": "Configuration schema version",
		},
		"project": map[string]interface{}{
			"type":        "object",
			"description": "Project settings",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Project name",
				},
				"root": map[string]interface{}{
					"type":        "string",
					"description": "Project root directory",
				},
			},
			"required": []string{"name", "root"},
		},
		"llm": map[string]interface{}{
			"type":        "object",
			"description": "LLM provider configuration",
			"properties": map[string]interface{}{
				"default_provider": map[string]interface{}{
					"type":        "string",
					"description": "Default LLM provider name",
				},
				"default_model": map[string]interface{}{
					"type":        "string",
					"description": "Default model identifier",
				},
				"providers": map[string]interface{}{
					"type":        "object",
					"description": "Map of provider name to provider config",
					"additionalProperties": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"api_key": map[string]interface{}{
								"type":        "string",
								"description": "Provider API key",
							},
							"base_url": map[string]interface{}{
								"type":        "string",
								"description": "Provider base URL (for self-hosted models)",
							},
							"models": map[string]interface{}{
								"type":        "array",
								"description": "Available models",
								"items": map[string]interface{}{
									"type": "object",
									"properties": map[string]interface{}{
										"id": map[string]interface{}{
											"type":        "string",
											"description": "Model identifier",
										},
										"alias": map[string]interface{}{
											"type":        "string",
											"description": "Short alias for the model",
										},
										"max_tokens": map[string]interface{}{
											"type":        "integer",
											"description": "Maximum token limit",
											"minimum":     1,
										},
									},
									"required": []string{"id", "max_tokens"},
								},
							},
							"options": map[string]interface{}{
								"type":        "object",
								"description": "Additional provider-specific options",
							},
						},
					},
				},
				"agent_model_mapping": map[string]interface{}{
					"type":        "object",
					"description": "Map of agent type to preferred model",
					"additionalProperties": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"default_provider", "default_model"},
		},
		"mcp": map[string]interface{}{
			"type":        "object",
			"description": "MCP (Model Context Protocol) server configuration",
			"properties": map[string]interface{}{
				"servers": map[string]interface{}{
					"type":        "object",
					"description": "Map of server name to server config",
					"additionalProperties": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"type": map[string]interface{}{
								"type":        "string",
								"enum":        []string{"stdio", "http", "websocket"},
								"description": "Server transport type",
							},
							"command": map[string]interface{}{
								"type":        "string",
								"description": "Command to run (stdio type)",
							},
							"args": map[string]interface{}{
								"type":        "array",
								"items":       map[string]interface{}{"type": "string"},
								"description": "Command arguments",
							},
							"env": map[string]interface{}{
								"type":                 "object",
								"additionalProperties": map[string]interface{}{"type": "string"},
								"description":          "Environment variables",
							},
							"url": map[string]interface{}{
								"type":        "string",
								"description": "Server URL (http/websocket type)",
							},
						},
						"required": []string{"type"},
					},
				},
			},
		},
		"agents": map[string]interface{}{
			"type":        "object",
			"description": "Agent configuration",
			"properties": map[string]interface{}{
				"defaults": map[string]interface{}{
					"type":        "object",
					"description": "Default settings for all agents",
					"properties": map[string]interface{}{
						"timeout": map[string]interface{}{
							"type":        "string",
							"description": "Default agent timeout (Go duration string, e.g. '5m')",
						},
						"max_retries": map[string]interface{}{
							"type":        "integer",
							"description": "Default max retries",
							"minimum":     0,
						},
					},
				},
				"roles": map[string]interface{}{
					"type":        "object",
					"description": "Map of role name to role config",
					"additionalProperties": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"min_instances": map[string]interface{}{
								"type":    "integer",
								"minimum": 0,
							},
							"max_instances": map[string]interface{}{
								"type":    "integer",
								"minimum": 0,
							},
							"model": map[string]interface{}{
								"type":        "string",
								"description": "Model to use for this role",
							},
						},
					},
				},
			},
		},
		"skills": map[string]interface{}{
			"type":        "object",
			"description": "Skill configuration",
			"properties": map[string]interface{}{
				"builtin": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "List of built-in skill names to enable",
				},
				"external": map[string]interface{}{
					"type":        "array",
					"description": "External skill definitions",
					"items": map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"name":    map[string]interface{}{"type": "string"},
							"version": map[string]interface{}{"type": "string"},
							"config": map[string]interface{}{
								"type":                 "object",
								"additionalProperties": map[string]interface{}{"type": "string"},
							},
						},
						"required": []string{"name"},
					},
				},
			},
		},
		"context": map[string]interface{}{
			"type":        "object",
			"description": "Context management configuration",
			"properties": map[string]interface{}{
				"max_tokens": map[string]interface{}{
					"type":    "integer",
					"minimum": 1,
				},
				"compression": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"enabled": map[string]interface{}{"type": "boolean"},
						"ratio": map[string]interface{}{
							"type":    "number",
							"minimum": 0,
							"maximum": 1,
						},
					},
				},
				"retrieval": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"vector_store":    map[string]interface{}{"type": "string"},
						"embedding_model": map[string]interface{}{"type": "string"},
						"top_k": map[string]interface{}{
							"type":    "integer",
							"minimum": 1,
						},
					},
				},
			},
		},
		"tui": map[string]interface{}{
			"type":        "object",
			"description": "Terminal UI configuration",
			"properties": map[string]interface{}{
				"theme": map[string]interface{}{"type": "string"},
				"layout": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"split_ratio": map[string]interface{}{
							"type":    "number",
							"minimum": 0,
							"maximum": 1,
						},
					},
				},
				"keybinds": map[string]interface{}{
					"type":                 "object",
					"additionalProperties": map[string]interface{}{"type": "string"},
				},
			},
		},
		"server": map[string]interface{}{
			"type":        "object",
			"description": "Server configuration",
			"properties": map[string]interface{}{
				"enabled": map[string]interface{}{"type": "boolean"},
				"grpc": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"port": map[string]interface{}{
							"type":    "integer",
							"minimum": 1,
							"maximum": 65535,
						},
					},
				},
				"http": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"port": map[string]interface{}{
							"type":    "integer",
							"minimum": 1,
							"maximum": 65535,
						},
					},
				},
				"auth": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"enabled":    map[string]interface{}{"type": "boolean"},
						"jwt_secret": map[string]interface{}{"type": "string"},
					},
				},
			},
		},
		"security": map[string]interface{}{
			"type":        "object",
			"description": "Security configuration",
			"properties": map[string]interface{}{
				"sandbox": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"enabled": map[string]interface{}{"type": "boolean"},
						"profile": map[string]interface{}{
							"type": "string",
							"enum": []string{"standard", "strict", "permissive"},
						},
					},
				},
				"audit": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"enabled": map[string]interface{}{"type": "boolean"},
						"path":    map[string]interface{}{"type": "string"},
					},
				},
				"secrets": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"provider": map[string]interface{}{
							"type": "string",
							"enum": []string{"environment", "vault", "file"},
						},
					},
				},
			},
		},
	},
	"required": []string{"version", "project"},
}
