package gline

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/robottwo/bishop/internal/git"
	"github.com/robottwo/bishop/internal/system"
	"github.com/robottwo/bishop/pkg/shellinput"
	"go.uber.org/zap"
)

type appModel struct {
	predictor Predictor
	explainer Explainer
	analytics PredictionAnalytics
	logger    *zap.Logger
	options   Options

	textInput           shellinput.Model
	dirty               bool
	prediction          string
	explanation         string
	defaultExplanation  string // Shown when buffer is blank (e.g., coach tips)
	lastError           error
	lastPredictionInput string
	lastPrediction      string
	predictionStateId   int

	historyValues []string
	result        string
	appState      appState
	interrupted   bool

	explanationStyle lipgloss.Style
	completionStyle  lipgloss.Style
	errorStyle       lipgloss.Style
	coachTipStyle    lipgloss.Style

	// Multiline support
	multilineState *MultilineState
	originalPrompt string
	height         int

	// LLM status indicator
	llmIndicator LLMIndicator

	// Border Status
	borderStatus BorderStatusModel

	// Idle summary tracking
	lastInputTime        time.Time
	idleSummaryShown     bool
	idleSummaryPending   bool
	idleSummaryStateId   int
	originalCoachTip     string // Stored to restore after dismissing idle summary
	idleSummaryStyle     lipgloss.Style
	idleSummaryHintStyle lipgloss.Style
}

type attemptPredictionMsg struct {
	stateId int
}

type setPredictionMsg struct {
	stateId      int
	prediction   string
	inputContext string
}

type attemptExplanationMsg struct {
	stateId    int
	prediction string
}

// resourceMsg carries updated system resources
type resourceMsg struct {
	resources *system.Resources
}

type gitStatusMsg struct {
	status *git.RepoStatus
}

// errorMsg wraps an error that occurred during prediction or explanation
type errorMsg struct {
	stateId int
	err     error
}

// helpHeaderRegex matches redundant help headers like "**#name** - "
var helpHeaderRegex = regexp.MustCompile(`^\*\*[^\*]+\*\* - `)

type setExplanationMsg struct {
	stateId     int
	explanation string
}

// Idle summary messages
type idleCheckMsg struct {
	stateId int
}

type setIdleSummaryMsg struct {
	stateId int
	summary string
}

// ErrInterrupted is returned when the user presses Ctrl+C
var ErrInterrupted = errors.New("interrupted by user")

type terminateMsg struct{}

func terminate() tea.Msg {
	return terminateMsg{}
}

type interruptMsg struct{}

func interrupt() tea.Msg {
	return interruptMsg{}
}

type appState int

const (
	Active appState = iota
	Terminated
)

func initialModel(
	prompt string,
	historyValues []string,
	explanation string,
	predictor Predictor,
	explainer Explainer,
	analytics PredictionAnalytics,
	logger *zap.Logger,
	options Options,
) appModel {
	textInput := shellinput.New()
	textInput.Prompt = prompt
	textInput.SetHistoryValues(historyValues)
	// Initialize rich history if available
	if len(options.RichHistory) > 0 {
		textInput.SetRichHistory(options.RichHistory)
	}
	if options.CurrentDirectory != "" {
		textInput.SetCurrentDirectory(options.CurrentDirectory)
	}
	if options.CurrentSessionID != "" {
		textInput.SetCurrentSessionID(options.CurrentSessionID)
	}
	// Set initial value if provided (e.g., for editing a suggested fix)
	if options.InitialValue != "" {
		textInput.SetValue(options.InitialValue)
	}
	textInput.Cursor.SetMode(cursor.CursorStatic)
	textInput.ShowSuggestions = true
	textInput.CompletionProvider = options.CompletionProvider
	textInput.Focus()

	borderStatus := NewBorderStatusModel()
	borderStatus.UpdateContext(options.User, options.Host, options.CurrentDirectory)

	return appModel{
		predictor: predictor,
		explainer: explainer,
		analytics: analytics,
		logger:    logger,
		options:   options,

		textInput:          textInput,
		dirty:              options.InitialValue != "", // Mark dirty if we have initial value
		prediction:         "",
		explanation:        explanation,
		defaultExplanation: explanation, // Store for restoring when buffer is blank
		historyValues:      historyValues,
		result:             "",
		appState:           Active,
		interrupted:        false, // Explicitly initialize to prevent stateful behavior

		predictionStateId: 0,

		explanationStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")),
		completionStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("10")),
		errorStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")), // Red
		coachTipStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")), // Faded gray

		// Initialize multiline state
		multilineState: NewMultilineState(),
		originalPrompt: prompt,

		llmIndicator: NewLLMIndicator(),
		borderStatus: borderStatus,

		// Initialize idle summary tracking
		lastInputTime:        time.Now(),
		idleSummaryShown:     false,
		idleSummaryPending:   false,
		idleSummaryStateId:   0,
		originalCoachTip:     explanation, // Store original coach tip
		idleSummaryStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("75")),  // Soft blue for summary
		idleSummaryHintStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("241")), // Subtle gray for hint
	}
}

