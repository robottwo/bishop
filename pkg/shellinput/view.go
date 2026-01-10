package shellinput

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/ansi"
	"github.com/muesli/reflow/wrap"
	"github.com/rivo/uniseg"
)

// View renders the input field with prompt, cursor, and inline suggestions.
func (m Model) View() string {
	if m.inReverseSearch {
		// When in reverse search mode, show the search prompt
		matchText := ""
		prefix := "(reverse-i-search)"

		// Use rich history state to determine if there are matches and what the selected one is
		if len(m.historySearchState.filteredIndices) > 0 {
			selectedIdx := m.historySearchState.selected
			if selectedIdx >= 0 && selectedIdx < len(m.historySearchState.filteredIndices) {
				originalIdx := m.historySearchState.filteredIndices[selectedIdx]
				if originalIdx >= 0 && originalIdx < len(m.historyItems) {
					matchText = m.historyItems[originalIdx].Command
				}
			}
		} else if m.reverseSearchQuery != "" {
			prefix = "(failed reverse-i-search)"
		}

		return m.ReverseSearchPromptStyle.Render(fmt.Sprintf("%s`%s': %s", prefix, m.reverseSearchQuery, matchText))
	}

	styleText := m.TextStyle.Inline(true).Render

	value := m.values[m.selectedValueIndex]
	pos := max(0, m.pos)
	v := m.PromptStyle.Render(m.Prompt) + styleText(m.echoTransform(string(value[:pos])))

	if pos < len(value) { //nolint:nestif
		char := m.echoTransform(string(value[pos]))
		m.Cursor.SetChar(char)
		v += m.Cursor.View()                                   // cursor and text under it
		v += styleText(m.echoTransform(string(value[pos+1:]))) // text after cursor
		v += m.completionView(0)                               // suggested completion
	} else {
		if m.canAcceptSuggestion() {
			suggestion := m.matchedSuggestions[m.currentSuggestionIndex]
			if len(value) < len(suggestion) {
				m.Cursor.TextStyle = m.CompletionStyle
				m.Cursor.SetChar(m.echoTransform(string(suggestion[pos])))
				v += m.Cursor.View()
				v += m.completionView(1)
			} else {
				m.Cursor.SetChar(" ")
				v += m.Cursor.View()
			}
		} else {
			m.Cursor.SetChar(" ")
			v += m.Cursor.View()
		}
		v += m.completionSuffixView() // suffix from active completion (e.g., "/" for directories)
	}

	totalWidth := uniseg.StringWidth(v)

	// If a max width is set, we need to respect the horizontal boundary
	if m.Width > 0 {
		if totalWidth <= m.Width {
			// fill empty spaces with the background color
			padding := max(0, m.Width-totalWidth)
			if totalWidth+padding <= m.Width && pos < len(value) {
				padding++
			}
			v += styleText(strings.Repeat(" ", padding))
		} else {
			v = wrap.String(v, m.Width)
		}
	}

	return v
}

// completionView renders the inline completion suggestion from the current position.
func (m Model) completionView(offset int) string {
	var (
		value = m.values[m.selectedValueIndex]
		style = m.CompletionStyle.Inline(true).Render
	)

	if m.canAcceptSuggestion() {
		suggestion := m.matchedSuggestions[m.currentSuggestionIndex]
		if len(value) < len(suggestion) {
			return style(string(suggestion[len(value)+offset:]))
		}
	}
	return ""
}

// completionSuffixView renders the suffix from the currently selected completion candidate
// as a greyed-out inline suggestion (e.g., "/" for directories)
func (m Model) completionSuffixView() string {
	// Only show suffix if completion is active and a suggestion is selected
	if !m.completion.active || m.completion.selected < 0 || m.completion.selected >= len(m.completion.suggestions) {
		return ""
	}

	// Get the currently selected completion candidate
	candidate := m.completion.suggestions[m.completion.selected]

	// If there's a suffix, render it with the completion style (greyed out)
	if candidate.Suffix != "" {
		return m.CompletionStyle.Inline(true).Render(candidate.Suffix)
	}

	return ""
}

