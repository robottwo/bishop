package wizard

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle        = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	subtitleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	selectedStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))
	helpStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	errorStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	successStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	validatingStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("226"))
)

type wizardState int

const (
	stateWelcome wizardState = iota
	stateSelectProvider
	stateConfigureProvider
	stateSummary
	stateFinished
)

type model struct {
	state            wizardState
	width            int
	height           int
	
	// Provider selection
	providerList     list.Model
	selectedProvider *Provider
	
	// Configuration
	textInput        textinput.Model
	configStep       int
	config           *ProviderConfig
	
	// Summary
	summaryList      list.Model
	
	// Status
	errorMsg         string
	validating       bool
	
	// Wizard mode (fast or slow model)
	modelType        string // "fast" or "slow"
	
	// Finished
	bishrcPath       string
	quit             bool
}

type providerItem Provider

func (p providerItem) Title() string       { return p.Name }
func (p providerItem) Description() string { return p.Desc }
func (p providerItem) FilterValue() string { return p.Name }

type summaryItem struct {
	key   string
	value string
}

func (s summaryItem) Title() string       { return fmt.Sprintf("%s: %s", s.key, s.value) }
func (s summaryItem) Description() string { return "" }
func (s summaryItem) FilterValue() string { return s.key }

// Start begins the wizard for configuring a model (fast or slow)
func Start(modelType string) error {
	// Create provider list
	items := make([]list.Item, len(AllProviders))
	for i, p := range AllProviders {
		items[i] = providerItem(p)
	}
	
	providerList := list.New(items, list.NewDefaultDelegate(), 0, 0)
	providerList.Title = "Select a Provider"
	providerList.SetShowHelp(false)
	
	// Create text input
	ti := textinput.New()
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50
	
	m := model{
		state:        stateWelcome,
		providerList: providerList,
		textInput:    ti,
		modelType:    modelType,
	}
	
	p := tea.NewProgram(m, tea.WithAltScreen())
	finalModel, err := p.Run()
	if err != nil {
		return err
	}
	
	// Check if user quit without finishing
	if m, ok := finalModel.(model); ok && m.quit && m.state != stateFinished {
		return fmt.Errorf("wizard cancelled")
	}
	
	return nil
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.state {
		case stateWelcome:
			return m.updateWelcome(msg)
		case stateSelectProvider:
			return m.updateProviderSelection(msg)
		case stateConfigureProvider:
			return m.updateConfiguration(msg)
		case stateSummary:
			return m.updateSummary(msg)
		case stateFinished:
			if msg.String() == "enter" || msg.String() == "q" {
				m.quit = true
				return m, tea.Quit
			}
		}
	
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.providerList.SetSize(msg.Width, msg.Height-10)
		if m.summaryList.Items() != nil {
			m.summaryList.SetSize(msg.Width, msg.Height-10)
		}
	
	case validationResultMsg:
		m.validating = false
		if msg.success {
			m.config.Validated = true
			m.config.ValidationMsg = msg.message
			m.state = stateSummary
			return m, m.buildSummary()
		} else {
			m.errorMsg = "Validation failed: " + msg.message
			// Stay in config state to allow retry
		}
	}
	
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Loading..."
	}
	
	switch m.state {
	case stateWelcome:
		return m.viewWelcome()
	case stateSelectProvider:
		return m.viewProviderSelection()
	case stateConfigureProvider:
		return m.viewConfiguration()
	case stateSummary:
		return m.viewSummary()
	case stateFinished:
		return m.viewFinished()
	}
	
	return "Unknown state"
}

func (m model) viewWelcome() string {
	var b strings.Builder
	
	modelName := "Fast Model"
	modelDesc := "used for auto-suggestions and command predictions"
	if m.modelType == "slow" {
		modelName = "Slow Model"
		modelDesc = "used for chat and agentic operations"
	}
	
	b.WriteString(titleStyle.Render("Welcome to Bishop Configuration Wizard"))
	b.WriteString("\n\n")
	b.WriteString(subtitleStyle.Render(fmt.Sprintf("Configure %s (%s)", modelName, modelDesc)))
	b.WriteString("\n\n")
	b.WriteString("This wizard will help you set up your LLM provider.\n")
	b.WriteString("\n")
	b.WriteString("Press Enter to continue, or q to quit.\n")
	
	return b.String()
}

