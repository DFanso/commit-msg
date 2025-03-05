package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dfanso/commit-msg/src/gemini"
	"github.com/dfanso/commit-msg/src/grok"
	"github.com/dfanso/commit-msg/src/types"
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
	// Get API key from environment variables
	var apiKey string
	if os.Getenv("COMMIT_LLM") == "google" {
		apiKey = os.Getenv("GOOGLE_API_KEY")
		if apiKey == "" {
			log.Fatalf("GOOGLE_API_KEY is not set")
		}
	} else if os.Getenv("COMMIT_LLM") == "grok" {
		apiKey = os.Getenv("GROK_API_KEY")
		if apiKey == "" {
			log.Fatalf("GROK_API_KEY is not set")
		}
	} else {
		log.Fatalf("Invalid COMMIT_LLM value: %s", os.Getenv("COMMIT_LLM"))
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Check if current directory is a git repository
	if !isGitRepository(currentDir) {
		log.Fatalf("Current directory is not a Git repository: %s", currentDir)
	}

	// Create a minimal config for the API
	config := &types.Config{
		GrokAPI: "https://api.x.ai/v1/chat/completions",
	}

	// Create a repo config for the current directory
	repoConfig := types.RepoConfig{
		Path: currentDir,
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

	// Pass API key to GenerateCommitMessage
	var commitMsg string
	if os.Getenv("COMMIT_LLM") == "google" {
		commitMsg, err = gemini.GenerateCommitMessage(config, changes, apiKey)
	} else {
		commitMsg, err = grok.GenerateCommitMessage(config, changes, apiKey)
	}
	if err != nil {
		log.Fatalf("Failed to generate commit message: %v", err)
	}

	// Display the commit message
	fmt.Println(commitMsg)
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
