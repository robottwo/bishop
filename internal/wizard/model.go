package wizard

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"mvdan.cc/sh/v3/interp"
)

type wizardStep int

const (
	stepWelcome wizardStep = iota
	stepFastProvider
	stepFastAPIKey
	stepFastModel
	stepFastTest
	stepSlowProvider
	stepSlowAPIKey
	stepSlowModel
	stepSlowTest
	stepSummary
	stepComplete
)

type modelConfig struct {
	provider  string
	apiKey    string
	baseURL   string
	modelID   string
	validated bool
	testError string
}

type wizardConfig struct {
	fastModel   modelConfig
	slowModel   modelConfig
	apiKeyCache map[string]string // Cache API keys by provider for reuse
}

type wizardModel struct {
	runner   *interp.Runner
	step     wizardStep
	config   wizardConfig
	width    int
	height   int
	quitting bool
	errorMsg string

	providerList list.Model
	textInput    textinput.Model
	modelList    list.Model
	helpViewport viewport.Model
	progress     progress.Model

	testingInProgress bool
}

func initialModel(runner *interp.Runner) wizardModel {
	m := wizardModel{
		runner:   runner,
		step:     stepWelcome,
		config:   wizardConfig{apiKeyCache: make(map[string]string)},
		quitting: false,
	}

	providerDelegate := list.NewDefaultDelegate()
	providerDelegate.Styles.SelectedTitle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	providerDelegate.Styles.SelectedDesc = providerDelegate.Styles.SelectedTitle.Foreground(lipgloss.Color("240"))

	providerList := list.New([]list.Item{}, providerDelegate, 0, 0)
	providerList.SetShowStatusBar(false)
	providerList.SetFilteringEnabled(false)
	providerList.SetShowTitle(false)
	providerList.SetShowHelp(false)
	m.providerList = providerList

	ti := textinput.New()
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	ti.EchoMode = textinput.EchoPassword
	ti.EchoCharacter = '•'
	m.textInput = ti

	modelDelegate := list.NewDefaultDelegate()
	modelDelegate.Styles.SelectedTitle = lipgloss.NewStyle().PaddingLeft(2).Foreground(lipgloss.Color("170"))
	modelDelegate.Styles.SelectedDesc = modelDelegate.Styles.SelectedTitle.Foreground(lipgloss.Color("240"))

	modelList := list.New([]list.Item{}, modelDelegate, 0, 0)
	modelList.SetShowStatusBar(false)
	modelList.SetShowTitle(false)
	modelList.SetShowHelp(false)
	modelList.SetFilteringEnabled(true)
	modelList.SetShowFilter(true)
	m.modelList = modelList

	helpViewport := viewport.New(0, 0)
	m.helpViewport = helpViewport

	progress := progress.New(progress.WithDefaultGradient())
	m.progress = progress

	return m
}

func (m wizardModel) Init() tea.Cmd {
	return nil
}