func (m model) updateWelcome(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		m.state = stateSelectProvider
		return m, nil
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	}
	return m, nil
}

func (m model) viewProviderSelection() string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render("Select a Provider"))
	b.WriteString("\n\n")
	b.WriteString(m.providerList.View())
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("↑/↓: navigate • enter: select • q: quit"))
	
	return b.String()
}

func (m model) updateProviderSelection(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if i, ok := m.providerList.SelectedItem().(providerItem); ok {
			provider := Provider(i)
			m.selectedProvider = &provider
			m.config = &ProviderConfig{
				Provider: &provider,
				BaseURL:  provider.DefaultBaseURL,
				ModelID:  provider.DefaultModel,
			}
			if !provider.RequiresAPIKey {
				m.config.APIKey = provider.DefaultAPIKey
			}
			m.configStep = 0
			m.state = stateConfigureProvider
			m.textInput.SetValue("")
			return m, m.focusInput()
		}
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	}
	
	var cmd tea.Cmd
	m.providerList, cmd = m.providerList.Update(msg)
	return m, cmd
}

func (m model) viewConfiguration() string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render(fmt.Sprintf("Configure %s", m.selectedProvider.Name)))
	b.WriteString("\n\n")
	
	// Show configuration steps
	steps := m.getConfigSteps()
	for i, step := range steps {
		if i < m.configStep {
			// Already completed
			b.WriteString(successStyle.Render("✓ "))
			b.WriteString(fmt.Sprintf("%s: %s\n", step, m.getStepValue(i)))
		} else if i == m.configStep {
			// Current step
			b.WriteString(selectedStyle.Render("→ "))
			b.WriteString(fmt.Sprintf("%s\n", step))
			b.WriteString("  ")
			b.WriteString(m.textInput.View())
			b.WriteString("\n")
			
			// Show default value hint
			if defaultVal := m.getStepDefault(i); defaultVal != "" {
				b.WriteString(subtitleStyle.Render(fmt.Sprintf("  (default: %s)", defaultVal)))
				b.WriteString("\n")
			}
		} else {
			// Not yet reached
			b.WriteString("  ")
			b.WriteString(subtitleStyle.Render(step))
			b.WriteString("\n")
		}
	}
	
	b.WriteString("\n")
	if m.errorMsg != "" {
		b.WriteString(errorStyle.Render("Error: " + m.errorMsg))
		b.WriteString("\n\n")
	}
	
	if m.validating {
		b.WriteString(validatingStyle.Render("Validating..."))
		b.WriteString("\n\n")
	}
	
	b.WriteString(helpStyle.Render("enter: next • ctrl+c: cancel"))
	
	return b.String()
}

func (m model) getConfigSteps() []string {
	steps := []string{}
	if m.selectedProvider.RequiresAPIKey {
		steps = append(steps, "API Key")
	}
	steps = append(steps, "Model ID")
	steps = append(steps, "Base URL")
	return steps
}

func (m model) getStepValue(step int) string {
	steps := m.getConfigSteps()
	if step >= len(steps) {
		return ""
	}
	
	switch steps[step] {
	case "API Key":
		if m.config.APIKey == "" {
			return "(not set)"
		}
		// Mask API key
		if len(m.config.APIKey) > 8 {
			return m.config.APIKey[:4] + "..." + m.config.APIKey[len(m.config.APIKey)-4:]
		}
		return "***"
	case "Model ID":
		return m.config.ModelID
	case "Base URL":
		return m.config.BaseURL
	}
	return ""
}

func (m model) getStepDefault(step int) string {
	steps := m.getConfigSteps()
	if step >= len(steps) {
		return ""
	}
	
	switch steps[step] {
	case "API Key":
		return "(required)"
	case "Model ID":
		return m.selectedProvider.DefaultModel
	case "Base URL":
		return m.selectedProvider.DefaultBaseURL
	}
	return ""
}

