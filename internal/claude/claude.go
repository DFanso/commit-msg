package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/dfanso/commit-msg/cmd/cli/store"
	httpClient "github.com/dfanso/commit-msg/internal/http"
	"github.com/dfanso/commit-msg/pkg/types"
)

const (
	claudeModel            = "claude-3-5-sonnet-20241022"
	claudeMaxTokens        = 200
	claudeAPIEndpoint      = "https://api.anthropic.com/v1/messages"
	claudeAPIVersion       = "2023-06-01"
	contentTypeJSON        = "application/json"
	anthropicVersionHeader = "anthropic-version"
	xAPIKeyHeader          = "x-api-key"
)

// ClaudeRequest describes the payload sent to Anthropic's Claude messages API.
type ClaudeRequest struct {
	Model     string          `json:"model"`
	Messages  []types.Message `json:"messages"`
	MaxTokens int             `json:"max_tokens"`
}

// ClaudeResponse captures the subset of fields used from Anthropic responses.
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

var storeMethods *store.StoreMethods

// GenerateCommitMessage produces a commit summary using Anthropic's Claude API.
func GenerateCommitMessage(config *types.Config, changes string, apiKey string, opts *types.GenerationOptions) (string, error) {

	customTemplate, _ := storeMethods.GetTemplate()
	prompt := types.BuildCommitPromptWithTemplate(changes, opts, customTemplate)

	reqBody := ClaudeRequest{
		Model:     claudeModel,
		MaxTokens: claudeMaxTokens,
		Messages: []types.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	req, err := http.NewRequestWithContext(ctx, "POST", claudeAPIEndpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", contentTypeJSON)
	req.Header.Set(xAPIKeyHeader, apiKey)
	req.Header.Set(anthropicVersionHeader, claudeAPIVersion)

	client := httpClient.GetClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("claude AI response %d", resp.StatusCode)
	}

	var claudeResponse ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResponse); err != nil {
		return "", err
	}

	if len(claudeResponse.Content) == 0 {
		return "", fmt.Errorf("no response generated")
	}

	commitMsg := claudeResponse.Content[0].Text
	return commitMsg, nil
}
