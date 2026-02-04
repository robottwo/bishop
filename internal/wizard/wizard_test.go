package wizard

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInitialModel(t *testing.T) {
	items := make([]list.Item, len(AllProviders))
	for i, p := range AllProviders {
		items[i] = providerItem(p)
	}
	
	m := model{
		state:     stateWelcome,
		modelType: "fast",
	}
	
	if m.state != stateWelcome {
		t.Errorf("Expected initial state to be stateWelcome, got %v", m.state)
	}
	if m.modelType != "fast" {
		t.Errorf("Expected model type 'fast', got %s", m.modelType)
	}
}

func TestProviderItemInterface(t *testing.T) {
	p := AllProviders[0]
	item := providerItem(p)
	
	if item.Title() != p.Name {
		t.Errorf("Expected Title() to return %s, got %s", p.Name, item.Title())
	}
	if item.Description() != p.Desc {
		t.Errorf("Expected Description() to return %s, got %s", p.Desc, item.Description())
	}
	if item.FilterValue() != p.Name {
		t.Errorf("Expected FilterValue() to return %s, got %s", p.Name, item.FilterValue())
	}
}

func TestSummaryItemInterface(t *testing.T) {
	item := summaryItem{
		key:   "API Key",
		value: "sk-test...",
	}
	
	expected := "API Key: sk-test..."
	if item.Title() != expected {
		t.Errorf("Expected Title() to return %s, got %s", expected, item.Title())
	}
	if item.Description() != "" {
		t.Errorf("Expected empty Description(), got %s", item.Description())
	}
	if item.FilterValue() != "API Key" {
		t.Errorf("Expected FilterValue() to return 'API Key', got %s", item.FilterValue())
	}
}

func TestGetConfigSteps(t *testing.T) {
	tests := []struct {
		name           string
		providerID     string
		expectedSteps  int
		hasAPIKeyStep  bool
	}{
		{"OpenAI (requires key)", "openai", 3, true},
		{"Ollama (no key)", "ollama", 2, false},
		{"Anthropic (requires key)", "anthropic", 3, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := GetProviderByID(tt.providerID)
			if provider == nil {
				t.Fatalf("Provider %s not found", tt.providerID)
			}
			
			m := model{
				selectedProvider: provider,
				config: &ProviderConfig{
					Provider: provider,
				},
			}
			
			steps := m.getConfigSteps()
			if len(steps) != tt.expectedSteps {
				t.Errorf("Expected %d steps, got %d: %v", tt.expectedSteps, len(steps), steps)
			}
			
			hasAPIKey := false
			for _, step := range steps {
				if step == "API Key" {
					hasAPIKey = true
					break
				}
			}
			
			if hasAPIKey != tt.hasAPIKeyStep {
				t.Errorf("Expected hasAPIKeyStep=%v, got %v", tt.hasAPIKeyStep, hasAPIKey)
			}
		})
	}
}

func TestGetStepValue(t *testing.T) {
	provider := GetProviderByID("openai")
	if provider == nil {
		t.Fatal("OpenAI provider not found")
	}
	
	m := model{
		selectedProvider: provider,
		config: &ProviderConfig{
			Provider: provider,
			APIKey:   "sk-proj-1234567890abcdef",
			ModelID:  "gpt-4",
			BaseURL:  "https://api.openai.com/v1",
		},
	}
	
	tests := []struct {
		step     int
		expected string
	}{
		{0, "sk-p...cdef"}, // API key masked
		{1, "gpt-4"},       // Model ID
		{2, "https://api.openai.com/v1"}, // Base URL
	}
	
	for _, tt := range tests {
		t.Run(m.getConfigSteps()[tt.step], func(t *testing.T) {
			value := m.getStepValue(tt.step)
			if value != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, value)
			}
		})
	}
}

func TestGetStepValueShortAPIKey(t *testing.T) {
	provider := GetProviderByID("openai")
	if provider == nil {
		t.Fatal("OpenAI provider not found")
	}
	
	m := model{
		selectedProvider: provider,
		config: &ProviderConfig{
			Provider: provider,
			APIKey:   "short",
		},
	}
	
	value := m.getStepValue(0) // API Key step
	if value != "***" {
		t.Errorf("Expected '***' for short API key, got %s", value)
	}
}

func TestGetStepDefault(t *testing.T) {
	provider := GetProviderByID("openai")
	if provider == nil {
		t.Fatal("OpenAI provider not found")
	}
	
	m := model{
		selectedProvider: provider,
		config: &ProviderConfig{
			Provider: provider,
		},
	}
	
	steps := m.getConfigSteps()
	defaults := []string{"(required)", "gpt-4", "https://api.openai.com/v1"}
	
	for i, expected := range defaults {
		if i >= len(steps) {
			break
		}
		value := m.getStepDefault(i)
		if value != expected {
			t.Errorf("Step %d (%s): expected default %s, got %s", i, steps[i], expected, value)
		}
	}
}

