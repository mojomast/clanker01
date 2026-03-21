package internal

import (
	"github.com/swarm-ai/swarm/internal/config"
)

func loadConfig() (*config.Config, error) {
	opts := &config.LoadOptions{}
	if cfgFile != "" {
		opts.ConfigPaths = []string{cfgFile}
	}
	cfg, err := config.NewLoader(opts).Load()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func loadConfigOrDefault() *config.Config {
	cfg, err := loadConfig()
	if err != nil {
		return &config.Config{}
	}
	return cfg
}
