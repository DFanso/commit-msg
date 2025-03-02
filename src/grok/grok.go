package grok

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/dfanso/commit-msg/src/types"
)

func GenerateCommitMessage(config *types.Config, changes string, apiKey string) (string, error) {
	// Prepare request to X.AI (Grok) API
	prompt := fmt.Sprintf(`
I need a concise git commit message based on the following changes from my Git repository.
Please generate a commit message that:
1. Starts with a verb in the present tense (e.g., "Add", "Fix", "Update", "Feat", "Refactor", etc.)
2. Is clear and descriptive
3. Focuses on the "what" and "why" of the changes
4. Is no longer than 50-72 characters for the first line
5. Can include a more detailed description after a blank line if needed
6. Dont say any other stuff only include the commit msg

here is a sample commit msgs:
'feat: improve performance with lazy load implementation for images'
Here are the changes:

%s
`, changes)

	request := types.GrokRequest{
		Messages: []types.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Model:       "grok-2-latest",
		Stream:      false,
		Temperature: 0,
	}

	requestBody, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", config.GrokAPI, bytes.NewBuffer(requestBody))
	if err != nil {
		return "", err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	// Configure HTTP client with improved TLS settings
	transport := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
		MaxIdleConns:        10,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  true,
		// Add TLS config to handle server name mismatch
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false, // Keep this false for security
		},
	}

	client := &http.Client{
		Timeout:   30 * time.Second,
		Transport: transport,
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := ioutil.ReadAll(resp.Body)
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
