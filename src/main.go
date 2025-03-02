package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/dfanso/commit-msg/src/config"
	"github.com/dfanso/commit-msg/src/grok"
	"github.com/dfanso/commit-msg/src/types"
	"github.com/joho/godotenv"
)

// Normalize path to handle both forward and backslashes
func normalizePath(path string) string {
	// Replace backslashes with forward slashes
	normalized := strings.ReplaceAll(path, "\\", "/")
	// Remove any trailing slash
	normalized = strings.TrimSuffix(normalized, "/")
	return normalized
}

// Main function
func main() {
	// Define command line flags
	configFile := flag.String("config", "commit-helper.json", "Path to config file")
	setupMode := flag.Bool("setup", false, "Setup configuration")
	path := flag.String("path", "", "Path to monitor (for setup)")
	repoName := flag.String("name", "", "Repository name (for setup)")
	apiEndpoint := flag.String("api-endpoint", "https://api.x.ai/v1/chat/completions", "X.AI API endpoint (for setup)")
	flag.Parse()

	// Load environment variables from .env file
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	// Get API key from environment variables
	apiKey := os.Getenv("GROK_API_KEY")
	if apiKey == "" {
		log.Fatalf("GROK_API_KEY not found in environment variables")
	}

	// If in setup mode, create a new configuration
	if *setupMode {
		if *path == "" || *repoName == "" {
			log.Fatalf("Both --path and --name are required in setup mode")
		}

		// Normalize the path
		normalizedPath := normalizePath(*path)

		// Create new config with default endpoint
		newConfig := &types.Config{
			GrokAPI: *apiEndpoint,
			APIKey:  apiKey, // Store API key in config
			Repos:   make(map[string]types.RepoConfig),
		}

		newConfig.Repos[*repoName] = types.RepoConfig{
			Path:    normalizedPath,
			LastRun: time.Now().Format(time.RFC3339),
		}

		if err := config.SaveConfig(*configFile, newConfig); err != nil {
			log.Fatalf("Failed to save configuration: %v", err)
		}

		fmt.Printf("Repository '%s' configured successfully.\n", *repoName)
		return
	}

	// Initialize or load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Make sure the API key is in the config
	if cfg.APIKey == "" {
		cfg.APIKey = apiKey
		if err := config.SaveConfig(*configFile, cfg); err != nil {
			log.Printf("Warning: Failed to update API key in config: %v", err)
		}
	}

	// If no path provided, list available repositories
	if *path == "" {
		fmt.Println("Available repositories:")
		if len(cfg.Repos) == 0 {
			fmt.Println("No repositories configured. Use --setup to add a repository.")
			return
		}
		for name, repo := range cfg.Repos {
			fmt.Printf("- %s: %s (Last run: %s)\n", name, repo.Path, repo.LastRun)
		}
		return
	}

	// Validate configuration
	if cfg.GrokAPI == "" {
		cfg.GrokAPI = *apiEndpoint // Use default endpoint if not set
		if err := config.SaveConfig(*configFile, cfg); err != nil {
			log.Printf("Warning: Failed to update API endpoint in config: %v", err)
		}
	}

	// Normalize the input path
	repoPath := normalizePath(*path)
	var selectedRepo string
	var repoConfig types.RepoConfig

	// Find the repository configuration by path
	for name, repo := range cfg.Repos {
		// Normalize the stored path for comparison
		storedPath := normalizePath(repo.Path)
		if storedPath == repoPath {
			selectedRepo = name
			repoConfig = repo
			break
		}
	}

	if selectedRepo == "" {
		log.Fatalf("No repository configured for path: %s", repoPath)
	}

	// Ensure the path is a Git repository
	if !isGitRepository(repoConfig.Path) {
		log.Fatalf("The specified path is not a Git repository: %s", repoConfig.Path)
	}

	// Get the changes
	changes, err := getGitChanges(&repoConfig)
	if err != nil {
		log.Fatalf("Failed to get Git changes: %v", err)
	}

	if len(changes) == 0 {
		fmt.Println("No changes detected in the Git repository.")
		return
	}

	// Generate commit message using X.AI API
	commitMsg, err := grok.GenerateCommitMessage(cfg, changes)
	if err != nil {
		log.Fatalf("Failed to generate commit message: %v", err)
	}

	// Display the commit message
	fmt.Println("Suggested commit message:")
	fmt.Println(commitMsg)

	// Update the last run time
	repoConfig.LastRun = time.Now().Format(time.RFC3339)
	cfg.Repos[selectedRepo] = repoConfig
	if err := config.SaveConfig(*configFile, cfg); err != nil {
		log.Printf("Warning: Failed to update last run time: %v", err)
	}
}

