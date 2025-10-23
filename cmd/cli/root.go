package cmd

import (
	"os"

	"github.com/dfanso/commit-msg/cmd/cli/store"
	"github.com/spf13/cobra"
)

// store instance
var Store *store.StoreMethods

// Initailize store
func StoreInit(sm *store.StoreMethods) {
	Store = sm
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "commit",
	Short: "CLI tool to write commit message",
	Long:  `Write a commit message with AI of your choice`,
	Example: `
	# Generate a commit message and run the interactive review flow
	commit .

	# Preview what would be sent to the LLM without making an API call
	commit . --dry-run

	# Generate a commit message and automatically commit it
	commit . --auto

	# Show verbose debug information (diff stats, full prompts, repository details)
	commit . --toggle
	commit . --dry-run --toggle
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var llmCmd = &cobra.Command{
	Use:   "llm",
	Short: "Manage LLM configuration",
}

var llmSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup your LLM provider and API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		return SetupLLM(Store)
	},
}

var llmUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update or Delete LLM Model",
	RunE: func(cmd *cobra.Command, args []string) error {
		return UpdateLLM(Store)
	},
}

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage commit message cache",
	Long:  `Manage the cache of generated commit messages to reduce API costs and improve performance.`,
}

var cacheStatsCmd = &cobra.Command{
	Use:   "stats",
	Short: "Show cache statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		return ShowCacheStats(Store)
	},
}

var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cached messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		return ClearCache(Store)
	},
}

var cacheCleanupCmd = &cobra.Command{
	Use:   "cleanup",
	Short: "Remove old cached messages",
	RunE: func(cmd *cobra.Command, args []string) error {
		return CleanupCache(Store)
	},
}

var creatCommitMsg = &cobra.Command{
	Use:   ".",
	Short: "Create Commit Message",
	RunE: func(cmd *cobra.Command, args []string) error {
		dryRun, err := cmd.Flags().GetBool("dry-run")
		if err != nil {
			return err
		}

		autoCommit, err := cmd.Flags().GetBool("auto")
		if err != nil {
			return err
		}

		verbose, err := cmd.Flags().GetBool("toggle")
		if err != nil {
			return err
		}

		CreateCommitMsg(Store, dryRun, autoCommit, verbose)
		return nil
	},
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.commit-msg.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.

	// Add --dry-run, --auto, and --toggle as persistent flags so they show in top-level help
	rootCmd.PersistentFlags().Bool("dry-run", false, "Preview the prompt that would be sent to the LLM without making an API call")
	rootCmd.PersistentFlags().Bool("auto", false, "Automatically commit with the generated message")
	rootCmd.PersistentFlags().BoolP("toggle", "t", false, "Show verbose debug information (diff stats, full prompts, repository details)")

	rootCmd.AddCommand(creatCommitMsg)
	rootCmd.AddCommand(llmCmd)
	rootCmd.AddCommand(cacheCmd)
	llmCmd.AddCommand(llmSetupCmd)
	llmCmd.AddCommand(llmUpdateCmd)
	cacheCmd.AddCommand(cacheStatsCmd)
	cacheCmd.AddCommand(cacheClearCmd)
	cacheCmd.AddCommand(cacheCleanupCmd)
}
