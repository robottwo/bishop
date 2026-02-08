package wizard

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sashabaranov/go-openai"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

var (
	titleStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("62")).Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	successStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	boxStyle      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("62"))
	stepIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type providerItem struct {
	title       string
	description string
	provider    string
}

func (p providerItem) Title() string       { return p.title }
func (p providerItem) Description() string { return p.description }
func (p providerItem) FilterValue() string { return p.title }

type modelItem struct {
	title       string
	description string
	modelID     string
}

func (m modelItem) Title() string       { return m.title }
func (m modelItem) Description() string { return m.description }
func (m modelItem) FilterValue() string { return m.title }

type testCompleteMsg struct {
	success bool
	err     error
}

type tickMsg struct{}

func (m wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.providerList.SetWidth(msg.Width - 8)
		m.providerList.SetHeight(msg.Height - 8)
		m.modelList.SetWidth(msg.Width - 8)
		m.modelList.SetHeight(msg.Height - 8)
		m.helpViewport.Width = msg.Width - 8
		m.helpViewport.Height = msg.Height - 8

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			m.quitting = true
			return m, tea.Quit
		}

		if m.testingInProgress {
			return m, nil
		}

		switch m.step {
		case stepWelcome:
			if msg.Type == tea.KeyEnter || msg.String() == " " {
				m.step = stepFastProvider
				m.initProviderList()
			}
		case stepFastProvider, stepSlowProvider:
			switch msg.Type {
			case tea.KeyEnter:
				if item, ok := m.providerList.SelectedItem().(providerItem); ok {
					if m.step == stepFastProvider {
						m.config.fastModel.provider = item.provider
						m.config.fastModel.baseURL = getDefaultBaseURL(item.provider)
					} else {
						m.config.slowModel.provider = item.provider
						m.config.slowModel.baseURL = getDefaultBaseURL(item.provider)
					}

					if item.provider == "ollama" {
						m.step = m.step + 2
						m.initModelList(item.provider, "")
					} else {
						cachedKey, hasKey := m.config.apiKeyCache[item.provider]
						if hasKey {
							if m.step == stepFastProvider {
								m.config.fastModel.apiKey = cachedKey
								m.config.fastModel.validated = true
							} else {
								m.config.slowModel.apiKey = cachedKey
								m.config.slowModel.validated = true
							}
							m.step = m.step + 2
							m.initModelList(item.provider, cachedKey)
						} else {
							m.step = m.step + 1
							m.textInput.Reset()
							m.textInput.Placeholder = fmt.Sprintf("Enter %s API key", item.provider)
							m.textInput.Focus()
						}
					}
				}
			case tea.KeyEsc:
				if m.step == stepSlowProvider {
					m.step = stepFastTest
				}
			}
			m.providerList, cmd = m.providerList.Update(msg)
			return m, cmd

		case stepFastAPIKey, stepSlowAPIKey:
			// Clear error message when user starts typing
			m.errorMsg = ""
			switch msg.Type {
			case tea.KeyEnter:
				apiKey := m.textInput.Value()
				if err := validateAPIKeyFormat(apiKey, m.getCurrentProvider()); err != nil {
					m.errorMsg = fmt.Sprintf("Invalid API key: %v", err)
					return m, nil
				}

				provider := m.getCurrentProvider()
				m.config.apiKeyCache[provider] = apiKey

				if m.step == stepFastAPIKey {
					m.config.fastModel.apiKey = apiKey
					m.config.fastModel.validated = true
				} else {
					m.config.slowModel.apiKey = apiKey
					m.config.slowModel.validated = true
				}

				m.step = m.step + 1
				m.textInput.Blur()
				m.initModelList(provider, apiKey)
			case tea.KeyEsc:
				m.step = m.step - 1
				m.errorMsg = ""
			}
			m.textInput, cmd = m.textInput.Update(msg)
			return m, cmd

		case stepFastModel, stepSlowModel:
			switch msg.String() {
			case "enter":
				if m.modelList.FilterState() == list.Filtering {
					break // let the list confirm the filter
				}
				if item, ok := m.modelList.SelectedItem().(modelItem); ok {
					if m.step == stepFastModel {
						m.config.fastModel.modelID = item.modelID
					} else {
						m.config.slowModel.modelID = item.modelID
					}
					m.step = m.step + 1
					m.testingInProgress = true
					return m, tea.Tick(time.Second*1, func(t time.Time) tea.Msg {
						return tickMsg{}
					})
				}
			case "esc":
				if m.modelList.FilterState() != list.Unfiltered {
					break // let the list cancel/clear the filter
				}
				m.step = m.step - 2
				m.errorMsg = ""
			}
			m.modelList, cmd = m.modelList.Update(msg)
			return m, cmd

		case stepFastTest, stepSlowTest:
			if !m.testingInProgress && msg.Type == tea.KeyEnter {
				if m.step == stepFastTest {
					if m.config.fastModel.testError != "" {
						m.step = stepFastAPIKey
					} else {
						m.step = stepSlowProvider
						m.initProviderList()
					}
				} else {
					if m.config.slowModel.testError != "" {
						m.step = stepSlowAPIKey
					} else {
						m.step = stepSummary
					}
				}
				m.errorMsg = ""
			}

		case stepSummary:
			switch msg.Type {
			case tea.KeyEnter:
				if err := m.saveConfig(); err != nil {
					m.errorMsg = fmt.Sprintf("Failed to save configuration: %v", err)
					return m, nil
				}
				m.step = stepComplete
			case tea.KeyEsc:
				m.step = stepSlowTest
			}

		case stepComplete:
			if msg.Type == tea.KeyEnter || msg.Type == tea.KeyEsc {
				m.quitting = true
				return m, tea.Quit
			}
		}
	case testCompleteMsg:
		m.testingInProgress = false
		if msg.err != nil {
			if m.step == stepFastTest {
				m.config.fastModel.testError = msg.err.Error()
			} else {
				m.config.slowModel.testError = msg.err.Error()
			}
		}
		return m, nil

	case tickMsg:
		if m.testingInProgress {
			var configToTest modelConfig
			if m.step == stepFastTest {
				configToTest = m.config.fastModel
			} else {
				configToTest = m.config.slowModel
			}

			success, err := testConnection(configToTest)
			return m, func() tea.Msg {
				return testCompleteMsg{success: success, err: err}
			}
		}

	default:
		// Forward unhandled messages (e.g. FilterMatchesMsg) to the model list
		if m.step == stepFastModel || m.step == stepSlowModel {
			m.modelList, cmd = m.modelList.Update(msg)
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m wizardModel) View() string {
	if m.quitting {
		return ""
	}

	availableWidth := m.width - 4
	if availableWidth < 20 {
		availableWidth = 20
	}
	availableHeight := m.height - 4
	if availableHeight < 5 {
		availableHeight = 5
	}

	var content strings.Builder
	var title string
	var helpText string

	switch m.step {
	case stepWelcome:
		title = "Welcome to Bishop Setup"
		helpText = "Press Enter or Space to continue"
		content.WriteString(m.renderWelcome())

	case stepFastProvider, stepSlowProvider:
		modelType := "Fast"
		if m.step == stepSlowProvider {
			modelType = "Slow"
		}
		title = fmt.Sprintf("Configure %s Model Provider", modelType)
		helpText = "↑/↓: Navigate | Enter: Select | Esc: Back"
		content.WriteString(m.renderProviderSelection())

	case stepFastAPIKey, stepSlowAPIKey:
		provider := m.getCurrentProvider()
		title = fmt.Sprintf("Enter %s API Key", cases.Title(language.English).String(provider))
		helpText = "Enter: Save | Esc: Back"
		content.WriteString(m.renderAPIKeyEntry())

	case stepFastModel, stepSlowModel:
		modelType := "Fast"
		if m.step == stepSlowModel {
			modelType = "Slow"
		}
		title = fmt.Sprintf("Configure %s Model", modelType)
		helpText = "Type to filter | ↑/↓: Navigate | Enter: Select | Esc: Back"
		content.WriteString(m.renderModelSelection())

	case stepFastTest, stepSlowTest:
		modelType := "Fast"
		if m.step == stepSlowTest {
			modelType = "Slow"
		}
		title = fmt.Sprintf("Testing %s Model Connection", modelType)
		if m.testingInProgress {
			helpText = "Testing connection..."
		} else {
			helpText = "Enter: Continue"
		}
		content.WriteString(m.renderTestResult())

	case stepSummary:
		title = "Configuration Summary"
		helpText = "Enter: Save Configuration | Esc: Back"
		content.WriteString(m.renderSummary())

	case stepComplete:
		title = "Setup Complete!"
		helpText = "Press Enter or Esc to start using Bishop"
		content.WriteString(m.renderComplete())
	}

	stepInfo := fmt.Sprintf("Step %d/%d", m.step+1, stepComplete+1)
	stepText := stepIndicator.Render(stepInfo)

	var boxContent strings.Builder
	boxContent.WriteString(stepText + "\n")

	titlePadding := (availableWidth - len(title)) / 2
	if titlePadding < 0 {
		titlePadding = 0
	}
	boxContent.WriteString(strings.Repeat(" ", titlePadding) + titleStyle.Render(title) + "\n\n")

	contentStr := content.String()
	contentLines := strings.Split(contentStr, "\n")
	contentHeight := availableHeight - 4
	if len(contentLines) > contentHeight {
		contentLines = contentLines[:contentHeight]
	}
	boxContent.WriteString(strings.Join(contentLines, "\n"))

	currentLines := len(contentLines) + 3
	for i := currentLines; i < availableHeight-2; i++ {
		boxContent.WriteString("\n")
	}

	footerContent := helpStyle.Render(helpText)
	if m.errorMsg != "" {
		footerContent = errorStyle.Render(m.errorMsg) + "\n" + footerContent
	}
	boxContent.WriteString("\n" + footerContent)

	styledBox := boxStyle.Width(availableWidth).Height(availableHeight)
	return styledBox.Render(boxContent.String())
}

