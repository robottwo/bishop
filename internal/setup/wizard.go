package setup

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	openai "github.com/sashabaranov/go-openai"
)

// Provider represents an LLM provider option
type Provider struct {
	ID          string
	Name        string
	Description string
	NeedsAPIKey bool
	DefaultURL  string
	HelpText    string
}

var providers = []Provider{
	{
		ID:          "ollama",
		Name:        "Ollama (Local)",
		Description: "Run LLMs locally on your machine - free and private",
		NeedsAPIKey: false,
		DefaultURL:  "http://localhost:11434/v1/",
		HelpText:    "Make sure Ollama is running: ollama serve",
	},
	{
		ID:          "openai",
		Name:        "OpenAI",
		Description: "Use OpenAI's GPT models (requires API key)",
		NeedsAPIKey: true,
		DefaultURL:  "https://api.openai.com/v1",
		HelpText:    "Get your API key at: https://platform.openai.com/api-keys",
	},
	{
		ID:          "openrouter",
		Name:        "OpenRouter",
		Description: "Access 100+ models through one API (requires API key)",
		NeedsAPIKey: true,
		DefaultURL:  "https://openrouter.ai/api/v1",
		HelpText:    "Get your API key at: https://openrouter.ai/keys",
	},
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("170")).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("170")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			MarginTop(1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("62")).
			Padding(1, 2)
)

// WizardState represents the current step in the wizard
type WizardState int

const (
	stateWelcome WizardState = iota
	stateSelectProvider
	stateEnterAPIKey
	stateEnterBaseURL
	stateTestConnection
	stateSelectModel
	stateComplete
)

// WizardResult contains the configuration from the wizard
type WizardResult struct {
	Provider string
	APIKey   string
	BaseURL  string
	ModelID  string
	Skipped  bool
}

// Model represents the Bubbletea model for the wizard
type Model struct {
	state          WizardState
	selectedIdx    int
	providers      []Provider
	selectedAPIKey string
	customBaseURL  string
	textInput      textinput.Model
	spinner        spinner.Model
	testing        bool
	testResult     *testResult
	models         []string
	modelIdx       int
	width          int
	height         int
	quitting       bool
	skipped        bool
	err            error
}

type testResult struct {
	success bool
	message string
	models  []string
}

// Messages
type testConnectionMsg struct {
	result *testResult
}

func NewWizard() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter your API key..."
	ti.Focus()
	ti.CharLimit = 256
	ti.Width = 50
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '*'

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("170"))

	return Model{
		state:     stateWelcome,
		providers: providers,
		textInput: ti,
		spinner:   s,
	}
}

