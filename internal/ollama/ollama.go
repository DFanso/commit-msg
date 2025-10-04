package ollama

import (
	"fmt"
	"net/http"
	"encoding/json"
	"bytes"
	"io"

	"github.com/dfanso/commit-msg/pkg/types"

)

type OllamaRequest struct {
	Model string `json:"model"`
	Prompt string `json:"prompt"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done bool `json:"done"`
}

func GenerateCommitMessage(config *types.Config, changes string, url string, model string) (string, error) {
	// Use llama3:latest as the default model
	if model == "" {
		model = "llama3:latest" 
	}

	// Preparing the prompt
	prompt := fmt.Sprintf("%s\n\n%s", types.CommitPrompt, changes)

	// Generating the request body - add stream: false for non-streaming response
	reqBody := map[string]interface{}{
		"model":  model,
		"prompt": prompt,
		"stream": false,
	}
	
	// Generating the body
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(body))
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