func (m wizardModel) getCurrentProvider() string {
	if m.step == stepFastAPIKey || m.step == stepFastModel || m.step == stepFastTest {
		return m.config.fastModel.provider
	}
	return m.config.slowModel.provider
}

func (m wizardModel) getCurrentConfig() *modelConfig {
	if m.step == stepFastAPIKey || m.step == stepFastModel || m.step == stepFastTest {
		return &m.config.fastModel
	}
	return &m.config.slowModel
}

func (m *wizardModel) initProviderList() {
	items := []list.Item{
		providerItem{
			title:       "Ollama",
			description: "Local LLM (recommended for privacy, no API key needed)",
			provider:    "ollama",
		},
		providerItem{
			title:       "OpenAI",
			description: "GPT models from OpenAI (requires API key)",
			provider:    "openai",
		},
		providerItem{
			title:       "OpenRouter",
			description: "Access many LLM providers (requires API key)",
			provider:    "openrouter",
		},
	}
	m.providerList.SetItems(items)
}

func (m *wizardModel) initModelList(provider string, apiKey string) {
	var items []list.Item

	if provider == "ollama" {
		apiKey = "ollama"
	}
	if apiKey == "" {
		apiKey = "ollama"
	}

	baseURL := getDefaultBaseURL(provider)

	clientConfig := openai.DefaultConfig(apiKey)
	clientConfig.BaseURL = baseURL

	client := openai.NewClientWithConfig(clientConfig)

	models, err := client.ListModels(context.Background())
	if err != nil {
		items = []list.Item{
			modelItem{title: "Error fetching models", description: err.Error(), modelID: ""},
		}
	} else {
		items = make([]list.Item, 0, len(models.Models))
		for _, model := range models.Models {
			title := model.ID
			description := "Available model"

			switch provider {
			case "openai":
				switch model.ID {
				case "gpt-4o":
					description = "Latest GPT-4 (recommended)"
				case "gpt-4o-mini":
					description = "Faster, more cost-effective"
				}
			case "openrouter":
				if strings.Contains(model.ID, "gpt-4o") {
					description = "Via OpenRouter"
				} else if strings.Contains(model.ID, "claude") {
					description = "High quality"
				}
			}

			items = append(items, modelItem{
				title:       title,
				description: description,
				modelID:     model.ID,
			})
		}

		sort.Slice(items, func(i, j int) bool {
			return items[i].(modelItem).title < items[j].(modelItem).title
		})
	}

	if len(items) == 0 {
		items = []list.Item{
			modelItem{title: "No models found", description: "Check your API key and connection", modelID: ""},
		}
	}

	m.modelList.SetItems(items)
	m.modelList.ResetFilter()
}