func (m Model) Init() tea.Cmd {
	return textinput.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.state == stateWelcome {
				m.skipped = true
				m.quitting = true
				return m, tea.Quit
			}
			if !m.testing {
				m.quitting = true
				return m, tea.Quit
			}

		case "esc":
			if m.testing {
				return m, nil
			}
			// Go back to previous state
			switch m.state {
			case stateEnterAPIKey:
				m.state = stateSelectProvider
			case stateEnterBaseURL:
				if m.providers[m.selectedIdx].NeedsAPIKey {
					m.state = stateEnterAPIKey
				} else {
					m.state = stateSelectProvider
				}
			case stateTestConnection:
				m.state = stateEnterBaseURL
				m.testResult = nil
			case stateSelectModel:
				m.state = stateTestConnection
			}
			return m, nil

		case "enter":
			return m.handleEnter()

		case "up", "k":
			if !m.testing {
				switch m.state {
				case stateSelectProvider:
					if m.selectedIdx > 0 {
						m.selectedIdx--
					}
				case stateSelectModel:
					if m.modelIdx > 0 {
						m.modelIdx--
					}
				}
			}

		case "down", "j":
			if !m.testing {
				switch m.state {
				case stateSelectProvider:
					if m.selectedIdx < len(m.providers)-1 {
						m.selectedIdx++
					}
				case stateSelectModel:
					if m.modelIdx < len(m.models)-1 {
						m.modelIdx++
					}
				}
			}

		case "s":
			// Skip wizard from welcome screen
			if m.state == stateWelcome {
				m.skipped = true
				m.quitting = true
				return m, tea.Quit
			}
		}

	case testConnectionMsg:
		m.testing = false
		m.testResult = msg.result
		if msg.result.success && len(msg.result.models) > 0 {
			m.models = msg.result.models
			m.state = stateSelectModel
		}
		return m, nil

	case spinner.TickMsg:
		if m.testing {
			m.spinner, cmd = m.spinner.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	// Update text input for API key and base URL states
	if m.state == stateEnterAPIKey || m.state == stateEnterBaseURL {
		m.textInput, cmd = m.textInput.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.state {
	case stateWelcome:
		m.state = stateSelectProvider
		return m, nil

	case stateSelectProvider:
		provider := m.providers[m.selectedIdx]
		if provider.NeedsAPIKey {
			m.textInput.Placeholder = "Enter your API key..."
			m.textInput.EchoMode = textinput.EchoPassword
			m.textInput.EchoCharacter = '*'
			m.textInput.SetValue("")
			m.state = stateEnterAPIKey
		} else {
			m.textInput.Placeholder = fmt.Sprintf("Base URL (default: %s)", provider.DefaultURL)
			m.textInput.EchoMode = textinput.EchoNormal
			m.textInput.SetValue("")
			m.state = stateEnterBaseURL
		}
		return m, nil

	case stateEnterAPIKey:
		m.selectedAPIKey = m.textInput.Value()
		if m.selectedAPIKey == "" {
			m.err = fmt.Errorf("API key is required")
			return m, nil
		}
		m.err = nil
		provider := m.providers[m.selectedIdx]
		m.textInput.Placeholder = fmt.Sprintf("Base URL (default: %s)", provider.DefaultURL)
		m.textInput.EchoMode = textinput.EchoNormal
		m.textInput.SetValue("")
		m.state = stateEnterBaseURL
		return m, nil

	case stateEnterBaseURL:
		m.customBaseURL = m.textInput.Value()
		m.state = stateTestConnection
		m.testing = true
		m.testResult = nil
		return m, tea.Batch(m.spinner.Tick, m.testConnection())

	case stateTestConnection:
		if m.testResult != nil && m.testResult.success {
			if len(m.models) > 0 {
				m.state = stateSelectModel
			} else {
				m.state = stateComplete
			}
		} else {
			// Retry connection
			m.testing = true
			m.testResult = nil
			return m, tea.Batch(m.spinner.Tick, m.testConnection())
		}
		return m, nil

	case stateSelectModel:
		m.state = stateComplete
		return m, nil

	case stateComplete:
		m.quitting = true
		return m, tea.Quit
	}

	return m, nil
}

func (m Model) testConnection() tea.Cmd {
	return func() tea.Msg {
		provider := m.providers[m.selectedIdx]
		baseURL := m.customBaseURL
		if baseURL == "" {
			baseURL = provider.DefaultURL
		}

		apiKey := m.selectedAPIKey
		if apiKey == "" && !provider.NeedsAPIKey {
			apiKey = "ollama"
		}

		// Create client config
		config := openai.DefaultConfig(apiKey)
		config.BaseURL = baseURL

		// Add OpenRouter headers if needed
		if provider.ID == "openrouter" {
			config.HTTPClient = newOpenRouterClient()
		}

		client := openai.NewClientWithConfig(config)

		// Test connection by listing models
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		resp, err := client.ListModels(ctx)
		if err != nil {
			return testConnectionMsg{
				result: &testResult{
					success: false,
					message: fmt.Sprintf("Connection failed: %v", err),
				},
			}
		}

		// Extract model IDs
		var models []string
		for _, model := range resp.Models {
			models = append(models, model.ID)
		}

		// For Ollama, suggest common models if list is empty or too long
		if provider.ID == "ollama" && len(models) == 0 {
			models = []string{"qwen2.5", "qwen2.5:32b", "llama3.2", "mistral", "codellama"}
		}

		// Limit the number of models shown
		if len(models) > 20 {
			models = models[:20]
		}

		return testConnectionMsg{
			result: &testResult{
				success: true,
				message: fmt.Sprintf("Connected! Found %d models", len(resp.Models)),
				models:  models,
			},
		}
	}
}

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	var content strings.Builder

	switch m.state {
	case stateWelcome:
		content.WriteString(m.viewWelcome())
	case stateSelectProvider:
		content.WriteString(m.viewSelectProvider())
	case stateEnterAPIKey:
		content.WriteString(m.viewEnterAPIKey())
	case stateEnterBaseURL:
		content.WriteString(m.viewEnterBaseURL())
	case stateTestConnection:
		content.WriteString(m.viewTestConnection())
	case stateSelectModel:
		content.WriteString(m.viewSelectModel())
	case stateComplete:
		content.WriteString(m.viewComplete())
	}

	// Center the content
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content.String())
}

