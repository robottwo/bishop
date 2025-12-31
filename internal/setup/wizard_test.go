package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFirstRun(t *testing.T) {
	// Save original home dir
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bishop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Test case 1: No config files exist
	os.Setenv("HOME", tmpDir)
	if !IsFirstRun() {
		t.Error("Expected IsFirstRun() to return true when no config files exist")
	}

	// Test case 2: .bishrc exists but no provider settings
	bishrcPath := filepath.Join(tmpDir, ".bishrc")
	err = os.WriteFile(bishrcPath, []byte("# Just a comment\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if !IsFirstRun() {
		t.Error("Expected IsFirstRun() to return true when .bishrc exists but has no provider settings")
	}

	// Test case 3: .bishenv exists with provider settings
	bishenvPath := filepath.Join(tmpDir, ".bishenv")
	err = os.WriteFile(bishenvPath, []byte("export BISH_SLOW_MODEL_PROVIDER='openai'\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if IsFirstRun() {
		t.Error("Expected IsFirstRun() to return false when .bishenv has provider settings")
	}
}

func TestSaveConfiguration(t *testing.T) {
	// Save original home dir
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bishop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)

	// Test saving configuration
	result := WizardResult{
		Provider: "openai",
		APIKey:   "sk-test-key",
		BaseURL:  "https://api.openai.com/v1",
		ModelID:  "gpt-4",
	}

	err = SaveConfiguration(result)
	if err != nil {
		t.Fatalf("SaveConfiguration failed: %v", err)
	}

	// Verify the file was created
	bishenvPath := filepath.Join(tmpDir, ".bishenv")
	content, err := os.ReadFile(bishenvPath)
	if err != nil {
		t.Fatalf("Failed to read .bishenv: %v", err)
	}

	// Check that the content contains expected values
	contentStr := string(content)
	expectedStrings := []string{
		"BISH_FAST_MODEL_PROVIDER='openai'",
		"BISH_FAST_MODEL_API_KEY='sk-test-key'",
		"BISH_SLOW_MODEL_PROVIDER='openai'",
		"BISH_SLOW_MODEL_ID='gpt-4'",
	}

	for _, expected := range expectedStrings {
		if !contains(contentStr, expected) {
			t.Errorf("Expected .bishenv to contain %q", expected)
		}
	}

	// Verify file permissions
	info, err := os.Stat(bishenvPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("Expected file permissions to be 0600, got %o", info.Mode().Perm())
	}
}

func TestSaveConfiguration_Skipped(t *testing.T) {
	// Save original home dir
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bishop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	os.Setenv("HOME", tmpDir)

	// Test skipped configuration
	result := WizardResult{
		Skipped: true,
	}

	err = SaveConfiguration(result)
	if err != nil {
		t.Fatalf("SaveConfiguration failed: %v", err)
	}

	// Verify no file was created
	bishenvPath := filepath.Join(tmpDir, ".bishenv")
	if _, err := os.Stat(bishenvPath); !os.IsNotExist(err) {
		t.Error("Expected .bishenv not to be created when wizard is skipped")
	}
}

func TestEscapeShellValue(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simple", "simple"},
		{"with'quote", "with'\\''quote"},
		{"multiple'quotes'here", "multiple'\\''quotes'\\''here"},
	}

	for _, tt := range tests {
		result := escapeShellValue(tt.input)
		if result != tt.expected {
			t.Errorf("escapeShellValue(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestProviders(t *testing.T) {
	// Verify we have the expected providers
	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}

	expectedIDs := []string{"ollama", "openai", "openrouter"}
	for i, expectedID := range expectedIDs {
		if providers[i].ID != expectedID {
			t.Errorf("Expected provider %d to have ID %q, got %q", i, expectedID, providers[i].ID)
		}
	}

	// Verify Ollama doesn't need API key
	if providers[0].NeedsAPIKey {
		t.Error("Expected Ollama provider to not need API key")
	}

	// Verify OpenAI and OpenRouter need API keys
	if !providers[1].NeedsAPIKey {
		t.Error("Expected OpenAI provider to need API key")
	}
	if !providers[2].NeedsAPIKey {
		t.Error("Expected OpenRouter provider to need API key")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
