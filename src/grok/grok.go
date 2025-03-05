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
	prompt := fmt.Sprintf("%s\n\n%s", types.CommitPrompt, changes)

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
	req, err := http.NewRequest("POST", "https://api.x.ai/v1/chat/completions", bytes.NewBuffer(requestBody))
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
