package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/dfanso/commit-msg/cmd/cli/store"
	"github.com/dfanso/commit-msg/pkg/types"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// statsCmd represents the statistics command
var statsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Display usage statistics",
	Long: `Display comprehensive usage statistics including:
- Most used LLM provider
- Average generation time
- Success/failure rates 
- Token usage per provider
- Cache hit rates
- Cost tracking`,
	RunE: func(cmd *cobra.Command, args []string) error {
		Store, err := store.NewStoreMethods()
		if err != nil {
			return fmt.Errorf("failed to initialize store: %w", err)
		}

		reset, _ := cmd.Flags().GetBool("reset")
		if reset {
			if err := resetStatistics(Store); err != nil {
				return err
			}
			return nil
		}

		detailed, _ := cmd.Flags().GetBool("detailed")
		return displayStatistics(Store, detailed)
	},
}

func init() {
	statsCmd.Flags().Bool("detailed", false, "Show detailed per-provider statistics")
	statsCmd.Flags().Bool("reset", false, "Reset all usage statistics")
}

func displayStatistics(store *store.StoreMethods, detailed bool) error {
	stats := store.GetUsageStats()
	
	if stats.TotalGenerations == 0 {
		pterm.Info.Println("No usage statistics available yet.")
		pterm.Info.Println("Statistics will be collected as you use the commit message generator.")
		return nil
	}

	// Header
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgWhite, pterm.Bold)).
		Println("Usage Statistics")

	pterm.Println()

	// Overall Statistics
	pterm.DefaultSection.WithLevel(2).Println("Overall Statistics")
	
	overallData := [][]string{
		{"Total Generations", fmt.Sprintf("%d", stats.TotalGenerations)},
		{"Successful Generations", fmt.Sprintf("%d (%.1f%%)", stats.SuccessfulGenerations, store.GetOverallSuccessRate())},
		{"Failed Generations", fmt.Sprintf("%d (%.1f%%)", stats.FailedGenerations, float64(stats.FailedGenerations)/float64(stats.TotalGenerations)*100)},
		{"Average Generation Time", fmt.Sprintf("%.1f ms", stats.AverageGenerationTime)},
		{"Total Cost", fmt.Sprintf("$%.4f", stats.TotalCost)},
		{"Total Tokens Used", fmt.Sprintf("%d", stats.TotalTokensUsed)},
	}

	if stats.CacheHits > 0 || stats.CacheMisses > 0 {
		cacheRate := store.GetCacheHitRate()
		overallData = append(overallData, []string{"Cache Hit Rate", fmt.Sprintf("%.1f%% (%d hits, %d misses)", cacheRate, stats.CacheHits, stats.CacheMisses)})
	}

	if stats.FirstUse != "" {
		if firstUse, err := time.Parse(time.RFC3339, stats.FirstUse); err == nil {
			overallData = append(overallData, []string{"First Use", firstUse.Local().Format("Jan 2, 2006 15:04")})
		}
	}

	if stats.LastUse != "" {
		if lastUse, err := time.Parse(time.RFC3339, stats.LastUse); err == nil {
			overallData = append(overallData, []string{"Last Use", lastUse.Local().Format("Jan 2, 2006 15:04")})
		}
	}

	pterm.DefaultTable.WithHasHeader(false).WithData(overallData).Render()
	pterm.Println()

	// Provider Rankings
	if len(stats.ProviderStats) > 0 {
		pterm.DefaultSection.WithLevel(2).Println("Provider Rankings")
		
		ranking := store.GetProviderRanking()
		rankingData := [][]string{{"Rank", "Provider", "Uses", "Success Rate", "Avg Time (ms)", "Total Cost"}}
		
		for i, provider := range ranking {
			providerStats := stats.ProviderStats[provider]
			rankingData = append(rankingData, []string{
				fmt.Sprintf("#%d", i+1),
				string(provider),
				fmt.Sprintf("%d", providerStats.TotalUses),
				fmt.Sprintf("%.1f%%", providerStats.SuccessRate),
				fmt.Sprintf("%.1f", providerStats.AverageGenerationTime),
				fmt.Sprintf("$%.4f", providerStats.TotalCost),
			})
		}

		pterm.DefaultTable.WithHasHeader(true).WithData(rankingData).Render()
		pterm.Println()
	}

	// Detailed Provider Statistics
	if detailed && len(stats.ProviderStats) > 0 {
		pterm.DefaultSection.WithLevel(2).Println("Detailed Provider Statistics")
		
		// Sort providers alphabetically for consistent display
		var providers []types.LLMProvider
		for provider := range stats.ProviderStats {
			providers = append(providers, provider)
		}
		sort.Slice(providers, func(i, j int) bool {
			return string(providers[i]) < string(providers[j])
		})

		for _, provider := range providers {
			providerStats := stats.ProviderStats[provider]
			
			pterm.DefaultSection.WithLevel(3).Printf("%s Details", provider)
			
			providerData := [][]string{
				{"Total Uses", fmt.Sprintf("%d", providerStats.TotalUses)},
				{"Successful Uses", fmt.Sprintf("%d", providerStats.SuccessfulUses)},
				{"Failed Uses", fmt.Sprintf("%d", providerStats.FailedUses)},
				{"Success Rate", fmt.Sprintf("%.1f%%", providerStats.SuccessRate)},
				{"Average Generation Time", fmt.Sprintf("%.1f ms", providerStats.AverageGenerationTime)},
				{"Total Cost", fmt.Sprintf("$%.4f", providerStats.TotalCost)},
				{"Total Tokens Used", fmt.Sprintf("%d", providerStats.TotalTokensUsed)},
			}

			if providerStats.FirstUsed != "" {
				if firstUsed, err := time.Parse(time.RFC3339, providerStats.FirstUsed); err == nil {
					providerData = append(providerData, []string{"First Used", firstUsed.Local().Format("Jan 2, 2006 15:04")})
				}
			}

			if providerStats.LastUsed != "" {
				if lastUsed, err := time.Parse(time.RFC3339, providerStats.LastUsed); err == nil {
					providerData = append(providerData, []string{"Last Used", lastUsed.Local().Format("Jan 2, 2006 15:04")})
				}
			}

			pterm.DefaultTable.WithHasHeader(false).WithData(providerData).Render()
			pterm.Println()
		}
	}

	// Show tips
	pterm.DefaultSection.WithLevel(2).Println("Tips")
	pterm.Info.Println("• Use --detailed flag to see comprehensive per-provider statistics")
	pterm.Info.Println("• Statistics help identify your most reliable and cost-effective providers")
	pterm.Info.Println("• Cache hits save both time and API costs")
	pterm.Info.Println("• Use --reset flag to clear all statistics (irreversible)")

	return nil
}

func resetStatistics(store *store.StoreMethods) error {
	pterm.Warning.Println("This will permanently delete all usage statistics.")
	
	confirm, _ := pterm.DefaultInteractiveConfirm.
		WithDefaultValue(false).
		WithDefaultText("Are you sure you want to reset all statistics?").
		Show()

	if !confirm {
		pterm.Info.Println("Statistics reset cancelled.")
		return nil
	}

	if err := store.ResetUsageStats(); err != nil {
		return fmt.Errorf("failed to reset statistics: %w", err)
	}

	pterm.Success.Println("All usage statistics have been reset.")
	return nil
}