func (m Model) viewWelcome() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Welcome to Bishop!"))
	s.WriteString("\n\n")
	s.WriteString(subtitleStyle.Render("The AI-Powered Generative Shell"))
	s.WriteString("\n\n")
	s.WriteString(normalStyle.Render("Bishop integrates LLM capabilities directly into your shell,"))
	s.WriteString("\n")
	s.WriteString(normalStyle.Render("providing AI-powered command suggestions, explanations, and more."))
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("This wizard will help you configure your LLM provider."))
	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("Press Enter to continue, or 's' to skip"))

	return boxStyle.Render(s.String())
}

func (m Model) viewSelectProvider() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Select an LLM Provider"))
	s.WriteString("\n\n")

	for i, provider := range m.providers {
		cursor := "  "
		style := normalStyle
		if i == m.selectedIdx {
			cursor = "> "
			style = selectedStyle
		}

		s.WriteString(style.Render(cursor + provider.Name))
		s.WriteString("\n")
		s.WriteString(dimStyle.Render("    " + provider.Description))
		s.WriteString("\n")
		if i < len(m.providers)-1 {
			s.WriteString("\n")
		}
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Use arrow keys to navigate, Enter to select, Esc to quit"))

	return boxStyle.Render(s.String())
}

func (m Model) viewEnterAPIKey() string {
	var s strings.Builder

	provider := m.providers[m.selectedIdx]

	s.WriteString(titleStyle.Render("Enter API Key"))
	s.WriteString("\n\n")
	s.WriteString(normalStyle.Render(fmt.Sprintf("Configure %s", provider.Name)))
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render(provider.HelpText))
	s.WriteString("\n\n")
	s.WriteString(m.textInput.View())

	if m.err != nil {
		s.WriteString("\n\n")
		s.WriteString(errorStyle.Render(m.err.Error()))
	}

	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("Enter to continue, Esc to go back"))

	return boxStyle.Render(s.String())
}

func (m Model) viewEnterBaseURL() string {
	var s strings.Builder

	provider := m.providers[m.selectedIdx]

	s.WriteString(titleStyle.Render("Configure Base URL (Optional)"))
	s.WriteString("\n\n")
	s.WriteString(normalStyle.Render("Leave empty to use the default URL:"))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render(provider.DefaultURL))
	s.WriteString("\n\n")
	s.WriteString(m.textInput.View())
	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("Enter to continue, Esc to go back"))

	return boxStyle.Render(s.String())
}

