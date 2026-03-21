package config

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
)

type ValidationError struct {
	Field   string
	Message string
	Value   interface{}
}

type ValidationResult struct {
	Valid    bool
	Errors   []ValidationError
	Warnings []string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("field '%s': %s", e.Field, e.Message)
}

func Validate(config *Config) error {
	result := &ValidationResult{
		Valid:    true,
		Errors:   []ValidationError{},
		Warnings: []string{},
	}

	if config.Version == "" {
		result.addError("version", "version is required", "")
	}

	validateProjectConfig(&config.Project, result)
	validateLLMConfig(&config.LLM, result)
	validateMCPConfig(&config.MCP, result)
	validateAgentsConfig(&config.Agents, result)
	validateSkillsConfig(&config.Skills, result)
	validateContextConfig(&config.Context, result)
	validateTUIConfig(&config.TUI, result)
	validateServerConfig(&config.Server, result)
	validateSecurityConfig(&config.Security, result)

	if !result.Valid {
		return formatValidationErrors(result.Errors)
	}

	return nil
}

func validateProjectConfig(project *ProjectConfig, result *ValidationResult) {
	if project.Name == "" {
		result.addError("project.name", "project name is required", project.Name)
	}
	if project.Root == "" {
		result.addError("project.root", "project root is required", project.Root)
	}
}

func validateLLMConfig(llm *LLMConfig, result *ValidationResult) {
	if llm.DefaultProvider == "" {
		result.addError("llm.default_provider", "default provider is required", llm.DefaultProvider)
	}
	if llm.DefaultModel == "" {
		result.addError("llm.default_model", "default model is required", llm.DefaultModel)
	}

	for providerName, provider := range llm.Providers {
		if provider.APIKey == "" && provider.BaseURL == "" {
			result.addWarning(fmt.Sprintf("llm.providers.%s: provider has no API key or base URL", providerName))
		}
		for i, model := range provider.Models {
			if model.ID == "" {
				result.addError(fmt.Sprintf("llm.providers.%s.models[%d].id", providerName, i), "model ID is required", model.ID)
			}
			if model.MaxTokens <= 0 {
				result.addError(fmt.Sprintf("llm.providers.%s.models[%d].max_tokens", providerName, i), "max tokens must be positive", model.MaxTokens)
			}
		}
	}

	validProviders := make(map[string]bool)
	for name := range llm.Providers {
		validProviders[name] = true
	}

	if !validProviders[llm.DefaultProvider] {
		result.addError("llm.default_provider", fmt.Sprintf("default provider '%s' not found in providers list", llm.DefaultProvider), llm.DefaultProvider)
	}
}

func validateMCPConfig(mcp *MCPConfig, result *ValidationResult) {
	for serverName, server := range mcp.Servers {
		if server.Type == "" {
			result.addError(fmt.Sprintf("mcp.servers.%s.type", serverName), "server type is required", server.Type)
			continue
		}

		if server.Type == "stdio" {
			if server.Cmd == "" {
				result.addError(fmt.Sprintf("mcp.servers.%s.command", serverName), "command is required for stdio type", server.Cmd)
			}
		} else if server.Type == "http" || server.Type == "websocket" {
			if server.URL == "" {
				result.addError(fmt.Sprintf("mcp.servers.%s.url", serverName), "url is required for http/websocket type", server.URL)
			}
		} else {
			result.addError(fmt.Sprintf("mcp.servers.%s.type", serverName), fmt.Sprintf("invalid server type '%s', must be one of: stdio, http, websocket", server.Type), server.Type)
		}
	}
}

func validateAgentsConfig(agents *AgentsConfig, result *ValidationResult) {
	if agents.Defaults.MaxRetries < 0 {
		result.addError("agents.defaults.max_retries", "max retries cannot be negative", agents.Defaults.MaxRetries)
	}
	if agents.Defaults.Timeout.Duration <= 0 {
		result.addError("agents.defaults.timeout", "timeout must be positive", agents.Defaults.Timeout.Duration)
	}

	for roleName, role := range agents.Roles {
		if role.MinInstances < 0 {
			result.addError(fmt.Sprintf("agents.roles.%s.min_instances", roleName), "min instances cannot be negative", role.MinInstances)
		}
		if role.MaxInstances < role.MinInstances {
			result.addError(fmt.Sprintf("agents.roles.%s.max_instances", roleName), "max instances must be >= min instances", role.MaxInstances)
		}
	}
}