func TestGenerateBishrcSection(t *testing.T) {
	provider := GetProviderByID("openai")
	if provider == nil {
		t.Fatal("OpenAI provider not found")
	}
	
	tests := []struct {
		modelType      string
		expectedPrefix string
	}{
		{"fast", "BISH_FAST_MODEL_"},
		{"slow", "BISH_SLOW_MODEL_"},
	}
	
	for _, tt := range tests {
		t.Run(tt.modelType, func(t *testing.T) {
			m := model{
				selectedProvider: provider,
				modelType:        tt.modelType,
				config: &ProviderConfig{
					Provider: provider,
					APIKey:   "test-key",
					BaseURL:  "https://api.openai.com/v1",
					ModelID:  "gpt-4",
				},
			}
			
			section := m.generateBishrcSection()
			
			// Check for required variables
			requiredVars := []string{
				tt.expectedPrefix + "API_KEY=test-key",
				tt.expectedPrefix + "BASE_URL=https://api.openai.com/v1",
				tt.expectedPrefix + "ID=gpt-4",
				tt.expectedPrefix + "TEMPERATURE=0.1",
				tt.expectedPrefix + "PARALLEL_TOOL_CALLS=true",
				tt.expectedPrefix + "HEADERS='{}'",
			}
			
			for _, required := range requiredVars {
				if !strings.Contains(section, required) {
					t.Errorf("Expected section to contain %s, got:\n%s", required, section)
				}
			}
			
			// Check for comment
			expectedComment := "# OpenAI Configuration (Generated by wizard)"
			if !strings.Contains(section, expectedComment) {
				t.Errorf("Expected section to contain comment, got:\n%s", section)
			}
		})
	}
}

func TestReplaceBishrcSection(t *testing.T) {
	provider := GetProviderByID("ollama")
	if provider == nil {
		t.Fatal("Ollama provider not found")
	}
	
	m := model{
		selectedProvider: provider,
		modelType:        "fast",
		config: &ProviderConfig{
			Provider: provider,
			APIKey:   "ollama",
			BaseURL:  "http://localhost:11434/v1",
			ModelID:  "qwen2.5",
		},
	}
	
	existingConfig := `# Bishop Configuration
BISH_PROMPT="bish> "

# Old fast model config
BISH_FAST_MODEL_API_KEY=old-key
BISH_FAST_MODEL_BASE_URL=http://old-url
BISH_FAST_MODEL_ID=old-model

# Slow model config
BISH_SLOW_MODEL_API_KEY=slow-key
BISH_SLOW_MODEL_BASE_URL=http://slow-url
`
	
	newSection := m.generateBishrcSection()
	result := m.replaceBishrcSection(existingConfig, newSection)
	
	// Old fast model config should be replaced
	if strings.Contains(result, "BISH_FAST_MODEL_API_KEY=old-key") {
		t.Error("Old fast model config should be replaced")
	}
	
	// New config should be present
	if !strings.Contains(result, "BISH_FAST_MODEL_API_KEY=ollama") {
		t.Error("New fast model config should be present")
	}
	
	// Slow model config should remain unchanged
	if !strings.Contains(result, "BISH_SLOW_MODEL_API_KEY=slow-key") {
		t.Error("Slow model config should remain unchanged")
	}
	
	// Other config should remain
	if !strings.Contains(result, `BISH_PROMPT="bish> "`) {
		t.Error("Other config should remain unchanged")
	}
}

func TestReplaceBishrcSectionNoExisting(t *testing.T) {
	provider := GetProviderByID("openai")
	if provider == nil {
		t.Fatal("OpenAI provider not found")
	}
	
	m := model{
		selectedProvider: provider,
		modelType:        "slow",
		config: &ProviderConfig{
			Provider: provider,
			APIKey:   "new-key",
			BaseURL:  "https://api.openai.com/v1",
			ModelID:  "gpt-4",
		},
	}
	
	existingConfig := `# Bishop Configuration
BISH_PROMPT="bish> "
`
	
	newSection := m.generateBishrcSection()
	result := m.replaceBishrcSection(existingConfig, newSection)
	
	// New section should be appended
	if !strings.Contains(result, "BISH_SLOW_MODEL_API_KEY=new-key") {
		t.Error("New slow model config should be appended")
	}
	
	// Original config should remain
	if !strings.Contains(result, `BISH_PROMPT="bish> "`) {
		t.Error("Original config should remain")
	}
}

