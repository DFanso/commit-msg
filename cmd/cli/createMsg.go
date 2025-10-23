package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/dfanso/commit-msg/cmd/cli/store"
	"github.com/dfanso/commit-msg/internal/display"
	"github.com/dfanso/commit-msg/internal/git"
	"github.com/dfanso/commit-msg/internal/llm"
	"github.com/dfanso/commit-msg/internal/stats"
	"github.com/dfanso/commit-msg/pkg/types"
	"github.com/google/shlex"
	"github.com/pterm/pterm"
)

// CreateCommitMsg launches the interactive flow for reviewing, regenerating,
// editing, and accepting AI-generated commit messages in the current repo.
// If dryRun is true, it displays the prompt without making an API call.
// If verbose is true, shows detailed diff statistics and processing info.
func CreateCommitMsg(Store *store.StoreMethods, dryRun bool, autoCommit bool, verbose bool) {
	// Validate COMMIT_LLM and required API keys
	useLLM, err := Store.DefaultLLMKey()
	if err != nil {
		pterm.Error.Printf("No LLM configured. Run: commit llm setup\n")
		os.Exit(1)
	}

	commitLLM := useLLM.LLM
	apiKey := useLLM.APIKey

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		pterm.Error.Printf("Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	// Check if current directory is a git repository
	if !git.IsRepository(currentDir) {
		pterm.Error.Printf("Current directory is not a Git repository: %s\n", currentDir)
		os.Exit(1)
	}

	config := &types.Config{
		GrokAPI: "https://api.x.ai/v1/chat/completions",
	}

	repoConfig := types.RepoConfig{Path: currentDir}

	fileStats, err := stats.GetFileStatistics(&repoConfig)
	if err != nil {
		pterm.Error.Printf("Failed to get file statistics: %v\n", err)
		os.Exit(1)
	}

	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgCyan)).
		WithTextStyle(pterm.NewStyle(pterm.FgBlack, pterm.Bold)).
		Println("Commit Message Generator")

	pterm.Println()
	display.ShowFileStatistics(fileStats)

	if verbose {
		pterm.Info.Printf("Repository: %s\n", currentDir)
		pterm.Info.Printf("File summary: %d staged, %d unstaged, %d untracked\n",
			len(fileStats.StagedFiles), len(fileStats.UnstagedFiles), len(fileStats.UntrackedFiles))
	}

	if fileStats.TotalFiles == 0 {
		pterm.Warning.Println("No changes detected in the Git repository.")
		pterm.Info.Println("Tips:")
		pterm.Info.Println("  - Stage your changes with: git add .")
		pterm.Info.Println("  - Check repository status with: git status")
		pterm.Info.Println("  - Make sure you're in the correct Git repository")
		return
	}

	changes, err := git.GetChanges(&repoConfig)
	if err != nil {
		pterm.Error.Printf("Failed to get Git changes: %v\n", err)
		os.Exit(1)
	}

	if len(changes) == 0 {
		pterm.Warning.Println("No changes detected in the Git repository.")
		pterm.Info.Println("Tips:")
		pterm.Info.Println("  - Stage your changes with: git add .")
		pterm.Info.Println("  - Check repository status with: git status")
		pterm.Info.Println("  - Make sure you're in the correct Git repository")
		return
	}

	//  Large diff handling
	const maxDiffChars = 8000 // can change as needed
	const maxDiffLines = 300

	diffLines := strings.Split(changes, "\n")
	diffTooLarge := len(changes) > maxDiffChars || len(diffLines) > maxDiffLines

	if diffTooLarge {
		pterm.Warning.Println("The diff is very large and may exceed the LLM's context window.")
		pterm.Info.Printf("Diff size: %d lines, %d characters.\n", len(diffLines), len(changes))
		pterm.Info.Println("Only the first part of the diff will be used for commit message generation.")

		// Truncate the diff for LLM input, preserving whole lines and UTF-8 safety
		truncatedLines := make([]string, 0, len(diffLines))
		totalChars := 0

		for i, line := range diffLines {
			lineLen := len([]rune(line)) + 1 // +1 for newline, using rune count for UTF-8 safety

			// Stop if we've reached max lines or adding this line would exceed max chars
			if i >= maxDiffLines || (totalChars+lineLen) > maxDiffChars {
				break
			}

			truncatedLines = append(truncatedLines, line)
			totalChars += lineLen
		}

		changes = strings.Join(truncatedLines, "\n")
		actualLineCount := len(truncatedLines)

		pterm.Info.Printf("Truncated diff to %d lines, %d characters.\n", actualLineCount, len(changes))
		pterm.Info.Println("Consider committing smaller changes for more accurate commit messages.")
	} else if verbose {
		pterm.Info.Printf("Diff statistics: %d lines, %d characters (within limits).\n", len(diffLines), len(changes))
		inputTokens := estimateTokens(changes)
		pterm.Info.Printf("Estimated tokens for LLM: %d input tokens.\n", inputTokens)
	}

	// Handle dry-run mode: display what would be sent to LLM without making API call
	if dryRun {
		pterm.Println()
		displayDryRunInfo(commitLLM, config, changes, apiKey, verbose)
		return
	}

	ctx := context.Background()

	providerInstance, err := llm.NewProvider(commitLLM, llm.ProviderOptions{
		Credential: apiKey,
		Config:     config,
	})
	if err != nil {
		displayProviderError(commitLLM, err)
		os.Exit(1)
	}

	pterm.Println()
	spinnerGenerating, err := pterm.DefaultSpinner.
		WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").
		Start("Generating commit message with " + commitLLM.String() + "...")
	if err != nil {
		pterm.Error.Printf("Failed to start spinner: %v\n", err)
		os.Exit(1)
	}

	attempt := 1
	commitMsg, err := generateMessageWithCache(ctx, providerInstance, Store, commitLLM, changes, withAttempt(nil, attempt))
	if err != nil {
		spinnerGenerating.Fail("Failed to generate commit message")
		displayProviderError(commitLLM, err)
		os.Exit(1)
	}

	spinnerGenerating.Success("Commit message generated successfully!")

	currentMessage := strings.TrimSpace(commitMsg)
	validateCommitMessageLength(currentMessage)
	currentStyleLabel := stylePresets[0].Label
	var currentStyleOpts *types.GenerationOptions
	accepted := false
	finalMessage := ""

