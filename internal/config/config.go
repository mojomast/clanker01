package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Loader struct {
	configPaths []string
	envPrefix   string
}

type LoadOptions struct {
	ConfigPaths []string
	EnvPrefix   string
}

func NewLoader(opts *LoadOptions) *Loader {
	if opts == nil {
		opts = &LoadOptions{
			ConfigPaths: []string{
				"swarm.yaml",
				"swarm.yml",
				"swarm.json",
				"config/swarm.yaml",
				"config/swarm.yml",
				"config/swarm.json",
				".swarm/config.yaml",
				".swarm/config.yml",
				".swarm/config.json",
			},
			EnvPrefix: "SWARM",
		}
	}
	return &Loader{
		configPaths: opts.ConfigPaths,
		envPrefix:   opts.EnvPrefix,
	}
}

func (l *Loader) Load() (*Config, error) {
	configPath, err := l.findConfigFile()
	if err != nil {
		return nil, fmt.Errorf("failed to find config file: %w", err)
	}

	config, err := l.loadFromFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	if err := l.applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply env overrides: %w", err)
	}

	if err := Validate(config); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return config, nil
}

func (l *Loader) LoadFromBytes(data []byte, format string) (*Config, error) {
	config := &Config{}

	switch strings.ToLower(format) {
	case "yaml", "yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse YAML: %w", err)
		}
	case "json":
		if err := json.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse JSON: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err := l.applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply env overrides: %w", err)
	}

	return config, nil
}

func (l *Loader) findConfigFile() (string, error) {
	for _, path := range l.configPaths {
		if absPath, err := filepath.Abs(path); err == nil {
			if _, err := os.Stat(absPath); err == nil {
				return absPath, nil
			}
		}
	}
	return "", fmt.Errorf("no config file found in search paths: %v", l.configPaths)
}

func (l *Loader) loadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".yaml", ".yml":
		return l.loadYAML(data)
	case ".json":
		return l.loadJSON(data)
	default:
		return nil, fmt.Errorf("unsupported file extension: %s", ext)
	}
}

func (l *Loader) loadYAML(data []byte) (*Config, error) {
	config := &Config{}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)

	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("YAML decode error: %w", err)
	}

	return config, nil
}

func (l *Loader) loadJSON(data []byte) (*Config, error) {
	config := &Config{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(config); err != nil {
		return nil, fmt.Errorf("JSON decode error: %w", err)
	}

	return config, nil
}

func (l *Loader) applyEnvOverrides(config *Config) error {
	if config.LLM.Providers == nil {
		config.LLM.Providers = make(map[string]ProviderConfig)
	}

	for providerName := range config.LLM.Providers {
		if apiKey := os.Getenv(fmt.Sprintf("%s_%s_API_KEY", l.envPrefix, strings.ToUpper(providerName))); apiKey != "" {
			provider := config.LLM.Providers[providerName]
			if provider.Options == nil {
				provider.Options = make(map[string]any)
			}
			provider.APIKey = apiKey
			config.LLM.Providers[providerName] = provider
		}
		if baseURL := os.Getenv(fmt.Sprintf("%s_%s_BASE_URL", l.envPrefix, strings.ToUpper(providerName))); baseURL != "" {
			provider := config.LLM.Providers[providerName]
			if provider.Options == nil {
				provider.Options = make(map[string]any)
			}
			provider.BaseURL = baseURL
			config.LLM.Providers[providerName] = provider
		}
	}

	if projectRoot := os.Getenv("PROJECT_ROOT"); projectRoot != "" {
		config.Project.Root = projectRoot
	}

	if jwtSecret := os.Getenv("JWT_SECRET"); jwtSecret != "" {
		config.Server.Auth.JWTSecret = jwtSecret
	}

	return nil
}

func LoadDefault() (*Config, error) {
	loader := NewLoader(nil)
	return loader.Load()
}

func LoadFromBytes(data []byte, format string) (*Config, error) {
	loader := NewLoader(nil)
	return loader.LoadFromBytes(data, format)
}
