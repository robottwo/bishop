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
	homeDir := getHomeDir()
	configPath := homeDir + "/.bishrc"

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return true
	}

	configUIPath := homeDir + "/.bish_config_ui"
	if _, err := os.Stat(configUIPath); os.IsNotExist(err) {
		return true
	}

	return false
}

func getHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return os.Getenv("HOME")
	}
	return home
}