interactionLoop:
	for {
		pterm.Println()
		display.ShowCommitMessage(currentMessage)

		action, err := promptActionSelection()
		if err != nil {
			pterm.Error.Printf("Failed to read selection: %v\n", err)
			return
		}

		switch action {
		case actionAcceptOption:
			finalMessage = strings.TrimSpace(currentMessage)
			if finalMessage == "" {
				pterm.Warning.Println("Commit message is empty; please edit or regenerate before accepting.")
				continue
			}
			if err := clipboard.WriteAll(finalMessage); err != nil {
				pterm.Warning.Printf("Could not copy to clipboard: %v\n", err)
			} else {
				pterm.Success.Println("Commit message copied to clipboard!")
			}
			accepted = true
			break interactionLoop
		case actionRegenerateOption:
			opts, styleLabel, err := promptStyleSelection(currentStyleLabel, currentStyleOpts)
			if errors.Is(err, errSelectionCancelled) {
				continue
			}
			if err != nil {
				pterm.Error.Printf("Failed to select style: %v\n", err)
				continue
			}
			if styleLabel != "" {
				currentStyleLabel = styleLabel
			}
			currentStyleOpts = opts
			nextAttempt := attempt + 1
			generationOpts := withAttempt(currentStyleOpts, nextAttempt)
			spinner, err := pterm.DefaultSpinner.
				WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").
				Start(fmt.Sprintf("Regenerating commit message (%s)...", currentStyleLabel))
			if err != nil {
				pterm.Error.Printf("Failed to start spinner: %v\n", err)
				continue
			}
			updatedMessage, genErr := generateMessageWithCache(ctx, providerInstance, Store, commitLLM, changes, generationOpts)
			if genErr != nil {
				spinner.Fail("Regeneration failed")
				displayProviderError(commitLLM, genErr)
				continue
			}
			spinner.Success("Commit message regenerated!")
			attempt = nextAttempt
			currentMessage = strings.TrimSpace(updatedMessage)
			validateCommitMessageLength(currentMessage)
		case actionEditOption:
			edited, editErr := editCommitMessage(currentMessage)
			if editErr != nil {
				pterm.Error.Printf("Failed to edit commit message: %v\n", editErr)
				continue
			}
			if strings.TrimSpace(edited) == "" {
				pterm.Warning.Println("Edited commit message is empty; keeping previous message.")
				continue
			}
			currentMessage = strings.TrimSpace(edited)
			validateCommitMessageLength(currentMessage)
		case actionExitOption:
			pterm.Info.Println("Exiting without copying commit message.")
			return
		default:
			pterm.Warning.Printf("Unknown selection: %s\n", action)
		}
	}

	if !accepted {
		return
	}

	pterm.Println()
	display.ShowChangesPreview(fileStats)

	// Auto-commit if flag is set (cross-platform compatible)
	if autoCommit && !dryRun {
		pterm.Println()
		spinner, err := pterm.DefaultSpinner.
			WithSequence("⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏").
			Start("Automatically committing with generated message...")
		if err != nil {
			pterm.Error.Printf("Failed to start spinner: %v\n", err)
			return
		}

		cmd := exec.Command("git", "commit", "-m", finalMessage)
		cmd.Dir = currentDir
		// Ensure git command works across all platforms
		cmd.Env = os.Environ()

		output, err := cmd.CombinedOutput()
		if err != nil {
			spinner.Fail("Commit failed")
			pterm.Error.Printf("Failed to commit: %v\n", err)
			if len(output) > 0 {
				pterm.Error.Println(string(output))
			}
			return
		}

		spinner.Success("Committed successfully!")
		if len(output) > 0 {
			pterm.Info.Println(strings.TrimSpace(string(output)))
		}
	}
}