func validateSkillsConfig(skills *SkillsConfig, result *ValidationResult) {
	for i, skill := range skills.External {
		if skill.Name == "" {
			result.addError(fmt.Sprintf("skills.external[%d].name", i), "skill name is required", skill.Name)
		}
	}
}

func validateContextConfig(context *ContextConfig, result *ValidationResult) {
	if context.MaxTokens <= 0 {
		result.addError("context.max_tokens", "max tokens must be positive", context.MaxTokens)
	}
	if context.Compression.Ratio < 0 || context.Compression.Ratio > 1 {
		result.addError("context.compression.ratio", "compression ratio must be between 0 and 1", context.Compression.Ratio)
	}
	if context.Retrieval.TopK <= 0 {
		result.addError("context.retrieval.top_k", "top_k must be positive", context.Retrieval.TopK)
	}
}

func validateTUIConfig(tui *TUIConfig, result *ValidationResult) {
	if tui.Theme == "" {
		result.addError("tui.theme", "theme is required", tui.Theme)
	}
	if tui.Layout.SplitRatio < 0 || tui.Layout.SplitRatio > 1 {
		result.addError("tui.layout.split_ratio", "split ratio must be between 0 and 1", tui.Layout.SplitRatio)
	}
}

func validateServerConfig(server *ServerConfig, result *ValidationResult) {
	if !server.Enabled {
		return
	}

	if server.GRPC.Port <= 0 || server.GRPC.Port > 65535 {
		result.addError("server.grpc.port", "invalid grpc port", server.GRPC.Port)
	}
	if server.HTTP.Port <= 0 || server.HTTP.Port > 65535 {
		result.addError("server.http.port", "invalid http port", server.HTTP.Port)
	}
	if server.Auth.Enabled && server.Auth.JWTSecret == "" {
		result.addError("server.auth.jwt_secret", "jwt_secret is required when auth is enabled", server.Auth.JWTSecret)
	}
}

func validateSecurityConfig(security *SecurityConfig, result *ValidationResult) {
	if security.Sandbox.Enabled {
		validProfiles := map[string]bool{"standard": true, "strict": true, "permissive": true}
		if !validProfiles[security.Sandbox.Profile] {
			result.addError("security.sandbox.profile", fmt.Sprintf("invalid sandbox profile '%s', must be one of: standard, strict, permissive", security.Sandbox.Profile), security.Sandbox.Profile)
		}
	}

	if security.Audit.Enabled && security.Audit.Path == "" {
		result.addError("security.audit.path", "audit path is required when audit is enabled", security.Audit.Path)
	}

	validProviders := map[string]bool{"environment": true, "vault": true, "file": true}
	if !validProviders[security.Secrets.Provider] {
		result.addError("security.secrets.provider", fmt.Sprintf("invalid secrets provider '%s', must be one of: environment, vault, file", security.Secrets.Provider), security.Secrets.Provider)
	}
}

func (r *ValidationResult) addError(field, message string, value interface{}) {
	r.Valid = false
	r.Errors = append(r.Errors, ValidationError{
		Field:   field,
		Message: message,
		Value:   value,
	})
}

func (r *ValidationResult) addWarning(message string) {
	r.Warnings = append(r.Warnings, message)
}

func formatValidationErrors(errors []ValidationError) error {
	if len(errors) == 1 {
		return &errors[0]
	}

	var errMsgs []string
	for _, err := range errors {
		errMsgs = append(errMsgs, err.Error())
	}
	return fmt.Errorf("validation failed:\n%s", strings.Join(errMsgs, "\n"))
}

func GetProviderConfig(config *Config, providerName string) (*ProviderConfig, error) {
	provider, exists := config.LLM.Providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider '%s' not found in config", providerName)
	}
	return &provider, nil
}