func (m model) updateConfiguration(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		m.quit = true
		return m, tea.Quit
	
	case "enter":
		// Save current step
		value := strings.TrimSpace(m.textInput.Value())
		steps := m.getConfigSteps()
		
		switch steps[m.configStep] {
		case "API Key":
			if value == "" {
				m.errorMsg = "API key is required"
				return m, nil
			}
			m.config.APIKey = value
		case "Model ID":
			if value != "" {
				m.config.ModelID = value
			}
		case "Base URL":
			if value != "" {
				m.config.BaseURL = value
			}
		}
		
		m.errorMsg = ""
		m.configStep++
		
		// Check if we're done
		if m.configStep >= len(steps) {
			// Validate configuration
			m.validating = true
			return m, m.validateConfig()
		}
		
		// Move to next step
		m.textInput.SetValue("")
		m.textInput.Focus()
		return m, nil
	}
	
	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

type validationResultMsg struct {
	success bool
	message string
}

func (m model) validateConfig() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		
		success, message := ValidateProvider(ctx, m.config)
		return validationResultMsg{
			success: success,
			message: message,
		}
	}
}

func (m model) buildSummary() tea.Cmd {
	items := []list.Item{
		summaryItem{key: "Provider", value: m.selectedProvider.Name},
		summaryItem{key: "Model", value: m.config.ModelID},
		summaryItem{key: "Base URL", value: m.config.BaseURL},
	}
	
	if m.selectedProvider.RequiresAPIKey {
		apiKeyMasked := "(hidden)"
		if len(m.config.APIKey) > 8 {
			apiKeyMasked = m.config.APIKey[:4] + "..." + m.config.APIKey[len(m.config.APIKey)-4:]
		}
		items = append(items, summaryItem{key: "API Key", value: apiKeyMasked})
	}
	
	items = append(items, summaryItem{key: "Validation", value: m.config.ValidationMsg})
	
	m.summaryList = list.New(items, list.NewDefaultDelegate(), m.width, m.height-10)
	m.summaryList.Title = "Configuration Summary"
	m.summaryList.SetShowHelp(false)
	
	return nil
}

func (m model) viewSummary() string {
	var b strings.Builder
	
	b.WriteString(titleStyle.Render("Configuration Summary"))
	b.WriteString("\n\n")
	
	for _, item := range m.summaryList.Items() {
		if s, ok := item.(summaryItem); ok {
			b.WriteString(fmt.Sprintf("%s: %s\n", s.key, s.value))
		}
	}
	
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter: save and continue • q: quit without saving"))
	
	return b.String()
}

func (m model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		// Save configuration to .bishrc
		if err := m.saveConfig(); err != nil {
			m.errorMsg = err.Error()
			return m, nil
		}
		m.state = stateFinished
		return m, nil
	
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	}
	
	return m, nil
}

func (m model) saveConfig() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot determine home directory: %w", err)
	}
	
	bishrcPath := filepath.Join(homeDir, ".bishrc")
	m.bishrcPath = bishrcPath
	
	// Check if file exists
	var existing []byte
	if _, err := os.Stat(bishrcPath); err == nil {
		existing, err = os.ReadFile(bishrcPath)
		if err != nil {
			return fmt.Errorf("cannot read existing .bishrc: %w", err)
		}
	}
	
	// Generate new configuration
	newConfig := m.generateBishrcSection()
	
	// If file exists, append; otherwise create new
	var content string
	if len(existing) > 0 {
		content = string(existing) + "\n\n" + newConfig
	} else {
		// Use default .bishrc as template
		content = getDefaultBishrc()
		// Replace the relevant model section
		content = m.replaceBishrcSection(content, newConfig)
	}
	
	// Write file
	if err := os.WriteFile(bishrcPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("cannot write .bishrc: %w", err)
	}
	
	return nil
}

