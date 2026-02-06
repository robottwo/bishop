package wizard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAllProvidersCount(t *testing.T) {
	expected := 20
	if len(AllProviders) != expected {
		t.Errorf("Expected %d providers, got %d", expected, len(AllProviders))
	}
}

func TestAllProvidersAlphabetical(t *testing.T) {
	for i := 0; i < len(AllProviders)-1; i++ {
		current := AllProviders[i].Name
		next := AllProviders[i+1].Name
		if current > next {
			t.Errorf("Providers not in alphabetical order: %s comes before %s", current, next)
		}
	}
}

func TestAllProvidersHaveRequiredFields(t *testing.T) {
	for _, p := range AllProviders {
		if p.ID == "" {
			t.Errorf("Provider missing ID: %+v", p)
		}
		if p.Name == "" {
			t.Errorf("Provider %s missing Name", p.ID)
		}
		if p.Desc == "" {
			t.Errorf("Provider %s missing Description", p.ID)
		}
		if p.DefaultBaseURL == "" {
			t.Errorf("Provider %s missing DefaultBaseURL", p.ID)
		}
		if p.DefaultModel == "" {
			t.Errorf("Provider %s missing DefaultModel", p.ID)
		}
		
		// Ollama should not require API key
		if p.ID == "ollama" && p.RequiresAPIKey {
			t.Errorf("Ollama should not require API key")
		}
		
		// Ollama should have DefaultAPIKey set
		if p.ID == "ollama" && p.DefaultAPIKey == "" {
			t.Errorf("Ollama should have DefaultAPIKey set")
		}
	}
}

func TestGetProviderByID(t *testing.T) {
	tests := []struct {
		id       string
		expected bool
	}{
		{"openai", true},
		{"anthropic", true},
		{"ollama", true},
		{"invalid-provider", false},
		{"", false},
	}
	
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			result := GetProviderByID(tt.id)
			if tt.expected && result == nil {
				t.Errorf("Expected to find provider %s, got nil", tt.id)
			}
			if !tt.expected && result != nil {
				t.Errorf("Expected nil for provider %s, got %+v", tt.id, result)
			}
			if tt.expected && result != nil && result.ID != tt.id {
				t.Errorf("Expected provider ID %s, got %s", tt.id, result.ID)
			}
		})
	}
}

func TestValidateProviderAPIKeyRequired(t *testing.T) {
	provider := &Provider{
		ID:             "test",
		Name:           "Test Provider",
		RequiresAPIKey: true,
		DefaultBaseURL: "https://api.test.com",
	}
	
	config := &ProviderConfig{
		Provider: provider,
		APIKey:   "",
		BaseURL:  provider.DefaultBaseURL,
		ModelID:  "test-model",
	}
	
	ctx := context.Background()
	success, msg := ValidateProvider(ctx, config)
	
	if success {
		t.Error("Expected validation to fail with missing API key")
	}
	if msg != "API key is required" {
		t.Errorf("Expected 'API key is required', got '%s'", msg)
	}
}

func TestValidateProviderAPIKeyProvided(t *testing.T) {
	// Mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	provider := &Provider{
		ID:             "test",
		Name:           "Test Provider",
		RequiresAPIKey: true,
		DefaultBaseURL: server.URL,
	}
	
	config := &ProviderConfig{
		Provider: provider,
		APIKey:   "test-key-12345",
		BaseURL:  server.URL,
		ModelID:  "test-model",
	}
	
	ctx := context.Background()
	success, msg := ValidateProvider(ctx, config)
	
	if !success {
		t.Errorf("Expected validation to succeed, got error: %s", msg)
	}
}

func TestValidateOllamaSuccess(t *testing.T) {
	// Mock Ollama server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"models": []}`))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	provider := GetProviderByID("ollama")
	if provider == nil {
		t.Fatal("Ollama provider not found")
	}
	
	config := &ProviderConfig{
		Provider: provider,
		APIKey:   provider.DefaultAPIKey,
		BaseURL:  server.URL + "/v1",
		ModelID:  "qwen2.5:32b",
	}
	
	ctx := context.Background()
	success, msg := ValidateProvider(ctx, config)
	
	if !success {
		t.Errorf("Expected Ollama validation to succeed, got: %s", msg)
	}
	if msg != "✓ Ollama is running and accessible" {
		t.Errorf("Expected success message, got: %s", msg)
	}
}

