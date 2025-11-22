package gemini

import (
	"context"
	"fmt"

	"github.com/dfanso/commit-msg/cmd/cli/store"
	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/dfanso/commit-msg/pkg/types"
)

const (
	geminiModel       = "gemini-2.0-flash"
	geminiTemperature = 0.2
)

var storeMethods *store.StoreMethods

// GenerateCommitMessage asks Google Gemini to author a commit message for the
// supplied repository changes and optional style instructions.
func GenerateCommitMessage(config *types.Config, changes string, apiKey string, opts *types.GenerationOptions) (string, error) {
	// Prepare request to Gemini API
	customTemplate, _ := storeMethods.GetTemplate()
	prompt := types.BuildCommitPromptWithTemplate(changes, opts, customTemplate)

	// Create context and client
	ctx := context.Background()
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a GenerativeModel with appropriate settings
	model := client.GenerativeModel(geminiModel)
	model.SetTemperature(geminiTemperature) // Lower temperature for more focused responses

	// Generate content using the prompt
	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return "", err
	}

	// Check if we got a valid response
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	// Extract the commit message from the response
	commitMsg := fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0])

	return commitMsg, nil
}
