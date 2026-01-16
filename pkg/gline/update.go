package gline

import (
	"context"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case LLMTickMsg:
		m.llmIndicator.Update()
		if m.llmIndicator.GetStatus() == LLMStatusInFlight {
			return m, m.llmIndicator.Tick()
		}
		return m, nil

	case resourceMsg:
		m.borderStatus.UpdateResources(msg.resources)
		// Schedule next update based on configured interval
		interval := time.Duration(m.options.ResourceUpdateInterval) * time.Second
		return m, tea.Tick(interval, func(t time.Time) tea.Msg {
			// Instead of returning resourceMsg directly (which would block if done synchronously),
			// we trigger another fetch command which runs in a goroutine
			return "fetch_resources_trigger"
		})

	case string:
		if msg == "fetch_resources_trigger" {
			return m, m.fetchResources()
		}

	case gitStatusMsg:
		if msg.status != nil {
			m.borderStatus.UpdateGit(msg.status)
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.height = msg.Height
		m.textInput.Width = msg.Width
		m.explanationStyle = m.explanationStyle.Width(max(1, msg.Width-2))
		m.completionStyle = m.completionStyle.Width(max(1, msg.Width-2))
		m.borderStatus.SetWidth(max(0, msg.Width-2))
		return m, nil

	case terminateMsg:
		m.appState = Terminated
		return m, nil

	case interruptMsg:
		m.appState = Terminated
		m.interrupted = true
		return m, nil

	case attemptPredictionMsg:
		m.llmIndicator.SetStatus(LLMStatusInFlight)
		model, cmd := m.attemptPrediction(msg)
		return model, tea.Batch(cmd, m.llmIndicator.Tick())

	case setPredictionMsg:
		return m.setPrediction(msg.stateId, msg.prediction, msg.inputContext)

	case attemptExplanationMsg:
		return m.attemptExplanation(msg)

	case setExplanationMsg:
		return m.setExplanation(msg)

	case errorMsg:
		if msg.stateId == m.predictionStateId {
			m.lastError = msg.err
			m.llmIndicator.SetStatus(LLMStatusError)
			m.prediction = ""
			m.explanation = ""
			m.textInput.SetSuggestions([]string{})
		}
		return m, nil

	case idleCheckMsg:
		return m.handleIdleCheck(msg)

	case setIdleSummaryMsg:
		return m.handleSetIdleSummary(msg)

	case tea.KeyMsg:
		switch msg.String() {

		case "esc":
			// Dismiss idle summary if shown, otherwise ignore
			if m.idleSummaryShown {
				m.dismissIdleSummary()
				return m, m.scheduleIdleCheck()
			}
			return m, nil

		// TODO: replace with custom keybindings
		case "backspace":
			if !m.textInput.InReverseSearch() {
				// if the input is already empty, we should clear prediction and restore default tip
				if m.textInput.Value() == "" {
					m.dirty = true
					m.predictionStateId++
					m.clearPredictionAndRestoreDefault()
					return m, nil
				}
			}

		case "enter":
			if m.textInput.InReverseSearch() {
				break
			}

			input := m.textInput.Value()

			// Handle multiline input with error handling
			complete, prompt := m.multilineState.AddLine(input)
			if !complete {
				// Need more input, update prompt and continue
				m.textInput.Prompt = prompt + " "
				// Clear the text input field but preserve the multiline buffer
				m.textInput.SetValue("")
				return m, nil
			}

			// We have a complete command - add error handling for GetCompleteCommand
			result := m.multilineState.GetCompleteCommand()
			if result == "" && input != "" {
				// Only treat empty result as error if input was not empty
				// Reset the multiline state and continue
				m.multilineState.Reset()
				m.textInput.SetValue("")
				return m, nil
			}

			m.result = result
			return m, tea.Sequence(terminate, tea.Quit)

		case "ctrl+c":
			if m.textInput.InReverseSearch() {
				break
			}

			// Handle Ctrl-C: cancel current line, preserve input with "^C" appended, and present fresh prompt

			// Set result to empty string so shell doesn't try to execute it
			m.result = ""
			// Use interrupt message to indicate Ctrl+C was pressed
			// We do not reset multiline state here so that Gline() can reconstruct the full input
			return m, tea.Sequence(interrupt, tea.Quit)
		case "ctrl+d":
			// Handle Ctrl-D: exit shell if on blank line
			currentInput := m.textInput.Value()
			if strings.TrimSpace(currentInput) == "" {
				// On blank line, exit the shell
				m.result = "exit"
				return m, tea.Sequence(terminate, tea.Quit)
			}
			// If there's content, do nothing (standard behavior)
			return m, nil
		case "ctrl+l":
			return m.handleClearScreen()
		}
	}

	return m.updateTextInput(msg)
}

