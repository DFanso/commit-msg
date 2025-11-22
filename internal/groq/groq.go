package groq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dfanso/commit-msg/cmd/cli/store"
	internalHTTP "github.com/dfanso/commit-msg/internal/http"
	"github.com/dfanso/commit-msg/pkg/types"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatResponse struct {
	Choices []chatChoice `json:"choices"`
}

// defaultModel uses Groq's recommended general-purpose model as of Oct 2025.
// If Groq updates their defaults again, override via GROQ_MODEL.
const defaultModel = "llama-3.3-70b-versatile"

const (
	groqTemperature         = 0.2
	groqMaxTokens           = 200
	groqSystemMessage       = "You are an assistant that writes clear, concise git commit messages."
	groqContentType         = "application/json"
	groqAuthorizationPrefix = "Bearer "
)

var (
	// allow overrides in tests
	baseURL = "https://api.groq.com/openai/v1/chat/completions"
	// httpClient can be overridden in tests; defaults to the internal http client
	httpClient   *http.Client
	storeMethods *store.StoreMethods
)

func init() {
	httpClient = internalHTTP.GetClient()
}

// GenerateCommitMessage calls Groq's OpenAI-compatible chat completions API.
func GenerateCommitMessage(_ *types.Config, changes string, apiKey string, opts *types.GenerationOptions) (string, error) {
	if changes == "" {
		return "", fmt.Errorf("no changes provided for commit message generation")
	}

	customTemplate, _ := storeMethods.GetTemplate()
	prompt := types.BuildCommitPromptWithTemplate(changes, opts, customTemplate)

	model := os.Getenv("GROQ_MODEL")
	if model == "" {
		model = defaultModel
	}

	payload := chatRequest{
		Model:       model,
		Temperature: groqTemperature,
		MaxTokens:   groqMaxTokens,
		Messages: []chatMessage{
			{Role: "system", Content: groqSystemMessage},
			{Role: "user", Content: prompt},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal Groq request: %w", err)
	}

	endpoint := baseURL
	if customEndpoint := os.Getenv("GROQ_API_URL"); customEndpoint != "" {
		endpoint = customEndpoint
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to create Groq request: %w", err)
	}

	req.Header.Set("Content-Type", groqContentType)
	req.Header.Set("Authorization", fmt.Sprintf("%s%s", groqAuthorizationPrefix, apiKey))

	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call Groq API: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read Groq response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, string(responseBody))
	}

	var completion chatResponse
	if err := json.Unmarshal(responseBody, &completion); err != nil {
		return "", fmt.Errorf("failed to decode Groq response: %w", err)
	}

	if len(completion.Choices) == 0 || completion.Choices[0].Message.Content == "" {
		return "", fmt.Errorf("groq API returned empty response")
	}

	return completion.Choices[0].Message.Content, nil
}