type styleOption struct {
	Label       string
	Instruction string
}

const (
	actionAcceptOption     = "Accept and copy commit message"
	actionRegenerateOption = "Regenerate with different tone/style"
	actionEditOption       = "Edit message in editor"
	actionExitOption       = "Discard and exit"
	customStyleOption      = "Custom instructions (enter your own)"
	styleBackOption        = "Back to actions"
)

var (
	actionOptions = []string{actionAcceptOption, actionRegenerateOption, actionEditOption, actionExitOption}
	stylePresets  = []styleOption{
		{Label: "Concise conventional (default)", Instruction: ""},
		{Label: "Detailed summary (adds bullet list)", Instruction: "Produce a conventional commit subject line followed by a blank line and bullet points summarizing the key changes."},
		{Label: "Casual tone", Instruction: "Write the commit message in a friendly, conversational tone while still clearly explaining the changes."},
		{Label: "Bug fix emphasis", Instruction: "Highlight the bug being fixed, reference the root cause when possible, and describe the remedy in the body."},
	}
	errSelectionCancelled = errors.New("selection cancelled")
)

// resolveOllamaConfig returns the URL and model for Ollama, using environment variables as fallbacks
func resolveOllamaConfig(apiKey string) (url, model string) {
	url = apiKey
	if strings.TrimSpace(url) == "" {
		url = os.Getenv("OLLAMA_URL")
		if url == "" {
			url = "http://localhost:11434/api/generate"
		}
	}
	model = os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "llama3.1"
	}
	return url, model
}

func generateMessage(ctx context.Context, provider llm.Provider, changes string, opts *types.GenerationOptions) (string, error) {
	return provider.Generate(ctx, changes, opts)
}

// generateMessageWithCache generates a commit message with caching support.
func generateMessageWithCache(ctx context.Context, provider llm.Provider, store *store.StoreMethods, providerType types.LLMProvider, changes string, opts *types.GenerationOptions) (string, error) {
	// Check cache first (only for first attempt to avoid caching regenerations)
	if opts == nil || opts.Attempt <= 1 {
		if cachedEntry, found := store.GetCachedMessage(providerType, changes, opts); found {
			pterm.Info.Printf("Using cached commit message (saved $%.4f)\n", cachedEntry.Cost)
			return cachedEntry.Message, nil
		}
	}

	// Generate new message
	message, err := provider.Generate(ctx, changes, opts)
	if err != nil {
		return "", err
	}

	// Cache the result (only for first attempt)
	if opts == nil || opts.Attempt <= 1 {
		// Estimate cost for caching
		cost := estimateCost(providerType, estimateTokens(types.BuildCommitPrompt(changes, opts)), 100)

		// Store in cache
		if cacheErr := store.SetCachedMessage(providerType, changes, opts, message, cost, nil); cacheErr != nil {
			// Log cache error but don't fail the generation
			fmt.Printf("Warning: Failed to cache message: %v\n", cacheErr)
		}
	}

	return message, nil
}

func promptActionSelection() (string, error) {
	return pterm.DefaultInteractiveSelect.
		WithOptions(actionOptions).
		WithDefaultOption(actionAcceptOption).
		Show()
}

