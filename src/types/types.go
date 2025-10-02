package types

import "context"

// LLM abstracts the LLM api communication.
type LLM interface {
	// GenerateCommitMessage generates a commit message based on the changes.
	GenerateCommitMessage(ctx context.Context, changes string) (string, error)
}

// LLMType designates a specific LLM provider.
type LLMType string

const (
	Grok   LLMType = "grok"
	Google LLMType = "google"
)

// Configuration structure
type Config struct {
	GrokAPI string                `json:"grok_api"`
	Repos   map[string]RepoConfig `json:"repos"`
}

// Repository configuration
type RepoConfig struct {
	Path    string `json:"path"`
	LastRun string `json:"last_run"`
}

// Grok/X.AI API request structure
type GrokRequest struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Grok/X.AI API response structure
type GrokResponse struct {
	Message Message   `json:"message,omitempty"`
	Choices []Choice  `json:"choices,omitempty"`
	Id      string    `json:"id,omitempty"`
	Object  string    `json:"object,omitempty"`
	Created int64     `json:"created,omitempty"`
	Model   string    `json:"model,omitempty"`
	Usage   UsageInfo `json:"usage,omitempty"`
}

type Choice struct {
	Message      Message `json:"message"`
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
}

type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