func GetModelConfig(config *Config, providerName, modelID string) (*ModelInfo, error) {
	provider, err := GetProviderConfig(config, providerName)
	if err != nil {
		return nil, err
	}

	for _, model := range provider.Models {
		if model.ID == modelID {
			return &model, nil
		}
		if model.Alias == modelID {
			return &model, nil
		}
	}

	return nil, fmt.Errorf("model '%s' not found in provider '%s'", modelID, providerName)
}

func GetMCPServerConfig(config *Config, serverName string) (*MCPServerConfig, error) {
	server, exists := config.MCP.Servers[serverName]
	if !exists {
		return nil, fmt.Errorf("MCP server '%s' not found in config", serverName)
	}
	return &server, nil
}

func GetAgentRoleConfig(config *Config, roleName string) (*Role, error) {
	role, exists := config.Agents.Roles[roleName]
	if !exists {
		return nil, fmt.Errorf("agent role '%s' not found in config", roleName)
	}
	return &role, nil
}

func HasProvider(config *Config, providerName string) bool {
	_, exists := config.LLM.Providers[providerName]
	return exists
}

func HasModel(config *Config, providerName, modelID string) bool {
	_, err := GetModelConfig(config, providerName, modelID)
	return err == nil
}

func GetAgentModel(config *Config, agentType string) (string, error) {
	if model, ok := config.LLM.AgentModelMapping[agentType]; ok {
		return model, nil
	}
	return config.LLM.DefaultModel, nil
}

func GetProviderForModel(config *Config, modelID string) (string, error) {
	for providerName, provider := range config.LLM.Providers {
		for _, model := range provider.Models {
			if model.ID == modelID || model.Alias == modelID {
				return providerName, nil
			}
		}
	}
	return "", fmt.Errorf("model '%s' not found in any provider", modelID)
}

func ValidateProviderConfig(provider *ProviderConfig) error {
	if provider.APIKey == "" && provider.BaseURL == "" {
		return fmt.Errorf("provider must have either API key or base URL")
	}
	if len(provider.Models) == 0 {
		return fmt.Errorf("provider must have at least one model")
	}
	for i, model := range provider.Models {
		if model.ID == "" {
			return fmt.Errorf("model[%d].id is required", i)
		}
		if model.MaxTokens <= 0 {
			return fmt.Errorf("model[%d].max_tokens must be positive", i)
		}
	}
	return nil
}

func MergeConfigs(base, override *Config) *Config {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}

	result := deepCopy(base)
	mergeStruct(reflect.ValueOf(result).Elem(), reflect.ValueOf(override).Elem())
	return result
}

func deepCopy(config *Config) *Config {
	data, err := json.Marshal(config)
	if err != nil {
		return nil
	}
	var copy Config
	if err := json.Unmarshal(data, &copy); err != nil {
		return nil
	}
	return &copy
}

func mergeStruct(base, override reflect.Value) {
	for i := 0; i < override.NumField(); i++ {
		fieldName := override.Type().Field(i).Name
		baseField := base.FieldByName(fieldName)
		overrideField := override.Field(i)

		if !baseField.IsValid() || !overrideField.IsValid() {
			continue
		}

		if overrideField.IsZero() {
			continue
		}

		switch overrideField.Kind() {
		case reflect.Struct:
			if baseField.CanAddr() {
				mergeStruct(baseField, overrideField)
			}
		case reflect.Map:
			if overrideField.Len() > 0 {
				baseField.Set(reflect.MakeMap(overrideField.Type()))
				for _, key := range overrideField.MapKeys() {
					baseField.SetMapIndex(key, overrideField.MapIndex(key))
				}
			}
		case reflect.Slice:
			if overrideField.Len() > 0 {
				baseField.Set(reflect.MakeSlice(overrideField.Type(), overrideField.Len(), overrideField.Cap()))
				for i := 0; i < overrideField.Len(); i++ {
					baseField.Index(i).Set(overrideField.Index(i))
				}
			}
		default:
			baseField.Set(overrideField)
		}
	}
}