func (m appModel) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.llmIndicator.Tick(),
		func() tea.Msg {
			return attemptPredictionMsg{
				stateId: m.predictionStateId,
			}
		},
		m.fetchGitStatus(),
	}

	// Only start resource monitoring if enabled (interval > 0)
	if m.options.ResourceUpdateInterval > 0 {
		cmds = append(cmds, m.fetchResources())
	}

	// Start idle check timer if enabled
	if m.options.IdleSummaryTimeout > 0 && m.options.IdleSummaryGenerator != nil {
		cmds = append(cmds, m.scheduleIdleCheck())
	}

	return tea.Batch(cmds...)
}

func (m appModel) scheduleIdleCheck() tea.Cmd {
	stateId := m.idleSummaryStateId
	timeout := time.Duration(m.options.IdleSummaryTimeout) * time.Second
	return tea.Tick(timeout, func(t time.Time) tea.Msg {
		return idleCheckMsg{stateId: stateId}
	})
}

func (m appModel) fetchResources() tea.Cmd {
	return func() tea.Msg {
		res := system.GetResources()
		return resourceMsg{resources: res}
	}
}

func (m appModel) fetchGitStatus() tea.Cmd {
	return func() tea.Msg {
		if m.options.CurrentDirectory == "" {
			return nil
		}
		// Create a context with timeout for git status check
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		status := git.GetStatusWithContext(ctx, m.options.CurrentDirectory)
		return gitStatusMsg{status: status}
	}
}


func (m appModel) updateTextInput(msg tea.Msg) (appModel, tea.Cmd) {
	oldVal := m.textInput.Value()
	oldMatchedSuggestions := m.textInput.MatchedSuggestions()
	oldSuppression := m.textInput.SuggestionsSuppressedUntilInput()
	updatedTextInput, cmd := m.textInput.Update(msg)
	newVal := updatedTextInput.Value()
	newMatchedSuggestions := updatedTextInput.MatchedSuggestions()

	textUpdated := oldVal != newVal
	suggestionsCleared := len(oldMatchedSuggestions) > 0 && len(newMatchedSuggestions) == 0
	m.textInput = updatedTextInput

	// if the text input has changed, we want to attempt a prediction
	if textUpdated && m.predictor != nil {
		m.predictionStateId++

		// Update border status with new input
		m.borderStatus.UpdateInput(newVal)

		// Clear any existing error when user types
		m.lastError = nil

		// Reset idle timer and state when user types
		m.lastInputTime = time.Now()
		m.idleSummaryShown = false
		m.idleSummaryStateId++

		userInput := updatedTextInput.Value()

		// whenever the user has typed something, mark the model as dirty
		if len(userInput) > 0 {
			m.dirty = true
		}

		suppressionActive := updatedTextInput.SuggestionsSuppressedUntilInput()
		suppressionLifted := !suppressionActive && oldSuppression

		switch {
		case len(userInput) == 0 && m.dirty:
			// if the model was dirty earlier, but now the user has cleared the input,
			// we should clear the prediction and restore the default tip
			m.clearPredictionAndRestoreDefault()
		case suppressionActive:
			// When suppression is active (e.g., after Ctrl+K), clear stale predictions but
			// still recompute assistant help for the remaining buffer while keeping
			// autocomplete hints hidden until new input arrives.
			m.clearPrediction()
			if len(userInput) > 0 {
				cmd = tea.Batch(cmd, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
					return attemptPredictionMsg{
						stateId: m.predictionStateId,
					}
				}))
			}
		case len(userInput) > 0 && strings.HasPrefix(m.prediction, userInput) && !suggestionsCleared && !suppressionLifted:
			// if the prediction already starts with the user input, we don't need to predict again
			m.logger.Debug("gline existing predicted input already starts with user input", zap.String("userInput", userInput))
		default:
			// in other cases, we should kick off a debounced prediction after clearing the current one
			m.clearPrediction()

			cmd = tea.Batch(cmd, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
				return attemptPredictionMsg{
					stateId: m.predictionStateId,
				}
			}))
		}
	} else if suggestionsCleared {
		// User trimmed away ghost suggestions (e.g., via Ctrl+K) without changing
		// the underlying input. Clear any pending prediction and explanation so the
		// assistant box reflects the truncated command, and re-request a prediction
		// for the remaining buffer so the assistant can refresh its help content.
		m.clearPrediction()

		if m.predictor != nil {
			m.predictionStateId++
			if len(m.textInput.Value()) > 0 {
				cmd = tea.Batch(cmd, tea.Tick(200*time.Millisecond, func(t time.Time) tea.Msg {
					return attemptPredictionMsg{stateId: m.predictionStateId}
				}))
			}
		}
	}

	return m, cmd
}