func (m *appModel) clearPrediction() {
	m.prediction = ""
	m.explanation = ""
	m.lastError = nil
	m.textInput.SetSuggestions([]string{})
}

// clearPredictionAndRestoreDefault clears the prediction and restores the default
// explanation (e.g., coach tips) - used when the input buffer becomes blank
func (m *appModel) clearPredictionAndRestoreDefault() {
	m.prediction = ""
	m.explanation = m.defaultExplanation
	m.lastError = nil
	m.textInput.SetSuggestions([]string{})
}

func (m appModel) setPrediction(stateId int, prediction string, inputContext string) (appModel, tea.Cmd) {
	if stateId != m.predictionStateId {
		m.logger.Debug(
			"gline discarding prediction",
			zap.Int("startStateId", stateId),
			zap.Int("newStateId", m.predictionStateId),
		)
		return m, nil
	}

	m.prediction = prediction
	m.lastPredictionInput = inputContext
	m.lastPrediction = prediction
	m.textInput.SetSuggestions([]string{prediction})
	m.textInput.UpdateHelpInfo()

	// When input is blank and there's no prediction, preserve the default explanation (coach tips)
	if strings.TrimSpace(m.textInput.Value()) == "" && prediction == "" {
		m.explanation = m.defaultExplanation
		// Reset LLM status to prevent pulsing when showing coaching tips
		m.llmIndicator.SetStatus(LLMStatusSuccess)
		return m, nil
	}

	m.explanation = ""
	explanationTarget := prediction
	if m.textInput.SuggestionsSuppressedUntilInput() {
		explanationTarget = m.textInput.Value()
	}

	return m, tea.Cmd(func() tea.Msg {
		return attemptExplanationMsg{stateId: m.predictionStateId, prediction: explanationTarget}
	})
}

// LLM call timeout for predictions
const predictionTimeout = 10 * time.Second

func (m appModel) attemptPrediction(msg attemptPredictionMsg) (tea.Model, tea.Cmd) {
	if m.predictor == nil {
		return m, nil
	}
	if msg.stateId != m.predictionStateId {
		return m, nil
	}
	// Skip LLM prediction for # commands (agentic commands)
	if strings.HasPrefix(strings.TrimSpace(m.textInput.Value()), "#") {
		// Don't show indicator when buffer is empty - just return clean state
		return m, nil
	}

	return m, tea.Cmd(func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), predictionTimeout)
		defer cancel()

		prediction, inputContext, err := m.predictor.Predict(ctx, m.textInput.Value())
		if err != nil {
			m.logger.Error("gline prediction failed", zap.Error(err))
			return errorMsg{stateId: msg.stateId, err: err}
		}

		m.logger.Debug(
			"gline predicted input",
			zap.Int("stateId", msg.stateId),
			zap.String("prediction", prediction),
			zap.String("inputContext", inputContext),
		)
		return setPredictionMsg{stateId: msg.stateId, prediction: prediction, inputContext: inputContext}
	})
}
