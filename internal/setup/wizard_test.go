package setup

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestIsFirstRun(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bishop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the home dir override for testing
	SetHomeDirForTesting(tmpDir)
	defer SetHomeDirForTesting("") // Reset after test

	// Test case 1: No config files exist
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

	// Test case 4: .bishrc has provider settings (no .bishenv)
	os.Remove(bishenvPath)
	err = os.WriteFile(bishrcPath, []byte("export BISH_FAST_MODEL_PROVIDER='ollama'\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if IsFirstRun() {
		t.Error("Expected IsFirstRun() to return false when .bishrc has provider settings")
	}

	// Test case 5: .bish_config_ui has provider settings
	os.Remove(bishenvPath)
	err = os.WriteFile(bishrcPath, []byte("# Just a comment\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	configUIPath := filepath.Join(tmpDir, ".bish_config_ui")
	err = os.WriteFile(configUIPath, []byte("export BISH_SLOW_MODEL_PROVIDER='openrouter'\n"), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if IsFirstRun() {
		t.Error("Expected IsFirstRun() to return false when .bish_config_ui has provider settings")
	}
}

func TestSaveConfiguration(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bishop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the home dir override for testing
	SetHomeDirForTesting(tmpDir)
	defer SetHomeDirForTesting("") // Reset after test

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
		if !strings.Contains(contentStr, expected) {
			t.Errorf("Expected .bishenv to contain %q, got:\n%s", expected, contentStr)
		}
	}

	// Verify file permissions (skip on Windows as it doesn't support Unix permissions)
	if runtime.GOOS != "windows" {
		info, err := os.Stat(bishenvPath)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0600 {
			t.Errorf("Expected file permissions to be 0600, got %o", info.Mode().Perm())
		}
	}
}

func TestSaveConfiguration_Skipped(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bishop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the home dir override for testing
	SetHomeDirForTesting(tmpDir)
	defer SetHomeDirForTesting("") // Reset after test

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

func TestSaveConfiguration_EscapesAPIKey(t *testing.T) {
	// Create temp directory for test
	tmpDir, err := os.MkdirTemp("", "bishop-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Set the home dir override for testing
	SetHomeDirForTesting(tmpDir)
	defer SetHomeDirForTesting("") // Reset after test

	// Test saving configuration with special characters in API key
	result := WizardResult{
		Provider: "openai",
		APIKey:   "sk-test'key'with'quotes",
		BaseURL:  "https://api.openai.com/v1",
		ModelID:  "gpt-4",
	}

	err = SaveConfiguration(result)
	if err != nil {
		t.Fatalf("SaveConfiguration failed: %v", err)
	}

	// Verify the file contains escaped quotes
	bishenvPath := filepath.Join(tmpDir, ".bishenv")
	content, err := os.ReadFile(bishenvPath)
	if err != nil {
		t.Fatalf("Failed to read .bishenv: %v", err)
	}

	// The single quotes in the API key should be escaped
	if !strings.Contains(string(content), `sk-test'\''key'\''with'\''quotes`) {
		t.Errorf("Expected API key to be properly escaped, got:\n%s", string(content))
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
		{"", ""},
		{"no special chars", "no special chars"},
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

	// Verify default URLs are set
	for _, p := range providers {
		if p.DefaultURL == "" {
			t.Errorf("Provider %q has empty DefaultURL", p.ID)
		}
	}
}

func TestGetHomeDir(t *testing.T) {
	// Test with override
	SetHomeDirForTesting("/test/path")
	dir, err := getHomeDir()
	if err != nil {
		t.Fatalf("getHomeDir failed: %v", err)
	}
	if dir != "/test/path" {
		t.Errorf("Expected /test/path, got %s", dir)
	}

	// Test without override
	SetHomeDirForTesting("")
	dir, err = getHomeDir()
	if err != nil {
		t.Fatalf("getHomeDir failed: %v", err)
	}
	if dir == "" {
		t.Error("Expected non-empty home directory")
	}
}

func TestWizardResult(t *testing.T) {
	// Test skipped result
	result := WizardResult{Skipped: true}
	if !result.Skipped {
		t.Error("Expected Skipped to be true")
	}

	// Test normal result
	result = WizardResult{
		Provider: "openai",
		APIKey:   "sk-test",
		BaseURL:  "https://api.openai.com/v1",
		ModelID:  "gpt-4",
	}
	if result.Skipped {
		t.Error("Expected Skipped to be false")
	}
	if result.Provider != "openai" {
		t.Errorf("Expected Provider to be openai, got %s", result.Provider)
	}
}
