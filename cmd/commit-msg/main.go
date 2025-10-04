package main

import (
	"log"
	"os"

	"github.com/atotto/clipboard"
	"github.com/dfanso/commit-msg/internal/chatgpt"
	"github.com/dfanso/commit-msg/internal/claude"
	"github.com/dfanso/commit-msg/internal/display"
	"github.com/dfanso/commit-msg/internal/gemini"
	"github.com/dfanso/commit-msg/internal/git"
	"github.com/dfanso/commit-msg/internal/grok"
	"github.com/dfanso/commit-msg/internal/ollama"
	"github.com/dfanso/commit-msg/internal/stats"
	"github.com/dfanso/commit-msg/pkg/types"
	"github.com/pterm/pterm"
	"github.com/joho/godotenv"
)

// main is the entry point of the commit message generator
func main() {
    // Validate COMMIT_LLM and required API keys
    commitLLM := os.Getenv("COMMIT_LLM")
    var apiKey string

    switch commitLLM {
    case "gemini":
        apiKey = os.Getenv("GEMINI_API_KEY")
        if apiKey == "" {
            log.Fatalf("GEMINI_API_KEY is not set")
        }
    case "grok":
        apiKey = os.Getenv("GROK_API_KEY")
        if apiKey == "" {
            log.Fatalf("GROK_API_KEY is not set")
        }
    case "chatgpt":
        apiKey = os.Getenv("OPENAI_API_KEY")
        if apiKey == "" {
            log.Fatalf("OPENAI_API_KEY is not set")
        }
    case "claude":
        apiKey = os.Getenv("CLAUDE_API_KEY")
        if apiKey == "" {
            log.Fatalf("CLAUDE_API_KEY is not set")
        }
	case "ollama":
		// No API key needed for an llm running locally on ollama
		apiKey = ""
    default:
        log.Fatalf("Invalid COMMIT_LLM value: %s", commitLLM)
    }

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get current directory: %v", err)
	}

	// Check if current directory is a git repository
	if !git.IsRepository(currentDir) {
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

	// Get file statistics before fetching changes
	fileStats, err := stats.GetFileStatistics(&repoConfig)
	if err != nil {
		log.Fatalf("Failed to get file statistics: %v", err)
	}

	// Display header
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgDarkGray)).
		WithTextStyle(pterm.NewStyle(pterm.FgLightWhite)).
		Println("🚀 Commit Message Generator")

	pterm.Println()

	// Display file statistics with icons
	display.ShowFileStatistics(fileStats)

	if fileStats.TotalFiles == 0 {
		pterm.Warning.Println("No changes detected in the Git repository.")
		return
	}

	// Get the changes
	changes, err := git.GetChanges(&repoConfig)
	if err != nil {
		log.Fatalf("Failed to get Git changes: %v", err)
	}

	if len(changes) == 0 {
		pterm.Warning.Println("No changes detected in the Git repository.")
		return
	}

	pterm.Println()

	// Show generating spinner
	spinnerGenerating, err := pterm.DefaultSpinner.
		WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").
		Start("🤖 Generating commit message...")
	if err != nil {
		log.Fatalf("Failed to start spinner: %v", err)
	}

	var commitMsg string
	if os.Getenv("COMMIT_LLM") == "gemini" {
		commitMsg, err = gemini.GenerateCommitMessage(config, changes, apiKey)
	} else if os.Getenv("COMMIT_LLM") == "chatgpt" {
		commitMsg, err = chatgpt.GenerateCommitMessage(config, changes, apiKey)
	} else if os.Getenv("COMMIT_LLM") == "claude" {
		commitMsg, err = claude.GenerateCommitMessage(config, changes, apiKey)
	} else if os.Getenv("COMMIT_LLM") == "ollama" {
		url := os.Getenv("OLLAMA_URL")
		if url == "" {
			url = "http://localhost:11434/api/generate"
		}
		model := os.Getenv("OLLAMA_MODEL")
		if model == "" {
			model = "llama3:latest" 
		}
		commitMsg, err = ollama.GenerateCommitMessage(config, changes, url, model)
	} else {
		commitMsg, err = grok.GenerateCommitMessage(config, changes, apiKey)
	}

	if err != nil {
		spinnerGenerating.Fail("Failed to generate commit message")
		log.Fatalf("Error: %v", err)
	}

	spinnerGenerating.Success("✅ Commit message generated successfully!")

	pterm.Println()

	// Display the commit message in a styled panel
	display.ShowCommitMessage(commitMsg)

	// Copy to clipboard
	err = clipboard.WriteAll(commitMsg)
	if err != nil {
		pterm.Warning.Printf("⚠️  Could not copy to clipboard: %v\n", err)
	} else {
		pterm.Success.Println("📋 Commit message copied to clipboard!")
	}

	pterm.Println()

	// Display changes preview
	display.ShowChangesPreview(fileStats)
}
