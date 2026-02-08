package wizard

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func (m wizardModel) renderWelcome() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62")).Render("Welcome to Bishop!") + "\n\n")

	b.WriteString("Bishop is a modern, POSIX-compatible, generative shell.\n\n")

	b.WriteString("Before we get started, let's configure your AI models.\n\n")

	b.WriteString("Bishop uses two types of models:\n")
	b.WriteString("  • Fast Model: For auto-completion and suggestions\n")
	b.WriteString("  • Slow Model: For chat and agent operations\n\n")

	b.WriteString("You can choose from these providers:\n")
	b.WriteString("  • Ollama: Local LLM (no API key, privacy-focused)\n")
	b.WriteString("  • OpenAI: GPT models (requires API key)\n")
	b.WriteString("  • OpenRouter: Access many LLM providers (requires API key)\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Press Enter or Space to continue..."))

	return b.String()
}

func (m wizardModel) renderProviderSelection() string {
	var b strings.Builder

	modelType := "Fast"
	if m.step == stepSlowProvider {
		modelType = "Slow"
	}

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Choose a provider for your "+modelType+" model.") + "\n\n")

	b.WriteString("The " + strings.ToLower(modelType) + " model is used for:\n")
	if modelType == "Fast" {
		b.WriteString("  • Auto-completion as you type\n")
		b.WriteString("  • Command predictions\n")
		b.WriteString("  • Quick suggestions\n\n")
		b.WriteString("Ollama is recommended for fast models because it runs locally.\n")
	} else {
		b.WriteString("  • Chat conversations\n")
		b.WriteString("  • Agent operations\n")
		b.WriteString("  • Complex tasks\n\n")
		b.WriteString("Choose based on your quality vs latency preferences.\n")
	}

	b.WriteString("\n" + m.providerList.View())

	return b.String()
}

func (m wizardModel) renderAPIKeyEntry() string {
	var b strings.Builder

	provider := m.getCurrentProvider()

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Enter your "+cases.Title(language.English).String(provider)+" API key") + "\n\n")

	b.WriteString("Your API key will be stored in ~/.bish_config_ui.\n")
	b.WriteString("For security, this file should only be readable by you.\n\n")

	switch provider {
	case "openai":
		b.WriteString("Get your API key from: https://platform.openai.com/api-keys\n")
		b.WriteString("Your key should start with 'sk-'\n\n")
	case "openrouter":
		b.WriteString("Get your API key from: https://openrouter.ai/keys\n")
		b.WriteString("Your key should start with 'sk-or-'\n\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("API Key:") + "\n")
	b.WriteString(m.textInput.View() + "\n")

	return b.String()
}

func (m wizardModel) renderModelSelection() string {
	var b strings.Builder

	modelType := "Fast"
	if m.step == stepSlowModel {
		modelType = "Slow"
	}

	currentProvider := m.getCurrentProvider()

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Choose a model for your "+modelType+" "+cases.Title(language.English).String(currentProvider)+" setup") + "\n\n")

	if modelType == "Fast" {
		b.WriteString("For the fast model, prioritize speed over quality.\n")
	} else {
		b.WriteString("For the slow model, prioritize quality over speed.\n")
	}

	b.WriteString("\n" + m.modelList.View())

	return b.String()
}

func (m wizardModel) renderTestResult() string {
	var b strings.Builder

	config := m.getCurrentConfig()

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Testing connection to "+cases.Title(language.English).String(config.provider)) + "\n\n")

	b.WriteString("Configuration:\n")
	b.WriteString("  Provider: " + config.provider + "\n")
	b.WriteString("  Model: " + config.modelID + "\n")
	if config.baseURL != "" {
		b.WriteString("  Base URL: " + config.baseURL + "\n")
	}
	b.WriteString("\n")

	if m.testingInProgress {
		b.WriteString(m.progress.View() + "\n")
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Italic(true).Render("Testing connection..."))
	} else {
		if config.testError != "" {
			b.WriteString(errorStyle.Render("✗ Connection failed") + "\n\n")
			b.WriteString("Error: " + config.testError + "\n\n")
			b.WriteString("Press Enter to go back and fix the configuration.")
		} else {
			b.WriteString(successStyle.Render("✓ Connection successful!") + "\n\n")
			b.WriteString("Your configuration is working correctly.\n\n")
			b.WriteString("Press Enter to continue.")
		}
	}

	return b.String()
}

func (m wizardModel) renderSummary() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("Configuration Summary") + "\n\n")

	b.WriteString("Please review your configuration before saving:\n\n")

	if m.config.fastModel.provider != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("Fast Model (Completions):") + "\n")
		b.WriteString("  Provider: " + m.config.fastModel.provider + "\n")
		b.WriteString("  Model: " + m.config.fastModel.modelID + "\n")
		if m.config.fastModel.apiKey != "" {
			b.WriteString("  API Key: " + maskAPIKey(m.config.fastModel.apiKey) + "\n")
		}
		if m.config.fastModel.baseURL != "" {
			b.WriteString("  Base URL: " + m.config.fastModel.baseURL + "\n")
		}
		b.WriteString("\n")
	}

	if m.config.slowModel.provider != "" {
		b.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("170")).Render("Slow Model (Chat/Agent):") + "\n")
		b.WriteString("  Provider: " + m.config.slowModel.provider + "\n")
		b.WriteString("  Model: " + m.config.slowModel.modelID + "\n")
		if m.config.slowModel.apiKey != "" {
			b.WriteString("  API Key: " + maskAPIKey(m.config.slowModel.apiKey) + "\n")
		}
		if m.config.slowModel.baseURL != "" {
			b.WriteString("  Base URL: " + m.config.slowModel.baseURL + "\n")
		}
		b.WriteString("\n")
	}

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Configuration will be saved to: ~/.bish_config_ui"))

	return b.String()
}

func (m wizardModel) renderComplete() string {
	var b strings.Builder

	b.WriteString("\n")
	b.WriteString(successStyle.Render("✓ Setup Complete!") + "\n\n")

	b.WriteString("Your Bishop configuration has been saved.\n\n")

	b.WriteString("You can now start using Bishop!\n\n")

	b.WriteString("Quick tips:\n")
	b.WriteString("  • Type #!config to change settings anytime\n")
	b.WriteString("  • Type # followed by a message to chat with the agent\n")
	b.WriteString("  • Type #!setup to run this wizard again\n")
	b.WriteString("  • Type #? to get help fixing errors\n\n")

	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("Press Enter or Esc to start using Bishop"))

	return b.String()
}

func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return "***"
	}
	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}