func promptStyleSelection(currentLabel string, currentOpts *types.GenerationOptions) (*types.GenerationOptions, string, error) {
	options := make([]string, 0, len(stylePresets)+3)
	foundCurrent := false
	for _, preset := range stylePresets {
		options = append(options, preset.Label)
		if preset.Label == currentLabel {
			foundCurrent = true
		}
	}
	if currentOpts != nil && currentLabel != "" && !foundCurrent {
		options = append(options, currentLabel)
	}
	options = append(options, customStyleOption, styleBackOption)

	selector := pterm.DefaultInteractiveSelect.WithOptions(options)
	if currentLabel != "" {
		selector = selector.WithDefaultOption(currentLabel)
	}

	choice, err := selector.Show()
	if err != nil {
		return currentOpts, currentLabel, err
	}

	switch choice {
	case styleBackOption:
		return currentOpts, currentLabel, errSelectionCancelled
	case customStyleOption:
		text, err := pterm.DefaultInteractiveTextInput.
			WithDefaultText("Describe the tone or style you're looking for").
			Show()
		if err != nil {
			return currentOpts, currentLabel, err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return currentOpts, currentLabel, errSelectionCancelled
		}
		return &types.GenerationOptions{StyleInstruction: text}, formatCustomStyleLabel(text), nil
	default:
		for _, preset := range stylePresets {
			if choice == preset.Label {
				if strings.TrimSpace(preset.Instruction) == "" {
					return nil, preset.Label, nil
				}
				return &types.GenerationOptions{StyleInstruction: preset.Instruction}, preset.Label, nil
			}
		}
		if currentOpts != nil && choice == currentLabel {
			clone := *currentOpts
			return &clone, currentLabel, nil
		}
	}

	if currentOpts != nil && currentLabel != "" {
		clone := *currentOpts
		return &clone, currentLabel, nil
	}
	return nil, currentLabel, nil
}

func editCommitMessage(initial string) (string, error) {
	command, args, err := resolveEditorCommand()
	if err != nil {
		return "", err
	}

	tmpFile, err := os.CreateTemp("", "commit-msg-*.txt")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(strings.TrimSpace(initial) + "\n"); err != nil {
		tmpFile.Close()
		return "", err
	}

	if err := tmpFile.Close(); err != nil {
		return "", err
	}

	cmdArgs := append(args, tmpFile.Name())
	cmd := exec.Command(command, cmdArgs...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	content, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(content)), nil
}

func resolveEditorCommand() (string, []string, error) {
	candidates := []string{
		os.Getenv("GIT_EDITOR"),
		os.Getenv("VISUAL"),
		os.Getenv("EDITOR"),
	}

	for _, candidate := range candidates {
		candidate = strings.TrimSpace(candidate)
		if candidate == "" {
			continue
		}
		parts, err := shlex.Split(candidate)
		if err != nil {
			return "", nil, fmt.Errorf("failed to parse editor command %q: %w", candidate, err)
		}
		if len(parts) == 0 {
			continue
		}
		return parts[0], parts[1:], nil
	}

	if runtime.GOOS == "windows" {
		return "notepad", nil, nil
	}

	return "nano", nil, nil
}

func formatCustomStyleLabel(instruction string) string {
	trimmed := strings.TrimSpace(instruction)
	runes := []rune(trimmed)
	if len(runes) > 40 {
		return fmt.Sprintf("Custom: %s…", string(runes[:37]))
	}
	return fmt.Sprintf("Custom: %s", trimmed)
}

func withAttempt(styleOpts *types.GenerationOptions, attempt int) *types.GenerationOptions {
	if styleOpts == nil {
		return &types.GenerationOptions{Attempt: attempt}
	}
	clone := *styleOpts
	clone.Attempt = attempt
	return &clone
}

