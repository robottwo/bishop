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

	// Only auto-launch for true first-run: neither config file exists.
	// Users with an existing ~/.bishrc have already configured bishop
	// (possibly manually) and should not be interrupted.
	configPath := homeDir + "/.bishrc"
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		return false
	}

	configUIFile := homeDir + "/.config/bish/config_ui"
	if _, err := os.Stat(configUIFile); !os.IsNotExist(err) {
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
