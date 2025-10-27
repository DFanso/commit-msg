// Package store persists user-selected LLM providers and credentials.
package store

import (
	"encoding/json"
	"errors"
	"fmt"

	"os"

	"github.com/99designs/keyring"

	"github.com/dfanso/commit-msg/internal/cache"
	"github.com/dfanso/commit-msg/internal/usage"
	"github.com/dfanso/commit-msg/pkg/types"
	StoreUtils "github.com/dfanso/commit-msg/utils"
)

type StoreMethods struct {
	ring  keyring.Keyring
	cache *cache.CacheManager
	usage *usage.StatsManager
}

// NewStoreMethods creates a new StoreMethods instance with cache support.
func NewStoreMethods() (*StoreMethods, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: "commit-msg",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open keyring: %w", err)
	}

	cacheManager, err := cache.NewCacheManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize cache: %w", err)
	}

	usageManager, err := usage.NewStatsManager()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize usage statistics: %w", err)
	}

	return &StoreMethods{
		ring:  ring,
		cache: cacheManager,
		usage: usageManager,
	}, nil
}

// KeyringInit initializes a StoreMethods instance with both keyring and cache support.
// This function is kept for backward compatibility with main.go.
func KeyringInit() (*StoreMethods, error) {
	return NewStoreMethods()
}

// LLMProvider represents a single stored LLM provider and its credential.
type LLMProvider struct {
	LLM    types.LLMProvider `json:"model"`
	APIKey string            `json:"api_key"`
}

// Config describes the on-disk structure for all saved LLM providers.
type Config struct {
	Default      types.LLMProvider   `json:"default"`
	LLMProviders []types.LLMProvider `json:"models"`
}

