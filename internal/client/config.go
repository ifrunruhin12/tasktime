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
	LogConfigEvent("loading_config", "path", cm.configPath)
	
	data, err := os.ReadFile(cm.configPath)
	if err != nil {
		LogError("Failed to read config file", "error", err.Error(), "path", cm.configPath)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config ClientConfig
	if err := json.Unmarshal(data, &config); err != nil {
		LogError("Failed to parse config file", "error", err.Error(), "path", cm.configPath)
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	LogConfigEvent("config_loaded", "username", config.Username, "server_url", config.ServerURL, "auto_connect", config.AutoConnect)
	return &config, nil
}

func (cm *ConfigManager) Save(config *ClientConfig) error {
	LogConfigEvent("saving_config", "username", config.Username, "server_url", config.ServerURL, "auto_connect", config.AutoConnect)
	
	configDir := filepath.Dir(cm.configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		LogError("Failed to create config directory", "error", err.Error(), "path", configDir)
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		LogError("Failed to marshal config", "error", err.Error())
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(cm.configPath, data, 0600); err != nil {
		LogError("Failed to write config file", "error", err.Error(), "path", cm.configPath)
		return fmt.Errorf("failed to write config file: %w", err)
	}

	LogConfigEvent("config_saved", "path", cm.configPath)
	return nil
}

func GetDefaultConfig() *ClientConfig {
	return &ClientConfig{
		ServerURL:   "http://localhost:8080",
		AutoConnect: true,
		Theme:       "default",
	}
}
