package wizard

import (
	"context"
	"fmt"
	"strings"

	"github.com/sashabaranov/go-openai"
)

func validateAPIKeyFormat(apiKey, provider string) error {
	if apiKey == "" {
		return fmt.Errorf("API key cannot be empty")
	}

	switch provider {
	case "openai":
		if !strings.HasPrefix(apiKey, "sk-") {
			return fmt.Errorf("OpenAI API keys must start with 'sk-'")
		}
		if len(apiKey) < 20 {
			return fmt.Errorf("API key appears to be too short")
		}
	case "openrouter":
		if !strings.HasPrefix(apiKey, "sk-or-") {
			return fmt.Errorf("OpenRouter API keys must start with 'sk-or-'")
		}
		if len(apiKey) < 30 {
			return fmt.Errorf("API key appears to be too short")
		}
	case "ollama":
		if apiKey != "" && apiKey != "ollama" {
			return fmt.Errorf("ollama typically doesn't require an API key")
		}
	default:
		return fmt.Errorf("unknown provider: %s", provider)
	}

	return nil
}

func testConnection(config modelConfig) (bool, error) {
	if config.provider == "ollama" {
		if config.apiKey == "" {
			config.apiKey = "ollama"
		}
		if config.baseURL == "" {
			config.baseURL = "http://localhost:11434/v1/"
		}
	}

	clientConfig := openai.DefaultConfig(config.apiKey)
	clientConfig.BaseURL = config.baseURL

	client := openai.NewClientWithConfig(clientConfig)

	ctx := context.Background()

	_, err := client.ListModels(ctx)
	if err != nil {
		return false, fmt.Errorf("connection test failed: %w", err)
	}

	return true, nil
}
