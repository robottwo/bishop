package wizard

import (
	"testing"

	"mvdan.cc/sh/v3/interp"
)

func TestAPIKeyCache(t *testing.T) {
	runner, _ := interp.New()
	model := initialModel(runner)

	if model.config.apiKeyCache == nil {
		t.Error("apiKeyCache should be initialized")
	}

	if len(model.config.apiKeyCache) != 0 {
		t.Error("apiKeyCache should be empty initially")
	}
}

func TestWizardConfigCacheReuse(t *testing.T) {
	config := wizardConfig{
		apiKeyCache: make(map[string]string),
	}

	config.apiKeyCache["openai"] = "sk-test-key"

	if cachedKey, ok := config.apiKeyCache["openai"]; !ok {
		t.Error("Should be able to retrieve cached API key")
	} else if cachedKey != "sk-test-key" {
		t.Errorf("Expected cached key 'sk-test-key', got '%s'", cachedKey)
	}
}