func (m appModel) handleClearScreen() (tea.Model, tea.Cmd) {
	// Log the current state before clearing
	m.logger.Debug("gline handleClearScreen called",
		zap.String("currentInput", m.textInput.Value()),
		zap.String("explanation", m.explanation),
		zap.Bool("hasExplanation", m.explanation != ""))

	// Use Bubble Tea's built-in screen clearing command for proper rendering pipeline integration
	// This ensures that lipgloss-styled components like info boxes render correctly after clearing
	m.logger.Debug("gline using tea.ClearScreen for proper rendering pipeline integration")

	// Return the model unchanged with the ClearScreen command
	// Bubble Tea will handle the screen clear and automatic re-render
	return m, tea.Cmd(func() tea.Msg {
		return tea.ClearScreen()
	})
}

// handleIdleCheck checks if the user is idle and triggers summary generation
func (m appModel) handleIdleCheck(msg idleCheckMsg) (tea.Model, tea.Cmd) {
	// Ignore stale idle check messages
	if msg.stateId != m.idleSummaryStateId {
		return m, nil
	}

	// Don't generate if idle summary is disabled or already shown
	if m.options.IdleSummaryTimeout <= 0 || m.options.IdleSummaryGenerator == nil {
		return m, nil
	}

	if m.idleSummaryShown || m.idleSummaryPending {
		return m, nil
	}

	// Check if user input is empty (idle at command prompt)
	if strings.TrimSpace(m.textInput.Value()) != "" {
		// User has typed something, reschedule idle check
		return m, m.scheduleIdleCheck()
	}

	// Check if enough time has passed since last input
	idleTimeout := time.Duration(m.options.IdleSummaryTimeout) * time.Second
	if time.Since(m.lastInputTime) < idleTimeout {
		// Not idle long enough, reschedule
		return m, m.scheduleIdleCheck()
	}

	// User is idle, trigger summary generation
	m.idleSummaryPending = true
	m.logger.Debug("user idle, generating summary",
		zap.Duration("idle_duration", time.Since(m.lastInputTime)),
	)

	stateId := m.idleSummaryStateId
	return m, tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		summary, err := m.options.IdleSummaryGenerator(ctx)
		if err != nil {
			m.logger.Debug("idle summary generation failed", zap.Error(err))
			return setIdleSummaryMsg{stateId: stateId, summary: ""}
		}

		return setIdleSummaryMsg{stateId: stateId, summary: summary}
	})
}

// handleSetIdleSummary sets the idle summary in the assistant box
func (m appModel) handleSetIdleSummary(msg setIdleSummaryMsg) (tea.Model, tea.Cmd) {
	// Ignore stale messages
	if msg.stateId != m.idleSummaryStateId {
		return m, nil
	}

	m.idleSummaryPending = false

	// If no summary (generation failed or no commands), don't update
	if msg.summary == "" {
		return m, nil
	}

	// Store original coach tip before replacing (if not already stored)
	if m.originalCoachTip == "" {
		m.originalCoachTip = m.defaultExplanation
	}

	// Set the summary with explicit header and dismiss hint
	m.idleSummaryShown = true
	m.defaultExplanation = msg.summary
	m.explanation = m.defaultExplanation

	m.logger.Debug("idle summary displayed",
		zap.String("summary", msg.summary),
	)

	return m, nil
}

// dismissIdleSummary restores the original coach tip and resets idle summary state
func (m *appModel) dismissIdleSummary() {
	if !m.idleSummaryShown {
		return
	}

	m.idleSummaryShown = false
	m.idleSummaryStateId++
	m.lastInputTime = time.Now()

	// Restore original coach tip
	if m.originalCoachTip != "" {
		m.defaultExplanation = m.originalCoachTip
		m.explanation = m.defaultExplanation
	}
}

func Gline(
	prompt string,
	historyValues []string,
	explanation string,
	predictor Predictor,
	explainer Explainer,
	analytics PredictionAnalytics,
	logger *zap.Logger,
	options Options,
) (string, error) {
	p := tea.NewProgram(
		initialModel(prompt, historyValues, explanation, predictor, explainer, analytics, logger, options),
	)

	m, err := p.Run()
	if err != nil {
		return "", err
	}

	appModel, ok := m.(appModel)
	if !ok {
		logger.Error("Gline resulted in an unexpected app model")
		panic("Gline resulted in an unexpected app model")
	}

	// Check if the session was interrupted by Ctrl+C
	if appModel.interrupted {
		// Reconstruct what was on screen so it persists
		var inputStr string
		if appModel.multilineState.IsActive() {
			lines := appModel.multilineState.GetLines()
			for i, line := range lines {
				if i == 0 {
					inputStr += appModel.originalPrompt + line + "\n"
				} else {
					inputStr += "> " + line + "\n"
				}
			}
		}
		// Append current line with ^C
		inputStr += appModel.textInput.Prompt + appModel.textInput.Value() + "^C\n"

		fmt.Print(RESET_CURSOR_COLUMN + inputStr)
		return "", ErrInterrupted
	}

	fmt.Print(RESET_CURSOR_COLUMN + appModel.getFinalOutput() + "\n")

	if analytics != nil {
		err = analytics.NewEntry(appModel.lastPredictionInput, appModel.lastPrediction, appModel.result)
		if err != nil {
			logger.Error("failed to log analytics entry", zap.Error(err))
		}
	}

	return appModel.result, nil
}
