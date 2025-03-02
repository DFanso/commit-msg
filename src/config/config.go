package config

import (
	"encoding/json"
	"os"

	"github.com/dfanso/commit-msg/src/types"
)

// Load configuration from file
func LoadConfig(configFile string) (*types.Config, error) {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}

	var config types.Config
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// Save configuration to file
func SaveConfig(configFile string, config *types.Config) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}
