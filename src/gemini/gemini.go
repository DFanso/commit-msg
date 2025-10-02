package gemini

import (
	"context"
	"fmt"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"

	"github.com/dfanso/commit-msg/src/types"
)

type Service struct {
	apiKey string
	config *types.Config
}

func NewGeminiLLM(apiKey string, config *types.Config) (types.LLM, error) {
	return Service{
		apiKey: apiKey,
		config: config,
	}, nil
}

func (s Service) GenerateCommitMessage(ctx context.Context, changes string) (string, error) {
	// Prepare request to Google API
	prompt := fmt.Sprintf("%s\n\n%s", types.CommitPrompt, changes)

	// Create context and client
	client, err := genai.NewClient(ctx, option.WithAPIKey(s.apiKey))
	if err != nil {
		return "", err
	}
	defer client.Close()

	// Create a GenerativeModel with appropriate settings
	model := client.GenerativeModel("gemini-2.0-flash")
	model.SetTemperature(0.2) // Lower temperature for more focused responses

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