func TestSaveConfigCreatesFile(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	
	// Override HOME for test
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	provider := GetProviderByID("ollama")
	if provider == nil {
		t.Fatal("Ollama provider not found")
	}
	
	m := model{
		selectedProvider: provider,
		modelType:        "fast",
		config: &ProviderConfig{
			Provider: provider,
			APIKey:   "ollama",
			BaseURL:  "http://localhost:11434/v1",
			ModelID:  "qwen2.5",
		},
	}
	
	err := m.saveConfig()
	if err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}
	
	bishrcPath := filepath.Join(tmpDir, ".bishrc")
	if _, err := os.Stat(bishrcPath); os.IsNotExist(err) {
		t.Error(".bishrc file was not created")
	}
	
	// Read and verify content
	content, err := os.ReadFile(bishrcPath)
	if err != nil {
		t.Fatalf("Failed to read .bishrc: %v", err)
	}
	
	contentStr := string(content)
	if !strings.Contains(contentStr, "BISH_FAST_MODEL_API_KEY=ollama") {
		t.Error("Config content not found in .bishrc")
	}
}

func TestSaveConfigUpdatesExisting(t *testing.T) {
	// Create temp directory for test
	tmpDir := t.TempDir()
	originalHome := os.Getenv("HOME")
	
	// Override HOME for test
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)
	
	// Create existing .bishrc
	bishrcPath := filepath.Join(tmpDir, ".bishrc")
	existingContent := `# Existing config
BISH_PROMPT="old> "
BISH_FAST_MODEL_API_KEY=old-key
`
	err := os.WriteFile(bishrcPath, []byte(existingContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create existing .bishrc: %v", err)
	}
	
	provider := GetProviderByID("openai")
	if provider == nil {
		t.Fatal("OpenAI provider not found")
	}
	
	m := model{
		selectedProvider: provider,
		modelType:        "fast",
		config: &ProviderConfig{
			Provider: provider,
			APIKey:   "new-key",
			BaseURL:  "https://api.openai.com/v1",
			ModelID:  "gpt-4",
		},
	}
	
	err = m.saveConfig()
	if err != nil {
		t.Fatalf("saveConfig failed: %v", err)
	}
	
	// Read and verify content
	content, err := os.ReadFile(bishrcPath)
	if err != nil {
		t.Fatalf("Failed to read .bishrc: %v", err)
	}
	
	contentStr := string(content)
	
	// New config should be present
	if !strings.Contains(contentStr, "BISH_FAST_MODEL_API_KEY=new-key") {
		t.Error("New config not found in updated .bishrc")
	}
	
	// Old key should be replaced
	if strings.Contains(contentStr, "BISH_FAST_MODEL_API_KEY=old-key") {
		t.Error("Old config should be replaced")
	}
	
	// Other config should remain
	if !strings.Contains(contentStr, `BISH_PROMPT="old> "`) {
		t.Error("Existing config should be preserved")
	}
}

func TestGetDefaultBishrc(t *testing.T) {
	content := getDefaultBishrc()
	
	// Check for essential sections
	essentials := []string{
		"BISH_PROMPT=",
		"BISH_LOG_LEVEL=",
		"BISH_AUTOCD=",
		"# Fast model configuration",
		"# Slow model configuration",
		"BISH_CONTEXT_TYPES_FOR_AGENT=",
	}
	
	for _, essential := range essentials {
		if !strings.Contains(content, essential) {
			t.Errorf("Default .bishrc missing: %s", essential)
		}
	}
}

func TestModelUpdateWelcome(t *testing.T) {
	m := model{
		state:     stateWelcome,
		modelType: "fast",
	}
	
	// Test Enter key
	newModel, _ := m.updateWelcome(tea.KeyMsg{Type: tea.KeyEnter})
	m2, ok := newModel.(model)
	if !ok {
		t.Fatal("Expected model type")
	}
	if m2.state != stateSelectProvider {
		t.Errorf("Expected state to transition to stateSelectProvider, got %v", m2.state)
	}
	
	// Test quit
	m = model{state: stateWelcome, modelType: "fast"}
	newModel, cmd := m.updateWelcome(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	m2, ok = newModel.(model)
	if !ok {
		t.Fatal("Expected model type")
	}
	if !m2.quit {
		t.Error("Expected quit to be true")
	}
	if cmd == nil {
		t.Error("Expected tea.Quit command")
	}
}

func TestValidationResultMsg(t *testing.T) {
	msg := validationResultMsg{
		success: true,
		message: "Test validation success",
	}
	
	if !msg.success {
		t.Error("Expected success to be true")
	}
	if msg.message != "Test validation success" {
		t.Errorf("Expected message 'Test validation success', got %s", msg.message)
	}
}

func TestModelViewStates(t *testing.T) {
	m := model{
		state:     stateWelcome,
		modelType: "fast",
		width:     80,
		height:    24,
	}
	
	// Test each view renders without panic
	views := []wizardState{
		stateWelcome,
		stateFinished,
	}
	
	for _, state := range views {
		m.state = state
		view := m.View()
		if view == "" {
			t.Errorf("View for state %v returned empty string", state)
		}
	}
}