func displayProviderError(provider types.LLMProvider, err error) {
	if errors.Is(err, llm.ErrMissingCredential) {
		displayMissingCredentialHint(provider)
		return
	}

	switch provider {
	case types.ProviderGemini:
		pterm.Error.Printf("Gemini API error: %v. Check your GEMINI_API_KEY environment variable or run: commit llm setup\n", err)
	case types.ProviderOpenAI:
		pterm.Error.Printf("OpenAI API error: %v. Check your OPENAI_API_KEY environment variable or run: commit llm setup\n", err)
	case types.ProviderClaude:
		pterm.Error.Printf("Claude API error: %v. Check your CLAUDE_API_KEY environment variable or run: commit llm setup\n", err)
	case types.ProviderGroq:
		pterm.Error.Printf("Groq API error: %v. Check your GROQ_API_KEY environment variable or run: commit llm setup\n", err)
	case types.ProviderGrok:
		pterm.Error.Printf("Grok API error: %v. Check your GROK_API_KEY environment variable or run: commit llm setup\n", err)
	case types.ProviderOllama:
		pterm.Error.Printf("Ollama error: %v. Verify the Ollama service URL or run: commit llm setup\n", err)
	default:
		pterm.Error.Printf("LLM error: %v\n", err)
	}
}

func displayMissingCredentialHint(provider types.LLMProvider) {
	switch provider {
	case types.ProviderGemini:
		pterm.Error.Println("Gemini requires an API key. Run: commit llm setup or set GEMINI_API_KEY.")
	case types.ProviderOpenAI:
		pterm.Error.Println("OpenAI requires an API key. Run: commit llm setup or set OPENAI_API_KEY.")
	case types.ProviderClaude:
		pterm.Error.Println("Claude requires an API key. Run: commit llm setup or set CLAUDE_API_KEY.")
	case types.ProviderGroq:
		pterm.Error.Println("Groq requires an API key. Run: commit llm setup or set GROQ_API_KEY.")
	case types.ProviderGrok:
		pterm.Error.Println("Grok requires an API key. Run: commit llm setup or set GROK_API_KEY.")
	case types.ProviderOllama:
		pterm.Error.Println("Ollama requires a reachable service URL. Run: commit llm setup or set OLLAMA_URL.")
	default:
		pterm.Error.Printf("%s is missing credentials. Run: commit llm setup.\n", provider)
	}
}

// displayDryRunInfo shows what would be sent to the LLM without making an API call
func displayDryRunInfo(provider types.LLMProvider, config *types.Config, changes string, apiKey string, verbose bool) {
	pterm.DefaultHeader.WithFullWidth().
		WithBackgroundStyle(pterm.NewStyle(pterm.BgBlue)).
		WithTextStyle(pterm.NewStyle(pterm.FgWhite, pterm.Bold)).
		Println("DRY RUN MODE - Preview Only")

	pterm.Println()
	pterm.Info.Println("This is a dry-run. No API call will be made to the LLM provider.")
	pterm.Println()

	// Display provider information
	pterm.DefaultSection.Println("LLM Provider Configuration")
	providerInfo := [][]string{
		{"Provider", provider.String()},
	}

	// Add provider-specific info
	switch provider {
	case types.ProviderOllama:
		url, model := resolveOllamaConfig(apiKey)
		providerInfo = append(providerInfo, []string{"Ollama URL", url})
		providerInfo = append(providerInfo, []string{"Model", model})
	case types.ProviderGrok:
		providerInfo = append(providerInfo, []string{"API Endpoint", config.GrokAPI})
		providerInfo = append(providerInfo, []string{"API Key", maskAPIKey(apiKey)})
	default:
		providerInfo = append(providerInfo, []string{"API Key", maskAPIKey(apiKey)})
	}

	pterm.DefaultTable.WithHasHeader(false).WithData(providerInfo).Render()

	pterm.Println()

	// Build and display the prompt
	opts := &types.GenerationOptions{Attempt: 1}
	prompt := types.BuildCommitPrompt(changes, opts)

	pterm.DefaultSection.Println("Prompt That Would Be Sent")
	pterm.Println()

	// Display prompt in a box
	promptBox := pterm.DefaultBox.
		WithTitle("Full LLM Prompt").
		WithTitleTopCenter().
		WithBoxStyle(pterm.NewStyle(pterm.FgCyan))
	promptBox.Println(prompt)

	pterm.Println()

	// Display changes statistics
	pterm.DefaultSection.Println("Changes Summary")
	linesCount := len(strings.Split(changes, "\n"))
	charsCount := len(changes)
	inputTokens := estimateTokens(prompt)
	// Estimate output tokens (typically 50-200 for commit messages)
	outputTokens := 100
	estimatedCost := estimateCost(provider, inputTokens, outputTokens)
	minTime, maxTime := estimateProcessingTime(provider)

	statsData := [][]string{
		{"Total Lines", fmt.Sprintf("%d", linesCount)},
		{"Total Characters", fmt.Sprintf("%d", charsCount)},
		{"Estimated Input Tokens", fmt.Sprintf("%d", inputTokens)},
		{"Estimated Output Tokens", fmt.Sprintf("%d", outputTokens)},
		{"Estimated Total Tokens", fmt.Sprintf("%d", inputTokens+outputTokens)},
	}

	if provider != types.ProviderOllama {
		statsData = append(statsData, []string{"Estimated Cost", fmt.Sprintf("$%.4f", estimatedCost)})
	}

	statsData = append(statsData, []string{"Estimated Processing Time", fmt.Sprintf("%d-%d seconds", minTime, maxTime)})
	if verbose {
		statsData = append(statsData, []string{"Prompt Length", fmt.Sprintf("%d characters", len(prompt))})
	}

	pterm.DefaultTable.WithHasHeader(false).WithData(statsData).Render()

	pterm.Println()
	pterm.Success.Println("Dry-run complete. To generate actual commit message, run without --dry-run flag.")
	if !verbose {
		pterm.Info.Println("Use --toggle flag to see debug details.")
	}
}