func (m wizardModel) saveConfig() error {
	config := m.config

	if config.fastModel.provider != "" {
		saveEnvVar(m.runner, "BISH_FAST_MODEL_PROVIDER", config.fastModel.provider)
		if config.fastModel.apiKey != "" {
			saveEnvVar(m.runner, "BISH_FAST_MODEL_API_KEY", config.fastModel.apiKey)
		}
		if config.fastModel.baseURL != "" {
			saveEnvVar(m.runner, "BISH_FAST_MODEL_BASE_URL", config.fastModel.baseURL)
		}
		if config.fastModel.modelID != "" {
			saveEnvVar(m.runner, "BISH_FAST_MODEL_ID", config.fastModel.modelID)
		}
	}

	if config.slowModel.provider != "" {
		saveEnvVar(m.runner, "BISH_SLOW_MODEL_PROVIDER", config.slowModel.provider)
		if config.slowModel.apiKey != "" {
			saveEnvVar(m.runner, "BISH_SLOW_MODEL_API_KEY", config.slowModel.apiKey)
		}
		if config.slowModel.baseURL != "" {
			saveEnvVar(m.runner, "BISH_SLOW_MODEL_BASE_URL", config.slowModel.baseURL)
		}
		if config.slowModel.modelID != "" {
			saveEnvVar(m.runner, "BISH_SLOW_MODEL_ID", config.slowModel.modelID)
		}
	}

	return saveConfigToFile(config)
}

func saveEnvVar(runner *interp.Runner, key, value string) {
	runner.Vars[key] = expand.Variable{
		Exported: true,
		Kind:     expand.String,
		Str:      value,
	}
}

func homeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}

func getDefaultBaseURL(provider string) string {
	switch provider {
	case "openai":
		return "https://api.openai.com/v1"
	case "openrouter":
		return "https://openrouter.ai/api/v1"
	default:
		return "http://localhost:11434/v1/"
	}
}

func clearScreen() {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", "cls")
	} else {
		cmd = exec.Command("clear")
	}
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}
