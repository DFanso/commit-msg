package types

// LLMProvider identifies the large language model backend used to author
// commit messages.
type LLMProvider string

const (
	ProviderOpenAI LLMProvider = "OpenAI"
	ProviderClaude LLMProvider = "Claude"
	ProviderGemini LLMProvider = "Gemini"
	ProviderGrok   LLMProvider = "Grok"
	ProviderGroq   LLMProvider = "Groq"
	ProviderOllama LLMProvider = "Ollama"
)

// String returns the provider identifier as a plain string.
func (p LLMProvider) String() string {
	return string(p)
}

// IsValid reports whether the provider is part of the supported set.
func (p LLMProvider) IsValid() bool {
	switch p {
	case ProviderOpenAI, ProviderClaude, ProviderGemini, ProviderGrok, ProviderGroq, ProviderOllama:
		return true
	default:
		return false
	}
}

// GetSupportedProviders returns all available provider enums.
func GetSupportedProviders() []LLMProvider {
	return []LLMProvider{
		ProviderOpenAI,
		ProviderClaude,
		ProviderGemini,
		ProviderGrok,
		ProviderGroq,
		ProviderOllama,
	}
}

// GetSupportedProviderStrings returns the human-friendly names for providers.
func GetSupportedProviderStrings() []string {
	providers := GetSupportedProviders()
	strings := make([]string, len(providers))
	for i, provider := range providers {
		strings[i] = provider.String()
	}
	return strings
}

// ParseLLMProvider converts a string into an LLMProvider enum when supported.
func ParseLLMProvider(s string) (LLMProvider, bool) {
	provider := LLMProvider(s)
	return provider, provider.IsValid()
}

// Config stores CLI-level configuration including named repositories.
type Config struct {
	GrokAPI string                `json:"grok_api"`
	Repos   map[string]RepoConfig `json:"repos"`
}

// RepoConfig tracks metadata for a configured Git repository.
type RepoConfig struct {
	Path    string `json:"path"`
	LastRun string `json:"last_run"`
}

// GrokRequest represents a chat completion request sent to X.AI's API.
type GrokRequest struct {
	Messages    []Message `json:"messages"`
	Model       string    `json:"model"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
}

// Message captures the role/content pairs exchanged with Grok.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// GrokResponse contains the relevant fields parsed from X.AI responses.
type GrokResponse struct {
	Message Message   `json:"message,omitempty"`
	Choices []Choice  `json:"choices,omitempty"`
	Id      string    `json:"id,omitempty"`
	Object  string    `json:"object,omitempty"`
	Created int64     `json:"created,omitempty"`
	Model   string    `json:"model,omitempty"`
	Usage   UsageInfo `json:"usage,omitempty"`
}

// Choice details a single response option returned by Grok.
type Choice struct {
	Message      Message `json:"message"`
	Index        int     `json:"index"`
	FinishReason string  `json:"finish_reason"`
}

// UsageInfo reports token usage statistics from Grok responses.
type UsageInfo struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CacheEntry represents a cached commit message with metadata.
type CacheEntry struct {
	Message          string      `json:"message"`
	Provider         LLMProvider `json:"provider"`
	DiffHash         string      `json:"diff_hash"`
	StyleInstruction string      `json:"style_instruction,omitempty"`
	Attempt          int         `json:"attempt"`
	CreatedAt        string      `json:"created_at"`
	LastAccessedAt   string      `json:"last_accessed_at"`
	AccessCount      int         `json:"access_count"`
	Cost             float64     `json:"cost,omitempty"`
	Tokens           *UsageInfo  `json:"tokens,omitempty"`
}

// CacheStats provides statistics about the cache.
type CacheStats struct {
	TotalEntries   int     `json:"total_entries"`
	TotalHits      int     `json:"total_hits"`
	TotalMisses    int     `json:"total_misses"`
	HitRate        float64 `json:"hit_rate"`
	TotalCostSaved float64 `json:"total_cost_saved"`
	OldestEntry    string  `json:"oldest_entry"`
	NewestEntry    string  `json:"newest_entry"`
	CacheSizeBytes int64   `json:"cache_size_bytes"`
}

// CacheConfig holds configuration for the cache system.
type CacheConfig struct {
	Enabled         bool   `json:"enabled"`
	MaxEntries      int    `json:"max_entries"`
	MaxAgeDays      int    `json:"max_age_days"`
	CleanupInterval int    `json:"cleanup_interval_hours"`
	CacheFilePath   string `json:"cache_file_path"`
}

// UsageStats tracks comprehensive usage statistics for the application.
type UsageStats struct {
	TotalGenerations    int                        `json:"total_generations"`
	SuccessfulGenerations int                      `json:"successful_generations"`
	FailedGenerations   int                        `json:"failed_generations"`
	ProviderStats       map[LLMProvider]*ProviderStats `json:"provider_stats"`
	FirstUse            string                     `json:"first_use"`
	LastUse             string                     `json:"last_use"`
	TotalCost           float64                    `json:"total_cost"`
	TotalTokensUsed     int                        `json:"total_tokens_used"`
	CacheHits           int                        `json:"cache_hits"`
	CacheMisses         int                        `json:"cache_misses"`
	AverageGenerationTime float64                  `json:"average_generation_time_ms"`
}

// ProviderStats tracks statistics for a specific LLM provider.
type ProviderStats struct {
	Name                  LLMProvider `json:"name"`
	TotalUses             int         `json:"total_uses"`
	SuccessfulUses        int         `json:"successful_uses"`
	FailedUses            int         `json:"failed_uses"`
	TotalCost             float64     `json:"total_cost"`
	TotalTokensUsed       int         `json:"total_tokens_used"`
	AverageGenerationTime float64     `json:"average_generation_time_ms"`
	FirstUsed             string      `json:"first_used"`
	LastUsed              string      `json:"last_used"`
	SuccessRate           float64     `json:"success_rate"`
}

// GenerationEvent represents a single commit message generation event for tracking.
type GenerationEvent struct {
	Provider      LLMProvider `json:"provider"`
	Success       bool        `json:"success"`
	GenerationTime float64     `json:"generation_time_ms"`
	TokensUsed    int         `json:"tokens_used"`
	Cost          float64     `json:"cost"`
	CacheHit      bool        `json:"cache_hit"`
	Timestamp     string      `json:"timestamp"`
	ErrorMessage  string      `json:"error_message,omitempty"`
}