// maskAPIKey masks the API key for display purposes
func maskAPIKey(apiKey string) string {
	if len(apiKey) == 0 {
		return "[NOT SET]"
	}
	// Don't mask URLs (used by Ollama)
	if strings.HasPrefix(apiKey, "http://") || strings.HasPrefix(apiKey, "https://") {
		return apiKey
	}
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}
	// Show first 4 and last 4 characters
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}

// estimateTokens provides a rough estimate of token count (1 token ≈ 4 characters)
func estimateTokens(text string) int {
	return len(text) / 4
}

// estimateCost calculates the estimated cost for a given provider and token count
func estimateCost(provider types.LLMProvider, inputTokens, outputTokens int) float64 {
	// Pricing per 1M tokens (as of 2024, approximate)
	switch provider {
	case types.ProviderOpenAI:
		// GPT-4o pricing: ~$2.50/M input, ~$10/M output
		return float64(inputTokens)*2.50/1000000 + float64(outputTokens)*10.00/1000000
	case types.ProviderClaude:
		// Claude pricing: ~$3/M input, ~$15/M output
		return float64(inputTokens)*3.00/1000000 + float64(outputTokens)*15.00/1000000
	case types.ProviderGemini:
		// Gemini pricing: ~$0.15/M input, ~$0.60/M output
		return float64(inputTokens)*0.15/1000000 + float64(outputTokens)*0.60/1000000
	case types.ProviderGrok:
		// Grok pricing: ~$5/M input, ~$15/M output
		return float64(inputTokens)*5.00/1000000 + float64(outputTokens)*15.00/1000000
	case types.ProviderGroq:
		// Groq pricing: similar to OpenAI ~$2.50/M input, ~$10/M output
		return float64(inputTokens)*2.50/1000000 + float64(outputTokens)*10.00/1000000
	case types.ProviderOllama:
		// Local model - no cost
		return 0.0
	default:
		return 0.0
	}
}

// estimateProcessingTime returns estimated processing time in seconds for a provider
func estimateProcessingTime(provider types.LLMProvider) (minTime, maxTime int) {
	switch provider {
	case types.ProviderOllama:
		// Local models take longer
		return 10, 30
	case types.ProviderOpenAI, types.ProviderClaude, types.ProviderGemini, types.ProviderGrok, types.ProviderGroq:
		// Cloud providers are faster
		return 5, 15
	default:
		return 5, 15
	}
}

// validateCommitMessageLength checks if the commit message exceeds recommended length limits
// and displays appropriate warnings
func validateCommitMessageLength(message string) {
	if message == "" {
		return
	}

	lines := strings.Split(message, "\n")
	if len(lines) == 0 {
		return
	}

	subjectLine := strings.TrimSpace(lines[0])
	subjectLength := len(subjectLine)

	// Git recommends subject lines be 50 characters or less, but allows up to 72
	const maxRecommendedLength = 50
	const maxAllowedLength = 72

	if subjectLength > maxAllowedLength {
		pterm.Warning.Printf("Commit message subject line is %d characters (exceeds %d character limit)\n", subjectLength, maxAllowedLength)
		pterm.Info.Println("Consider shortening the subject line for better readability")
	} else if subjectLength > maxRecommendedLength {
		pterm.Warning.Printf("Commit message subject line is %d characters (recommended limit is %d)\n", subjectLength, maxRecommendedLength)
	}
}
