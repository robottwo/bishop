package wizard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Provider represents an LLM provider configuration
type Provider struct {
	ID               string
	Name             string
	Desc             string
	DefaultBaseURL   string
	DefaultModel     string
	RequiresAPIKey   bool
	DefaultAPIKey    string // For providers like Ollama that don't need a key
	SupportsStreaming bool
}

// ProviderConfig holds the user's configuration for a specific provider
type ProviderConfig struct {
	Provider       *Provider
	APIKey         string
	BaseURL        string
	ModelID        string
	Validated      bool
	ValidationMsg  string
}

// AllProviders returns the complete list of supported providers (alphabetically sorted)
var AllProviders = []Provider{
	{
		ID:                "bedrock",
		Name:              "AWS Bedrock",
		Desc:       "AWS Bedrock managed AI service",
		DefaultBaseURL:    "https://bedrock-runtime.us-east-1.amazonaws.com",
		DefaultModel:      "anthropic.claude-3-sonnet-20240229-v1:0",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "anthropic",
		Name:              "Anthropic (Claude)",
		Desc:       "Claude models from Anthropic",
		DefaultBaseURL:    "https://api.anthropic.com/v1",
		DefaultModel:      "claude-3-5-sonnet-20241022",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "azure-openai",
		Name:              "Azure OpenAI",
		Desc:       "OpenAI models via Azure",
		DefaultBaseURL:    "https://YOUR-RESOURCE.openai.azure.com",
		DefaultModel:      "gpt-4",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "cohere",
		Name:              "Cohere",
		Desc:       "Cohere language models",
		DefaultBaseURL:    "https://api.cohere.ai/v1",
		DefaultModel:      "command-r-plus",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "deepinfra",
		Name:              "DeepInfra",
		Desc:       "Hosted AI inference platform",
		DefaultBaseURL:    "https://api.deepinfra.com/v1/openai",
		DefaultModel:      "meta-llama/Meta-Llama-3.1-70B-Instruct",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "deepseek",
		Name:              "DeepSeek",
		Desc:       "DeepSeek language models",
		DefaultBaseURL:    "https://api.deepseek.com/v1",
		DefaultModel:      "deepseek-chat",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "fireworks",
		Name:              "Fireworks AI",
		Desc:       "Fast inference platform",
		DefaultBaseURL:    "https://api.fireworks.ai/inference/v1",
		DefaultModel:      "accounts/fireworks/models/llama-v3p1-70b-instruct",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "google",
		Name:              "Google (Gemini)",
		Desc:       "Google Gemini models",
		DefaultBaseURL:    "https://generativelanguage.googleapis.com/v1beta",
		DefaultModel:      "gemini-1.5-pro",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "grok",
		Name:              "Grok (xAI)",
		Desc:       "xAI's Grok models",
		DefaultBaseURL:    "https://api.x.ai/v1",
		DefaultModel:      "grok-beta",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "groq",
		Name:              "Groq",
		Desc:       "Ultra-fast LLM inference",
		DefaultBaseURL:    "https://api.groq.com/openai/v1",
		DefaultModel:      "llama-3.1-70b-versatile",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "huggingface",
		Name:              "Hugging Face Inference",
		Desc:       "Hugging Face hosted models",
		DefaultBaseURL:    "https://api-inference.huggingface.co/models",
		DefaultModel:      "meta-llama/Meta-Llama-3-70B-Instruct",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "mistral",
		Name:              "Mistral AI",
		Desc:       "Mistral language models",
		DefaultBaseURL:    "https://api.mistral.ai/v1",
		DefaultModel:      "mistral-large-latest",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "moonshot",
		Name:              "Moonshot AI (Kimi)",
		Desc:       "Moonshot's Kimi models",
		DefaultBaseURL:    "https://api.moonshot.cn/v1",
		DefaultModel:      "moonshot-v1-8k",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "ollama",
		Name:              "Ollama",
		Desc:       "Local LLM runtime",
		DefaultBaseURL:    "http://localhost:11434/v1",
		DefaultModel:      "qwen2.5:32b",
		RequiresAPIKey:    false,
		DefaultAPIKey:     "ollama", // Placeholder
		SupportsStreaming: true,
	},
	{
		ID:                "openai",
		Name:              "OpenAI",
		Desc:       "GPT models from OpenAI",
		DefaultBaseURL:    "https://api.openai.com/v1",
		DefaultModel:      "gpt-4",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "openrouter",
		Name:              "OpenRouter",
		Desc:       "Unified API for multiple providers",
		DefaultBaseURL:    "https://openrouter.ai/api/v1",
		DefaultModel:      "anthropic/claude-3-opus",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "perplexity",
		Name:              "Perplexity",
		Desc:       "Perplexity AI models",
		DefaultBaseURL:    "https://api.perplexity.ai",
		DefaultModel:      "llama-3.1-sonar-large-128k-online",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "replicate",
		Name:              "Replicate",
		Desc:       "Cloud platform for ML models",
		DefaultBaseURL:    "https://api.replicate.com/v1",
		DefaultModel:      "meta/meta-llama-3-70b-instruct",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "together",
		Name:              "Together AI",
		Desc:       "Fast inference on open models",
		DefaultBaseURL:    "https://api.together.xyz/v1",
		DefaultModel:      "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
	{
		ID:                "zai",
		Name:              "ZAI",
		Desc:       "ZAI language models",
		DefaultBaseURL:    "https://api.zai.com/v1",
		DefaultModel:      "zai-chat",
		RequiresAPIKey:    true,
		SupportsStreaming: true,
	},
}

// ValidateProvider performs a lightweight validation check for a provider
// Returns (success bool, message string)
func ValidateProvider(ctx context.Context, config *ProviderConfig) (bool, string) {
	// For Ollama, check if the server is reachable
	if config.Provider.ID == "ollama" {
		return validateOllama(ctx, config.BaseURL)
	}

	// For other providers, do a basic API key format check
	if config.Provider.RequiresAPIKey && strings.TrimSpace(config.APIKey) == "" {
		return false, "API key is required"
	}

	// Try a simple connectivity check for providers with public endpoints
	if strings.HasPrefix(config.BaseURL, "http") {
		return validateHTTPEndpoint(ctx, config.BaseURL)
	}

	return true, "Configuration saved (validation skipped for custom endpoint)"
}

func validateOllama(ctx context.Context, baseURL string) (bool, string) {
	client := &http.Client{Timeout: 3 * time.Second}
	
	// Try to list models endpoint
	url := strings.TrimSuffix(baseURL, "/v1") + "/api/tags"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Sprintf("Invalid URL: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("Cannot reach Ollama at %s. Is it running?", baseURL)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("Ollama returned status %d", resp.StatusCode)
	}

	// Try to parse response to ensure it's valid
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, "Invalid response from Ollama"
	}

	return true, "✓ Ollama is running and accessible"
}

func validateHTTPEndpoint(ctx context.Context, baseURL string) (bool, string) {
	client := &http.Client{Timeout: 5 * time.Second}
	
	req, err := http.NewRequestWithContext(ctx, "GET", baseURL, nil)
	if err != nil {
		return false, fmt.Sprintf("Invalid URL: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		// Don't fail - endpoint might require auth or specific paths
		return true, "Configuration saved (endpoint connectivity check failed, but this may be expected)"
	}
	defer resp.Body.Close()

	return true, fmt.Sprintf("✓ Endpoint is reachable (HTTP %d)", resp.StatusCode)
}

// GetProviderByID returns a provider by its ID
func GetProviderByID(id string) *Provider {
	for i := range AllProviders {
		if AllProviders[i].ID == id {
			return &AllProviders[i]
		}
	}
	return nil
}