func (m Model) viewTestConnection() string {
	var s strings.Builder

	provider := m.providers[m.selectedIdx]
	baseURL := m.customBaseURL
	if baseURL == "" {
		baseURL = provider.DefaultURL
	}

	s.WriteString(titleStyle.Render("Testing Connection"))
	s.WriteString("\n\n")
	s.WriteString(normalStyle.Render(fmt.Sprintf("Provider: %s", provider.Name)))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render(fmt.Sprintf("URL: %s", baseURL)))
	s.WriteString("\n\n")

	if m.testing {
		s.WriteString(m.spinner.View())
		s.WriteString(" Connecting...")
	} else if m.testResult != nil {
		if m.testResult.success {
			s.WriteString(successStyle.Render("+ " + m.testResult.message))
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("Press Enter to continue"))
		} else {
			s.WriteString(errorStyle.Render("x " + m.testResult.message))
			s.WriteString("\n\n")
			s.WriteString(helpStyle.Render("Press Enter to retry, Esc to go back"))
		}
	}

	return boxStyle.Render(s.String())
}

func (m Model) viewSelectModel() string {
	var s strings.Builder

	s.WriteString(titleStyle.Render("Select a Default Model"))
	s.WriteString("\n\n")
	s.WriteString(dimStyle.Render("This model will be used for both fast and slow operations."))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("You can configure separate models later with #!config"))
	s.WriteString("\n\n")

	// Show up to 10 models at a time with scrolling
	startIdx := 0
	if m.modelIdx > 7 {
		startIdx = m.modelIdx - 7
	}
	endIdx := startIdx + 10
	if endIdx > len(m.models) {
		endIdx = len(m.models)
	}

	for i := startIdx; i < endIdx; i++ {
		cursor := "  "
		style := normalStyle
		if i == m.modelIdx {
			cursor = "> "
			style = selectedStyle
		}
		s.WriteString(style.Render(cursor + m.models[i]))
		s.WriteString("\n")
	}

	if len(m.models) > 10 {
		s.WriteString(dimStyle.Render(fmt.Sprintf("\n... and %d more models", len(m.models)-10)))
	}

	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Use arrow keys to navigate, Enter to select"))

	return boxStyle.Render(s.String())
}

func (m Model) viewComplete() string {
	var s strings.Builder

	provider := m.providers[m.selectedIdx]
	modelID := ""
	if len(m.models) > 0 && m.modelIdx < len(m.models) {
		modelID = m.models[m.modelIdx]
	}

	s.WriteString(successStyle.Render("Setup Complete!"))
	s.WriteString("\n\n")
	s.WriteString(normalStyle.Render("Configuration Summary:"))
	s.WriteString("\n\n")
	s.WriteString(fmt.Sprintf("  Provider: %s\n", provider.Name))
	if modelID != "" {
		s.WriteString(fmt.Sprintf("  Model:    %s\n", modelID))
	}
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("Your settings have been saved to ~/.bishenv"))
	s.WriteString("\n")
	s.WriteString(dimStyle.Render("Use #!config to change settings anytime."))
	s.WriteString("\n\n")
	s.WriteString(helpStyle.Render("Press Enter to start using Bishop!"))

	return boxStyle.Render(s.String())
}

// GetResult returns the wizard result after completion
func (m Model) GetResult() WizardResult {
	if m.skipped {
		return WizardResult{Skipped: true}
	}

	provider := m.providers[m.selectedIdx]
	baseURL := m.customBaseURL
	if baseURL == "" {
		baseURL = provider.DefaultURL
	}

	modelID := ""
	if len(m.models) > 0 && m.modelIdx < len(m.models) {
		modelID = m.models[m.modelIdx]
	}

	return WizardResult{
		Provider: provider.ID,
		APIKey:   m.selectedAPIKey,
		BaseURL:  baseURL,
		ModelID:  modelID,
	}
}

