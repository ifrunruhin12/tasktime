package client

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type ClientConfig struct {
	Username    string `json:"username"`
	ServerURL   string `json:"server_url"`
	AuthToken   string `json:"auth_token"`
	AutoConnect bool   `json:"auto_connect"`
	Theme       string `json:"theme"`
}

type ConfigManager struct {
	configPath string
}

func NewConfigManager() (*ConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".tasktime")
	configPath := filepath.Join(configDir, "config.json")

	return &ConfigManager{
		configPath: configPath,
	}, nil
}

func (cm *ConfigManager) Exists() bool {
	_, err := os.Stat(cm.configPath)
	return err == nil
}

func (cm *ConfigManager) Load() (*ClientConfig, error) {
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

func (cm *ConfigManager) Save(config *ClientConfig) error {
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func GetDefaultConfig() *ClientConfig {
	return &ClientConfig{
		ServerURL:   "http://localhost:8080",
		AutoConnect: true,
		Theme:       "default",
	}
}