// Check if directory is a git repository
func isGitRepository(path string) bool {
	cmd := exec.Command("git", "-C", path, "rev-parse", "--is-inside-work-tree")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) == "true"
}

// Get changes using Git
func getGitChanges(config *types.RepoConfig) (string, error) {
	var changes strings.Builder

	// 1. Check for unstaged changes
	cmd := exec.Command("git", "-C", config.Path, "diff", "--name-status")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff failed: %v", err)
	}

	if len(output) > 0 {
		changes.WriteString("Unstaged changes:\n")
		changes.WriteString(string(output))
		changes.WriteString("\n\n")

		// Get the content of these changes
		diffCmd := exec.Command("git", "-C", config.Path, "diff")
		diffOutput, err := diffCmd.Output()
		if err != nil {
			return "", fmt.Errorf("git diff content failed: %v", err)
		}

		changes.WriteString("Unstaged diff content:\n")
		changes.WriteString(string(diffOutput))
		changes.WriteString("\n\n")
	}

	// 2. Check for staged changes
	stagedCmd := exec.Command("git", "-C", config.Path, "diff", "--name-status", "--cached")
	stagedOutput, err := stagedCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git diff --cached failed: %v", err)
	}

	if len(stagedOutput) > 0 {
		changes.WriteString("Staged changes:\n")
		changes.WriteString(string(stagedOutput))
		changes.WriteString("\n\n")

		// Get the content of these changes
		stagedDiffCmd := exec.Command("git", "-C", config.Path, "diff", "--cached")
		stagedDiffOutput, err := stagedDiffCmd.Output()
		if err != nil {
			return "", fmt.Errorf("git diff --cached content failed: %v", err)
		}

		changes.WriteString("Staged diff content:\n")
		changes.WriteString(string(stagedDiffOutput))
		changes.WriteString("\n\n")
	}

	// 3. Check for untracked files
	untrackedCmd := exec.Command("git", "-C", config.Path, "ls-files", "--others", "--exclude-standard")
	untrackedOutput, err := untrackedCmd.Output()
	if err != nil {
		return "", fmt.Errorf("git ls-files failed: %v", err)
	}

	if len(untrackedOutput) > 0 {
		changes.WriteString("Untracked files:\n")
		changes.WriteString(string(untrackedOutput))
		changes.WriteString("\n\n")

		// Try to get content of untracked files (limited to text files and smaller size)
		untrackedFiles := strings.Split(strings.TrimSpace(string(untrackedOutput)), "\n")
		for _, file := range untrackedFiles {
			if file == "" {
				continue
			}

			fullPath := filepath.Join(config.Path, file)
			if isTextFile(fullPath) && isSmallFile(fullPath) {
				fileContent, err := os.ReadFile(fullPath)
				if err == nil {
					changes.WriteString(fmt.Sprintf("Content of new file %s:\n", file))
					changes.WriteString(string(fileContent))
					changes.WriteString("\n\n")
				}
			}
		}
	}

	// 4. Get recent commits for context
	recentCommitsCmd := exec.Command("git", "-C", config.Path, "log", "--oneline", "-n", "3")
	recentCommitsOutput, err := recentCommitsCmd.Output()
	if err == nil && len(recentCommitsOutput) > 0 {
		changes.WriteString("Recent commits for context:\n")
		changes.WriteString(string(recentCommitsOutput))
		changes.WriteString("\n")
	}

	return changes.String(), nil
}

// Check if a file is likely to be a text file
func isTextFile(filename string) bool {
	// List of common text file extensions
	textExtensions := []string{".txt", ".md", ".go", ".js", ".py", ".java", ".c", ".cpp", ".h", ".html", ".css", ".json", ".xml", ".yaml", ".yml", ".sh", ".bash", ".ts", ".tsx", ".jsx", ".php", ".rb", ".rs", ".dart"}

	ext := strings.ToLower(filepath.Ext(filename))
	for _, textExt := range textExtensions {
		if ext == textExt {
			return true
		}
	}

	return false
}

// Check if a file is small enough to include in context
func isSmallFile(filename string) bool {
	const maxSize = 10 * 1024 // 10KB max

	info, err := os.Stat(filename)
	if err != nil {
		return false
	}

	return info.Size() <= maxSize
}
