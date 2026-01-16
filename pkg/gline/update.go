package gline

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