// echoTransform transforms the input value based on the echo mode (normal, password, none).
func (m Model) echoTransform(v string) string {
	switch m.EchoMode {
	case EchoPassword:
		return strings.Repeat(string(m.EchoCharacter), uniseg.StringWidth(v))
	case EchoNone:
		return ""
	case EchoNormal:
		return v
	default:
		return v
	}
}

// CompletionBoxView renders the completion suggestions box with multi-column support.
func (m Model) CompletionBoxView(height int, width int) string {
	if !m.completion.shouldShowInfoBox() {
		return ""
	}

	if height <= 0 {
		height = 4 // default fallback
	}

	totalItems := len(m.completion.suggestions)
	if totalItems == 0 {
		return ""
	}

	// Check if we need to show descriptions (Zsh style)
	hasDescriptions := false
	maxCandidateWidth := 0
	maxItemWidth := 0
	for _, s := range m.completion.suggestions {
		if s.Description != "" {
			hasDescriptions = true
		}

		// Use ansi.PrintableRuneWidth to get visual width without ANSI codes
		displayWidth := 0
		if s.Display != "" {
			displayWidth = ansi.PrintableRuneWidth(s.Display)
		} else {
			displayWidth = ansi.PrintableRuneWidth(s.Value)
		}
		if displayWidth > maxCandidateWidth {
			maxCandidateWidth = displayWidth
		}

		// Length + prefix ("> ") + spacing ("  ")
		l := displayWidth + 4
		if l > maxItemWidth {
			maxItemWidth = l
		}
	}

	// Ensure at least some width
	if maxItemWidth < 10 {
		maxItemWidth = 10
	}

	// Calculate columns - single column when showing descriptions for alignment
	numColumns := 1
	if !hasDescriptions && width > 0 {
		numColumns = width / maxItemWidth
		if numColumns < 1 {
			numColumns = 1
		}
	}

	// If items <= height, we stick to 1 column regardless of width (looks cleaner)
	if totalItems <= height {
		numColumns = 1
	}

	capacity := height * numColumns

	// Calculate visible window
	var startIdx int
	selectedIdx := m.completion.selected
	if selectedIdx < 0 {
		selectedIdx = 0
	}

	// Page-based scrolling logic
	page := selectedIdx / capacity
	startIdx = page * capacity

	// Ensure bounds are valid
	if startIdx < 0 {
		startIdx = 0
	}

	var content strings.Builder

	// Render rows
	for r := 0; r < height; r++ {
		lineContent := ""

		for c := 0; c < numColumns; c++ {
			idx := startIdx + c*height + r
			if idx >= totalItems {
				continue
			}

			candidate := m.completion.suggestions[idx]
			displayText := candidate.Display
			if displayText == "" {
				displayText = candidate.Value
			}

			var prefix string

			// Regular line with spacing
			prefix = " "

			// Add selection indicator
			if idx == m.completion.selected {
				prefix += "> "
			} else {
				prefix += "  "
			}

			itemStr := prefix + displayText

			if hasDescriptions {
				// Render as two columns: Candidate | Description
				// Pad the candidate to align descriptions
				// Use ansi.PrintableRuneWidth to get visual width without ANSI codes
				visualWidth := ansi.PrintableRuneWidth(displayText)
				padding := maxCandidateWidth - visualWidth + 2
				itemStr += strings.Repeat(" ", padding)
				itemStr += lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(candidate.Description)
			} else {
				// Pad the column (except the last one)
				if c < numColumns-1 {
					// Use ansi.PrintableRuneWidth to get visual width without ANSI codes
					itemWidth := ansi.PrintableRuneWidth(itemStr)
					if itemWidth < maxItemWidth {
						itemStr += strings.Repeat(" ", maxItemWidth-itemWidth)
					} else {
						itemStr += "  "
					}
				}
			}

			lineContent += itemStr
		}

		if lineContent != "" {
			content.WriteString(lineContent)
		}

		if r < height-1 {
			content.WriteString("\n")
		}
	}

	return content.String()
}

// HelpBoxView renders the completion help information box.
func (m Model) HelpBoxView() string {
	if !m.completion.shouldShowHelpBox() {
		return ""
	}

	return m.completion.helpInfo
}
