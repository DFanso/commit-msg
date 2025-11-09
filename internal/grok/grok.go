package grok

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dfanso/commit-msg/cmd/cli/store"
	httpClient "github.com/dfanso/commit-msg/internal/http"
	"github.com/dfanso/commit-msg/pkg/types"
)

const (
	grokModel           = "grok-3-mini-fast-beta"
	grokTemperature     = 0
	grokAPIEndpoint     = "https://api.x.ai/v1/chat/completions"
	grokContentType     = "application/json"
	authorizationPrefix = "Bearer "
)

var storeMethods *store.StoreMethods

// GenerateCommitMessage calls X.AI's Grok API to create a commit message from
// the provided Git diff and generation options.
func GenerateCommitMessage(config *types.Config, changes string, apiKey string, opts *types.GenerationOptions) (string, error) {
	// Prepare request to X.AI (Grok) API
	customTemplate, _ := storeMethods.GetTemplate()
	prompt := types.BuildCommitPromptWithTemplate(changes, opts, customTemplate)

	request := types.GrokRequest{

		Messages: []types.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Model:       grokModel,
		Stream:      false,
		Temperature: grokTemperature,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", grokAPIEndpoint, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Content-Type", grokContentType)
	req.Header.Set("Authorization", fmt.Sprintf("%s%s", authorizationPrefix, apiKey))

	client := httpClient.GetClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Parse response
	var grokResponse types.GrokResponse
	if err := json.NewDecoder(resp.Body).Decode(&grokResponse); err != nil {
		return "", err
	}

	// Check if the response follows the expected structure
	if grokResponse.Message.Content == "" && grokResponse.Choices != nil && len(grokResponse.Choices) > 0 {
		return grokResponse.Choices[0].Message.Content, nil
	}

	return grokResponse.Message.Content, nil
}