func TestValidateOllamaUnreachable(t *testing.T) {
	provider := GetProviderByID("ollama")
	if provider == nil {
		t.Fatal("Ollama provider not found")
	}
	
	config := &ProviderConfig{
		Provider: provider,
		APIKey:   provider.DefaultAPIKey,
		BaseURL:  "http://localhost:99999/v1", // Invalid port
		ModelID:  "qwen2.5:32b",
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	success, msg := ValidateProvider(ctx, config)
	
	if success {
		t.Error("Expected Ollama validation to fail for unreachable server")
	}
	if msg == "" {
		t.Error("Expected error message for unreachable Ollama")
	}
}

func TestValidateHTTPEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		expectSuccess  bool
	}{
		{"200 OK", http.StatusOK, true},
		{"404 Not Found", http.StatusNotFound, true}, // Still reachable
		{"500 Server Error", http.StatusInternalServerError, true}, // Still reachable
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()
			
			ctx := context.Background()
			success, _ := validateHTTPEndpoint(ctx, server.URL)
			
			if success != tt.expectSuccess {
				t.Errorf("Expected success=%v for status %d, got %v", tt.expectSuccess, tt.statusCode, success)
			}
		})
	}
}

func TestValidateHTTPEndpointTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second) // Longer than client timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	success, msg := validateHTTPEndpoint(ctx, server.URL)
	
	// Should still return success with a caveat message (endpoint might require auth)
	if !success {
		t.Errorf("Expected success despite timeout, got failure: %s", msg)
	}
}

func TestSpecificProviders(t *testing.T) {
	tests := []struct {
		id             string
		name           string
		requiresAPIKey bool
		hasDefaultKey  bool
	}{
		{"anthropic", "Anthropic (Claude)", true, false},
		{"openai", "OpenAI", true, false},
		{"ollama", "Ollama", false, true},
		{"groq", "Groq", true, false},
		{"deepseek", "DeepSeek", true, false},
		{"grok", "Grok (xAI)", true, false},
		{"moonshot", "Moonshot AI (Kimi)", true, false},
		{"zai", "ZAI", true, false},
	}
	
	for _, tt := range tests {
		t.Run(tt.id, func(t *testing.T) {
			p := GetProviderByID(tt.id)
			if p == nil {
				t.Fatalf("Provider %s not found", tt.id)
			}
			if p.Name != tt.name {
				t.Errorf("Expected name %s, got %s", tt.name, p.Name)
			}
			if p.RequiresAPIKey != tt.requiresAPIKey {
				t.Errorf("Expected RequiresAPIKey=%v, got %v", tt.requiresAPIKey, p.RequiresAPIKey)
			}
			if tt.hasDefaultKey && p.DefaultAPIKey == "" {
				t.Errorf("Expected default API key for %s", tt.id)
			}
		})
	}
}

func TestProviderConfigValidation(t *testing.T) {
	provider := GetProviderByID("openai")
	if provider == nil {
		t.Fatal("OpenAI provider not found")
	}
	
	config := &ProviderConfig{
		Provider:      provider,
		APIKey:        "sk-test123",
		BaseURL:       "https://api.openai.com/v1",
		ModelID:       "gpt-4",
		Validated:     false,
		ValidationMsg: "",
	}
	
	if config.Provider.ID != "openai" {
		t.Error("Provider ID mismatch")
	}
	if config.APIKey == "" {
		t.Error("API key should be set")
	}
	if config.Validated {
		t.Error("Should not be validated yet")
	}
}

func TestAllProvidersHaveUniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	
	for _, p := range AllProviders {
		if seen[p.ID] {
			t.Errorf("Duplicate provider ID: %s", p.ID)
		}
		seen[p.ID] = true
	}
}

func TestAllProvidersHaveValidURLs(t *testing.T) {
	for _, p := range AllProviders {
		if p.DefaultBaseURL == "" {
			t.Errorf("Provider %s has empty DefaultBaseURL", p.ID)
			continue
		}
		
		// Check if URL has scheme
		if !hasScheme(p.DefaultBaseURL) {
			t.Errorf("Provider %s has invalid URL (missing scheme): %s", p.ID, p.DefaultBaseURL)
		}
	}
}

func hasScheme(url string) bool {
	return len(url) > 0 && (url[0:4] == "http" || url[0:5] == "https")
}