// Save persists or updates an LLM provider entry, marking it as the default.
func (s *StoreMethods) Save(LLMConfig LLMProvider) error {

	var cfg Config

	configPath, err := StoreUtils.GetConfigPath()
	if err != nil {
		return err
	}

	isConfigExists := StoreUtils.CheckConfig(configPath)
	if !isConfigExists {
		err := StoreUtils.CreateConfigFile(configPath)
		if err != nil {
			return err
		}
	}

	data, err := os.ReadFile(configPath)
	if errors.Is(err, os.ErrNotExist) {
		data = []byte("{}")
	} else if err != nil {
		return err
	}

	if len(data) > 2 {
		err = json.Unmarshal(data, &cfg)
		if err != nil {
			// If unmarshal fails, it might be due to old config format
			// Reset to empty config to allow fresh setup
			return fmt.Errorf("config file format error: %w. Please delete the config and run setup again", err)
		}
	}

	// If Model already present in config, update the apiKey
	updated := false
	for _, p := range cfg.LLMProviders {
		if p == LLMConfig.LLM {
			err := s.ring.Set(keyring.Item{ //save apiKey using keychain to OS credentials
				Key:  string(LLMConfig.LLM),
				Data: []byte(LLMConfig.APIKey),
			})
			if err != nil {
				return fmt.Errorf("failed to store credentials in keyring: %w", err)
			}
			updated = true
			break
		}
	}

	// If fresh Model is saved, means model not exists in config file
	if !updated {
		cfg.LLMProviders = append(cfg.LLMProviders, LLMConfig.LLM)
		err := s.ring.Set(keyring.Item{ //save apiKey using keychain to OS credentials
			Key:  string(LLMConfig.LLM),
			Data: []byte(LLMConfig.APIKey),
		})
		if err != nil {
			return fmt.Errorf("failed to store credentials in keyring: %w", err)
		}
	}

	cfg.Default = LLMConfig.LLM

	data, err = json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// DefaultLLMKey returns the currently selected default LLM provider, if any.
func (s *StoreMethods) DefaultLLMKey() (*LLMProvider, error) {

	var cfg Config
	var useModel LLMProvider

	configPath, err := StoreUtils.GetConfigPath()
	if err != nil {
		return nil, err
	}

	isConfigExists := StoreUtils.CheckConfig(configPath)
	if !isConfigExists {
		return nil, fmt.Errorf("config file does not exist at %s, run 'commit llm setup' to create it", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if len(data) > 2 {
		err = json.Unmarshal(data, &cfg)
		if err != nil {
			// If unmarshal fails, it might be due to old config format
			return nil, fmt.Errorf("config file format error: %w. Please delete the config and run setup again", err)
		}
	} else {
		return nil, errors.New("config file is empty, run 'commit llm setup' to add your first LLM provider")
	}

	defaultLLM := cfg.Default

	for i, p := range cfg.LLMProviders {
		if p == defaultLLM {
			useModel.LLM = cfg.LLMProviders[i]         // Fetches default Model from config json
			i, err := s.ring.Get(string(useModel.LLM)) //Fetches apiKey from OS credential for default model
			if err != nil {
				return nil, err
			}
			useModel.APIKey = string(i.Data)
			return &useModel, nil
		}
	}
	return nil, fmt.Errorf("default model '%s' not found in saved providers, run 'commit llm setup' to configure it", defaultLLM)
}

// ListSavedModels loads all persisted LLM provider configurations.
func ListSavedModels() (*Config, error) {

	var cfg Config

	configPath, err := StoreUtils.GetConfigPath()
	if err != nil {
		return nil, err
	}

	isConfigExists := StoreUtils.CheckConfig(configPath)
	if !isConfigExists {
		return nil, fmt.Errorf("config file does not exist at %s, run 'commit llm setup' to create it", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if len(data) > 2 {
		err = json.Unmarshal(data, &cfg)
		if err != nil {
			// If unmarshal fails, it might be due to old config format
			return nil, fmt.Errorf("config file format error: %w. Please delete the config and run setup again", err)
		}
	} else {
		return nil, errors.New("config file is empty, run 'commit llm setup' to add your first LLM provider")
	}

	return &cfg, nil

}

// ChangeDefault updates the default LLM provider selection in the config.
func ChangeDefault(Model types.LLMProvider) error {

	var cfg Config

	configPath, err := StoreUtils.GetConfigPath()
	if err != nil {
		return err
	}

	isConfigExists := StoreUtils.CheckConfig(configPath)
	if !isConfigExists {
		return fmt.Errorf("config file does not exist at %s, run 'commit llm setup' to create it", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if len(data) > 2 {
		err = json.Unmarshal(data, &cfg)
		if err != nil {
			// If unmarshal fails, it might be due to old config format
			return fmt.Errorf("config file format error: %w. Please delete the config and run setup again", err)
		}
	}

	found := false
	for _, p := range cfg.LLMProviders {
		if p == Model {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("cannot set default to %s: no saved entry", Model.String())
	}

	cfg.Default = Model

	data, err = json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)
}

// DeleteModel removes the specified provider from the saved configuration.
func (s *StoreMethods) DeleteModel(Model types.LLMProvider) error {

	var cfg Config
	var newCfg Config

	configPath, err := StoreUtils.GetConfigPath()
	if err != nil {
		return err
	}

	isConfigExists := StoreUtils.CheckConfig(configPath)
	if !isConfigExists {
		return fmt.Errorf("config file does not exist at %s, run 'commit llm setup' to create it", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if len(data) > 2 {
		err = json.Unmarshal(data, &cfg)
		if err != nil {
			// If unmarshal fails, it might be due to old config format
			return fmt.Errorf("config file format error: %w. Please delete the config and run setup again", err)
		}
	}

	if Model == cfg.Default {
		if len(cfg.LLMProviders) > 1 {
			return fmt.Errorf("cannot delete %s while it is default, set other model default first", Model.String())
		} else {
			err := s.ring.Remove(string(Model)) // Removes the apiKey from OS credentials
			if err != nil {
				return err
			}
			return os.WriteFile(configPath, []byte("{}"), 0600)
		}
	} else {

		for _, p := range cfg.LLMProviders {

			if p != Model {
				newCfg.LLMProviders = append(newCfg.LLMProviders, p)
			}
		}

		err := s.ring.Remove(string(Model)) //Remove the apiKey from OS credentials
		if err != nil {
			return err
		}
		newCfg.Default = cfg.Default

		data, err = json.MarshalIndent(newCfg, "", " ")
		if err != nil {
			return err
		}
		return os.WriteFile(configPath, data, 0600)

	}
}

// UpdateAPIKey rotates the credential for an existing provider entry.
func (s *StoreMethods) UpdateAPIKey(Model types.LLMProvider, APIKey string) error {

	var cfg Config

	configPath, err := StoreUtils.GetConfigPath()
	if err != nil {
		return err
	}

	isConfigExists := StoreUtils.CheckConfig(configPath)
	if !isConfigExists {
		return fmt.Errorf("config file does not exist at %s, run 'commit llm setup' to create it", configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	if len(data) > 2 {
		err = json.Unmarshal(data, &cfg)
		if err != nil {
			// If unmarshal fails, it might be due to old config format
			return fmt.Errorf("config file format error: %w. Please delete the config and run setup again", err)
		}
	}

	updated := false
	for _, p := range cfg.LLMProviders {
		if p == Model {
			err := s.ring.Set(keyring.Item{ // Update the apiKey in OS credential
				Key:  string(Model),
				Data: []byte(APIKey),
			})
			if err != nil {
				return fmt.Errorf("failed to update credentials in keyring: %w", err)
			}
			updated = true
		}
	}

	if !updated {
		return fmt.Errorf("no saved entry for %s to update", Model.String())
	}

	data, err = json.MarshalIndent(cfg, "", " ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0600)

}

// Cache management methods

// GetCacheManager returns the cache manager instance.
func (s *StoreMethods) GetCacheManager() *cache.CacheManager {
	return s.cache
}

// GetCachedMessage retrieves a cached commit message if it exists.
func (s *StoreMethods) GetCachedMessage(provider types.LLMProvider, diff string, opts *types.GenerationOptions) (*types.CacheEntry, bool) {
	return s.cache.Get(provider, diff, opts)
}

// SetCachedMessage stores a commit message in the cache.
func (s *StoreMethods) SetCachedMessage(provider types.LLMProvider, diff string, opts *types.GenerationOptions, message string, cost float64, tokens *types.UsageInfo) error {
	return s.cache.Set(provider, diff, opts, message, cost, tokens)
}

// ClearCache removes all entries from the cache.
func (s *StoreMethods) ClearCache() error {
	return s.cache.Clear()
}

// GetCacheStats returns cache statistics.
func (s *StoreMethods) GetCacheStats() *types.CacheStats {
	return s.cache.GetStats()
}

// CleanupCache removes old entries from the cache.
func (s *StoreMethods) CleanupCache() error {
	return s.cache.Cleanup()
}

// Usage statistics management methods

// GetUsageManager returns the usage statistics manager instance.
func (s *StoreMethods) GetUsageManager() *usage.StatsManager {
	return s.usage
}

// RecordGenerationEvent records a commit message generation event for statistics.
func (s *StoreMethods) RecordGenerationEvent(event *types.GenerationEvent) error {
	return s.usage.RecordGeneration(event)
}

// GetUsageStats returns comprehensive usage statistics.
func (s *StoreMethods) GetUsageStats() *types.UsageStats {
	return s.usage.GetStats()
}

// GetMostUsedProvider returns the most frequently used LLM provider.
func (s *StoreMethods) GetMostUsedProvider() (types.LLMProvider, int) {
	return s.usage.GetMostUsedProvider()
}

// GetOverallSuccessRate returns the overall success rate as a percentage.
func (s *StoreMethods) GetOverallSuccessRate() float64 {
	return s.usage.GetSuccessRate()
}

// GetCacheHitRate returns the cache hit rate as a percentage.
func (s *StoreMethods) GetCacheHitRate() float64 {
	return s.usage.GetCacheHitRate()
}

// GetProviderRanking returns providers ranked by usage frequency.
func (s *StoreMethods) GetProviderRanking() []types.LLMProvider {
	return s.usage.GetProviderRanking()
}

// ResetUsageStats clears all usage statistics.
func (s *StoreMethods) ResetUsageStats() error {
	return s.usage.ResetStats()
}
