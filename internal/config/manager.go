package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type Manager struct {
	mu         sync.RWMutex
	config     *Config
	configPath string
	envPrefix  string
	loader     *Loader
	watcher    *watcher
	reloadChan chan *Config
	errorChan  chan error
	stopChan   chan struct{}
	closed     bool
}

type ManagerOptions struct {
	ConfigPath string
	EnvPrefix  string
	AutoReload bool
}

type watcher struct {
	enabled  bool
	path     string
	modTime  time.Time
	stopChan chan struct{}
}

func NewManager(opts *ManagerOptions) (*Manager, error) {
	if opts == nil {
		opts = &ManagerOptions{}
	}

	loaderOpts := &LoadOptions{
		EnvPrefix: opts.EnvPrefix,
	}
	if opts.ConfigPath != "" {
		loaderOpts.ConfigPaths = []string{opts.ConfigPath}
	}

	mgr := &Manager{
		loader:     NewLoader(loaderOpts),
		envPrefix:  opts.EnvPrefix,
		reloadChan: make(chan *Config, 1),
		errorChan:  make(chan error, 10),
		stopChan:   make(chan struct{}),
	}

	config, err := mgr.loader.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load initial config: %w", err)
	}

	if opts.ConfigPath != "" {
		mgr.configPath = opts.ConfigPath
	} else {
		absPath, err := filepath.Abs("swarm.yaml")
		if err == nil {
			mgr.configPath = absPath
		}
	}

	mgr.config = config

	if opts.AutoReload && mgr.configPath != "" {
		mgr.watcher = &watcher{
			enabled:  true,
			path:     mgr.configPath,
			stopChan: make(chan struct{}),
		}
		if err := mgr.startWatcher(); err != nil {
			return nil, fmt.Errorf("failed to start config watcher: %w", err)
		}
	}

	return mgr, nil
}

func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	config, err := m.loader.Load()
	if err != nil {
		return fmt.Errorf("failed to reload config: %w", err)
	}

	oldConfig := m.config
	m.config = config

	if m.watcher != nil {
		if info, err := os.Stat(m.configPath); err == nil {
			m.watcher.modTime = info.ModTime()
		}
	}

	if m.reloadChan != nil {
		select {
		case m.reloadChan <- config:
		default:
		}
	}

	_ = oldConfig
	return nil
}

func (m *Manager) Update(updater func(*Config) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if err := updater(m.config); err != nil {
		return fmt.Errorf("config update failed: %w", err)
	}

	if err := Validate(m.config); err != nil {
		return fmt.Errorf("config validation failed after update: %w", err)
	}

	return nil
}

func (m *Manager) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.configPath == "" {
		return fmt.Errorf("no config path set, cannot save")
	}

	ext := filepath.Ext(m.configPath)
	switch ext {
	case ".yaml", ".yml":
		return m.saveYAML()
	case ".json":
		return m.saveJSON()
	default:
		return fmt.Errorf("unsupported config file extension: %s", ext)
	}
}

func (m *Manager) saveYAML() error {
	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config to YAML: %w", err)
	}
	return os.WriteFile(m.configPath, data, 0644)
}

func (m *Manager) saveJSON() error {
	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config to JSON: %w", err)
	}
	return os.WriteFile(m.configPath, data, 0644)
}

func (m *Manager) Watch() <-chan *Config {
	return m.reloadChan
}

func (m *Manager) Errors() <-chan error {
	return m.errorChan
}

func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true

	if m.stopChan != nil {
		close(m.stopChan)
		m.stopChan = nil
	}

	if m.reloadChan != nil {
		close(m.reloadChan)
		m.reloadChan = nil
	}

	if m.errorChan != nil {
		close(m.errorChan)
		m.errorChan = nil
	}

	if m.watcher != nil && m.watcher.stopChan != nil {
		close(m.watcher.stopChan)
		m.watcher = nil
	}

	return nil
}

func (m *Manager) startWatcher() error {
	if info, err := os.Stat(m.configPath); err == nil {
		m.watcher.modTime = info.ModTime()
	}

	go m.watchLoop()
	return nil
}

func (m *Manager) watchLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := m.checkConfigChange(); err != nil && m.errorChan != nil {
				select {
				case m.errorChan <- err:
				default:
				}
			}
		case <-m.watcher.stopChan:
			return
		case <-m.stopChan:
			return
		}
	}
}

func (m *Manager) checkConfigChange() error {
	info, err := os.Stat(m.configPath)
	if err != nil {
		return fmt.Errorf("failed to stat config file: %w", err)
	}

	m.mu.RLock()
	lastModTime := m.watcher.modTime
	m.mu.RUnlock()

	if info.ModTime().After(lastModTime) {
		if err := m.Reload(); err != nil {
			return fmt.Errorf("config reload failed: %w", err)
		}
	}

	return nil
}

func (m *Manager) GetLLMConfig() *LLMConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return &m.config.LLM
}

func (m *Manager) GetProviderConfig(providerName string) (*ProviderConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return GetProviderConfig(m.config, providerName)
}

func (m *Manager) GetModelConfig(providerName, modelID string) (*ModelInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return GetModelConfig(m.config, providerName, modelID)
}

func (m *Manager) GetMCPServerConfig(serverName string) (*MCPServerConfig, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return GetMCPServerConfig(m.config, serverName)
}

func (m *Manager) GetAgentRoleConfig(roleName string) (*Role, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return GetAgentRoleConfig(m.config, roleName)
}

func (m *Manager) HasProvider(providerName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return HasProvider(m.config, providerName)
}

func (m *Manager) HasModel(providerName, modelID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return HasModel(m.config, providerName, modelID)
}

func (m *Manager) GetAgentModel(agentType string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return GetAgentModel(m.config, agentType)
}

func (m *Manager) GetProviderForModel(modelID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return GetProviderForModel(m.config, modelID)
}

func (m *Manager) GetProjectName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Project.Name
}

func (m *Manager) GetProjectRoot() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Project.Root
}

func (m *Manager) GetDefaultProvider() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.LLM.DefaultProvider
}

func (m *Manager) GetDefaultModel() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.LLM.DefaultModel
}

func (m *Manager) IsServerEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server.Enabled
}

func (m *Manager) GetGRPCPort() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server.GRPC.Port
}

func (m *Manager) GetHTTPPort() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Server.HTTP.Port
}