func (m model) generateBishrcSection() string {
	var b strings.Builder
	
	prefix := "BISH_FAST_MODEL_"
	if m.modelType == "slow" {
		prefix = "BISH_SLOW_MODEL_"
	}
	
	b.WriteString(fmt.Sprintf("# %s Configuration (Generated by wizard)\n", m.selectedProvider.Name))
	b.WriteString(fmt.Sprintf("%sAPI_KEY=%s\n", prefix, m.config.APIKey))
	b.WriteString(fmt.Sprintf("%sBASE_URL=%s\n", prefix, m.config.BaseURL))
	b.WriteString(fmt.Sprintf("%sID=%s\n", prefix, m.config.ModelID))
	b.WriteString(fmt.Sprintf("%sTEMPERATURE=0.1\n", prefix))
	b.WriteString(fmt.Sprintf("%sPARALLEL_TOOL_CALLS=true\n", prefix))
	b.WriteString(fmt.Sprintf("%sHEADERS='{}'\n", prefix))
	
	return b.String()
}

func (m model) replaceBishrcSection(content, newConfig string) string {
	// Simple replace for now - just replace the model section
	prefix := "BISH_FAST_MODEL_"
	if m.modelType == "slow" {
		prefix = "BISH_SLOW_MODEL_"
	}
	
	lines := strings.Split(content, "\n")
	var result []string
	inSection := false
	replaced := false
	
	for _, line := range lines {
		if strings.HasPrefix(line, prefix) {
			if !inSection {
				inSection = true
				if !replaced {
					result = append(result, newConfig)
					replaced = true
				}
			}
			// Skip old config lines
			continue
		} else {
			if inSection {
				inSection = false
			}
			result = append(result, line)
		}
	}
	
	if !replaced {
		// Didn't find section to replace, append
		result = append(result, "", newConfig)
	}
	
	return strings.Join(result, "\n")
}

func (m model) viewFinished() string {
	var b strings.Builder
	
	b.WriteString(successStyle.Render("✓ Configuration Saved"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("Your configuration has been saved to: %s\n", m.bishrcPath))
	b.WriteString("\n")
	b.WriteString("You can now start using Bishop with your configured provider.\n")
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press Enter or q to exit"))
	
	return b.String()
}

func (m model) focusInput() tea.Cmd {
	return textinput.Blink
}

// getDefaultBishrc returns the default .bishrc template
// This should be the same as the embedded default in cmd/bish/main.go
func getDefaultBishrc() string {
	// For now, return a minimal template
	// In a real implementation, this should be embedded or loaded from the same source
	return `# Bishop Configuration

# BISH_UPDATE_PROMPT gets called each time before bishop renders the prompt
function BISH_UPDATE_PROMPT() {
  # BISH_PROMPT="bish> "
}

BISH_PROMPT="bish> "
BISH_LOG_LEVEL="info"
BISH_CLEAN_LOG_FILE=0
BISH_AUTOCD=1
BISH_AUTOCD_VERBOSE=1
BISH_ASSISTANT_HEIGHT=3

# Fast model configuration will be added here by wizard

# Slow model configuration will be added here by wizard

# RAG Configuration
BISH_CONTEXT_TYPES_FOR_AGENT=system_info,working_directory,git_status,history_verbose
BISH_CONTEXT_TYPES_FOR_PREDICTION_WITH_PREFIX=system_info,working_directory,git_status,history_concise
BISH_CONTEXT_TYPES_FOR_PREDICTION_WITHOUT_PREFIX=system_info,working_directory,git_status,history_verbose
BISH_CONTEXT_TYPES_FOR_EXPLANATION=system_info,working_directory
BISH_CONTEXT_NUM_HISTORY_CONCISE=30
BISH_CONTEXT_NUM_HISTORY_VERBOSE=30

# Agent Configuration
BISH_DEFAULT_TO_YES=0
BISH_AGENT_CONTEXT_WINDOW_TOKENS=32768
BISH_IDLE_SUMMARY_TIMEOUT_SECONDS=60
`
}
