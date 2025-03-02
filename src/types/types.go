package types

// Configuration structure
type Config struct {
	Path    string `json:"path"`
	GrokAPI string `json:"grok_api"`
	APIKey  string `json:"api_key"`
	LastRun string `json:"last_run"`
}

// Grok API request structure
type GrokRequest struct {
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Grok API response structure
type GrokResponse struct {
	Message Message `json:"message"`
}
