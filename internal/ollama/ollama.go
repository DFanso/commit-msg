package ollama

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
	ollamaDefaultModel = "llama3:latest"
	ollamaStream       = false
	ollamaContentType  = "application/json"
)

// OllamaRequest captures the prompt payload sent to an Ollama HTTP endpoint.
type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

// OllamaResponse represents the non-streaming response from Ollama.
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

var storeMethods *store.StoreMethods

// GenerateCommitMessage uses a locally hosted Ollama model to draft a commit
// message from repository changes and optional style guidance.
func GenerateCommitMessage(_ *types.Config, changes string, url string, model string, opts *types.GenerationOptions) (string, error) {
	// Use llama3:latest as the default model
	if model == "" {
		model = ollamaDefaultModel
	}

	// Checking for the custom template, when building the prompt
	customTemplate, _ := storeMethods.GetTemplate()

	// Preparing the prompt
	prompt := types.BuildCommitPromptWithTemplate(changes, opts, customTemplate)

	// Generating the request body - add stream: false for non-streaming response
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": ollamaStream,
	}

	// Generating the body
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", ollamaContentType)

	resp, err := httpClient.GetOllamaClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to Ollama: %v", err)
	}
	defer resp.Body.Close()

	// Read the full response body for better error handling
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	// Check HTTP status
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	// Since we set stream: false, we get a single response object
	var response OllamaResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", fmt.Errorf("failed to decode response: %v", err)
	}

	// Check if we got any response
	if response.Response == "" {
		return "", fmt.Errorf("received empty response from Ollama")
	}

	return response.Response, nil
}