// IsFirstRun checks if this is the first time the shell is being run
// by checking if the configuration files exist and have LLM settings
func IsFirstRun() bool {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	// Check if .bishrc exists
	bishrcPath := filepath.Join(homeDir, ".bishrc")
	if _, err := os.Stat(bishrcPath); os.IsNotExist(err) {
		return true
	}

	// Check if .bishenv exists with LLM settings
	bishenvPath := filepath.Join(homeDir, ".bishenv")
	if _, err := os.Stat(bishenvPath); os.IsNotExist(err) {
		// Check if .bishrc has LLM settings
		content, err := os.ReadFile(bishrcPath)
		if err != nil {
			return true
		}
		// If no provider is configured, consider it first run
		if !strings.Contains(string(content), "BISH_SLOW_MODEL_PROVIDER") &&
			!strings.Contains(string(content), "BISH_FAST_MODEL_PROVIDER") {
			return true
		}
	}

	// Check if .bish_config_ui exists with LLM settings
	configUIPath := filepath.Join(homeDir, ".bish_config_ui")
	if _, err := os.Stat(configUIPath); err == nil {
		content, err := os.ReadFile(configUIPath)
		if err == nil && strings.Contains(string(content), "BISH_SLOW_MODEL_PROVIDER") {
			return false
		}
	}

	// Check .bishenv for provider settings
	if content, err := os.ReadFile(bishenvPath); err == nil {
		if strings.Contains(string(content), "BISH_SLOW_MODEL_PROVIDER") ||
			strings.Contains(string(content), "BISH_FAST_MODEL_PROVIDER") {
			return false
		}
	}

	// Check .bishrc for provider settings
	if content, err := os.ReadFile(bishrcPath); err == nil {
		if strings.Contains(string(content), "BISH_SLOW_MODEL_PROVIDER") ||
			strings.Contains(string(content), "BISH_FAST_MODEL_PROVIDER") {
			return false
		}
	}

	return true
}

// SaveConfiguration saves the wizard result to ~/.bishenv
func SaveConfiguration(result WizardResult) error {
	if result.Skipped {
		return nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	bishenvPath := filepath.Join(homeDir, ".bishenv")

	var content strings.Builder
	content.WriteString("# Bishop LLM Configuration\n")
	content.WriteString("# Generated by setup wizard\n\n")

	// Fast model settings
	content.WriteString(fmt.Sprintf("export BISH_FAST_MODEL_PROVIDER='%s'\n", result.Provider))
	if result.APIKey != "" {
		content.WriteString(fmt.Sprintf("export BISH_FAST_MODEL_API_KEY='%s'\n", escapeShellValue(result.APIKey)))
	}
	if result.BaseURL != "" {
		content.WriteString(fmt.Sprintf("export BISH_FAST_MODEL_BASE_URL='%s'\n", result.BaseURL))
	}
	if result.ModelID != "" {
		content.WriteString(fmt.Sprintf("export BISH_FAST_MODEL_ID='%s'\n", result.ModelID))
	}

	content.WriteString("\n")

	// Slow model settings (same as fast by default)
	content.WriteString(fmt.Sprintf("export BISH_SLOW_MODEL_PROVIDER='%s'\n", result.Provider))
	if result.APIKey != "" {
		content.WriteString(fmt.Sprintf("export BISH_SLOW_MODEL_API_KEY='%s'\n", escapeShellValue(result.APIKey)))
	}
	if result.BaseURL != "" {
		content.WriteString(fmt.Sprintf("export BISH_SLOW_MODEL_BASE_URL='%s'\n", result.BaseURL))
	}
	if result.ModelID != "" {
		content.WriteString(fmt.Sprintf("export BISH_SLOW_MODEL_ID='%s'\n", result.ModelID))
	}

	// Write to file with secure permissions
	if err := os.WriteFile(bishenvPath, []byte(content.String()), 0600); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// escapeShellValue escapes a value for safe use in shell config
func escapeShellValue(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}

// RunWizard runs the setup wizard and returns the result
func RunWizard() (WizardResult, error) {
	p := tea.NewProgram(NewWizard(), tea.WithAltScreen())
	m, err := p.Run()
	if err != nil {
		return WizardResult{}, err
	}

	model := m.(Model)
	return model.GetResult(), nil
}
