package usage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dfanso/commit-msg/pkg/types"
	StoreUtils "github.com/dfanso/commit-msg/utils"
)

// StatsManager handles usage statistics tracking and persistence.
type StatsManager struct {
	mu       sync.RWMutex
	stats    *types.UsageStats
	filePath string
}

// NewStatsManager creates a new statistics manager instance.
func NewStatsManager() (*StatsManager, error) {
	configPath, err := StoreUtils.GetConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	// Get the directory from the config path
	configDir := filepath.Dir(configPath)
	statsPath := filepath.Join(configDir, "usage_stats.json")
	
	manager := &StatsManager{
		filePath: statsPath,
		stats: &types.UsageStats{
			ProviderStats: make(map[types.LLMProvider]*types.ProviderStats),
		},
	}

	// Load existing stats if they exist
	if err := manager.load(); err != nil {
		// If file doesn't exist, that's okay - we'll create it on first save
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to load existing stats: %w", err)
		}
	}

	return manager, nil
}

// RecordGeneration records a commit message generation event.
func (sm *StatsManager) RecordGeneration(event *types.GenerationEvent) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	now := time.Now().UTC().Format(time.RFC3339)
	
	// Update global stats
	sm.stats.TotalGenerations++
	if event.Success {
		sm.stats.SuccessfulGenerations++
	} else {
		sm.stats.FailedGenerations++
	}

	sm.stats.TotalCost += event.Cost
	sm.stats.TotalTokensUsed += event.TokensUsed
	sm.stats.LastUse = now

	if sm.stats.FirstUse == "" {
		sm.stats.FirstUse = now
	}

	// Update cache stats (only when cache was actually checked)
	if event.CacheChecked {
		if event.CacheHit {
			sm.stats.CacheHits++
		} else {
			sm.stats.CacheMisses++
		}
	}

	// Update average generation time
	totalTime := sm.stats.AverageGenerationTime * float64(sm.stats.TotalGenerations-1)
	sm.stats.AverageGenerationTime = (totalTime + event.GenerationTime) / float64(sm.stats.TotalGenerations)

	// Update provider-specific stats
	if sm.stats.ProviderStats[event.Provider] == nil {
		sm.stats.ProviderStats[event.Provider] = &types.ProviderStats{
			Name:      event.Provider,
			FirstUsed: now,
		}
	}

	providerStats := sm.stats.ProviderStats[event.Provider]
	providerStats.TotalUses++
	if event.Success {
		providerStats.SuccessfulUses++
	} else {
		providerStats.FailedUses++
	}

	providerStats.TotalCost += event.Cost
	providerStats.TotalTokensUsed += event.TokensUsed
	providerStats.LastUsed = now

	// Update provider average generation time
	totalProviderTime := providerStats.AverageGenerationTime * float64(providerStats.TotalUses-1)
	providerStats.AverageGenerationTime = (totalProviderTime + event.GenerationTime) / float64(providerStats.TotalUses)

	// Calculate success rate
	if providerStats.TotalUses > 0 {
		providerStats.SuccessRate = float64(providerStats.SuccessfulUses) / float64(providerStats.TotalUses) * 100
	}

	// Save to disk
	return sm.save()
}

// GetStats returns a copy of the current usage statistics.
func (sm *StatsManager) GetStats() *types.UsageStats {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Create a deep copy to prevent external modifications
	statsCopy := &types.UsageStats{
		TotalGenerations:      sm.stats.TotalGenerations,
		SuccessfulGenerations: sm.stats.SuccessfulGenerations,
		FailedGenerations:     sm.stats.FailedGenerations,
		FirstUse:              sm.stats.FirstUse,
		LastUse:               sm.stats.LastUse,
		TotalCost:             sm.stats.TotalCost,
		TotalTokensUsed:       sm.stats.TotalTokensUsed,
		CacheHits:             sm.stats.CacheHits,
		CacheMisses:           sm.stats.CacheMisses,
		AverageGenerationTime: sm.stats.AverageGenerationTime,
		ProviderStats:         make(map[types.LLMProvider]*types.ProviderStats),
	}

	// Deep copy provider stats
	for provider, stats := range sm.stats.ProviderStats {
		statsCopy.ProviderStats[provider] = &types.ProviderStats{
			Name:                  stats.Name,
			TotalUses:             stats.TotalUses,
			SuccessfulUses:        stats.SuccessfulUses,
			FailedUses:            stats.FailedUses,
			TotalCost:             stats.TotalCost,
			TotalTokensUsed:       stats.TotalTokensUsed,
			AverageGenerationTime: stats.AverageGenerationTime,
			FirstUsed:             stats.FirstUsed,
			LastUsed:              stats.LastUsed,
			SuccessRate:           stats.SuccessRate,
		}
	}

	return statsCopy
}

// GetMostUsedProvider returns the provider with the highest usage count.
func (sm *StatsManager) GetMostUsedProvider() (types.LLMProvider, int) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var mostUsed types.LLMProvider
	maxUses := 0

	for provider, stats := range sm.stats.ProviderStats {
		if stats.TotalUses > maxUses {
			maxUses = stats.TotalUses
			mostUsed = provider
		}
	}

	return mostUsed, maxUses
}

// GetSuccessRate returns the overall success rate as a percentage.
func (sm *StatsManager) GetSuccessRate() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if sm.stats.TotalGenerations == 0 {
		return 0.0
	}

	return float64(sm.stats.SuccessfulGenerations) / float64(sm.stats.TotalGenerations) * 100
}

// GetCacheHitRate returns the cache hit rate as a percentage.
func (sm *StatsManager) GetCacheHitRate() float64 {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	totalCacheAttempts := sm.stats.CacheHits + sm.stats.CacheMisses
	if totalCacheAttempts == 0 {
		return 0.0
	}

	return float64(sm.stats.CacheHits) / float64(totalCacheAttempts) * 100
}

// ResetStats clears all usage statistics.
func (sm *StatsManager) ResetStats() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.stats = &types.UsageStats{
		ProviderStats: make(map[types.LLMProvider]*types.ProviderStats),
	}

	return sm.save()
}

// load reads statistics from the persistent storage file.
func (sm *StatsManager) load() error {
	data, err := os.ReadFile(sm.filePath)
	if err != nil {
		return err
	}

	if len(data) == 0 {
		return nil // Empty file is okay
	}

	return json.Unmarshal(data, sm.stats)
}

// save writes current statistics to the persistent storage file.
func (sm *StatsManager) save() error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(sm.filePath), 0755); err != nil {
		return fmt.Errorf("failed to create stats directory: %w", err)
	}

	data, err := json.MarshalIndent(sm.stats, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	return os.WriteFile(sm.filePath, data, 0600)
}

// GetProviderRanking returns providers ranked by usage count.
func (sm *StatsManager) GetProviderRanking() []types.LLMProvider {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	type providerUsage struct {
		provider types.LLMProvider
		uses     int
	}

	var rankings []providerUsage
	for provider, stats := range sm.stats.ProviderStats {
		rankings = append(rankings, providerUsage{
			provider: provider,
			uses:     stats.TotalUses,
		})
	}

	// Sort by usage count (descending)
	for i := 0; i < len(rankings)-1; i++ {
		for j := i + 1; j < len(rankings); j++ {
			if rankings[j].uses > rankings[i].uses {
				rankings[i], rankings[j] = rankings[j], rankings[i]
			}
		}
	}

	result := make([]types.LLMProvider, len(rankings))
	for i, r := range rankings {
		result[i] = r.provider
	}

	return result
}