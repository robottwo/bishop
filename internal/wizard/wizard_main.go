package wizard

import (
	"mvdan.cc/sh/v3/interp"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func RunWizard(runner *interp.Runner) error {
	clearScreen()

	model := initialModel(runner)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()

	if err == nil {
		clearScreen()
	}

	return err
}

func NeedsSetup() bool {
	// Already configured via the wizard/config TUI.
	configUIFile := getHomeDir() + "/.config/bish/config_ui"
	if _, err := os.Stat(configUIFile); !os.IsNotExist(err) {
		return false
	}

	// Already configured manually via environment variables.
	if os.Getenv("BISH_FAST_MODEL_PROVIDER") != "" || os.Getenv("BISH_SLOW_MODEL_PROVIDER") != "" {
		return false
	}

	return true
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}
