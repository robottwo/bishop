package gline

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m appModel) View() string {
	// Once terminated, render nothing
	if m.appState == Terminated {
		return ""
	}

	var inputStr string

	// If we have multiline content, show each line with its original prompt
	if m.multilineState.IsActive() {
		lines := m.multilineState.GetLines()
		for i, line := range lines {
			if i == 0 {
				// First line uses the original prompt (textInput already adds the space)
				inputStr += m.originalPrompt + line + "\n"
			} else {
				// Subsequent lines use continuation prompt
				inputStr += "> " + line + "\n"
			}
		}
	}

	// Add the current input line with appropriate prompt
	inputStr += m.textInput.View()

	// Determine assistant content
	var assistantContent string

	// We need to handle truncation manually because lipgloss Height doesn't truncate automatically
	// Use expanded height when in reverse search mode (close to full screen)
	availableHeight := m.options.AssistantHeight
	if m.textInput.InReverseSearch() && m.height > 0 {
		// Use most of terminal height, leaving room for prompt line (2) and borders (2)
		availableHeight = max(m.options.AssistantHeight, m.height-4)
	}

	// Track if content is pre-formatted (completion/history boxes) and should skip word wrapping
	isPreformatted := false

	// Display error if present
	if m.lastError != nil {
		errorContent := fmt.Sprintf("LLM Inference Error: %s", m.lastError.Error())
		assistantContent = m.errorStyle.Render(errorContent)
	} else {
		// Normal assistant content logic
		helpBox := m.textInput.HelpBoxView()

		// Determine available width for completion box
		completionWidth := max(0, m.textInput.Width-4)
		if helpBox != "" {
			completionWidth = completionWidth / 2
		}

		completionBox := m.textInput.CompletionBoxView(availableHeight, completionWidth)
		historyBox := m.textInput.HistorySearchBoxView(availableHeight, max(0, m.textInput.Width-2))

		if historyBox != "" {
			assistantContent = historyBox
			isPreformatted = true
		} else if completionBox != "" && helpBox != "" {
			// Clean up help box text to avoid redundancy
			// Remove headers like "**#name** - " or "**name** - " using regex
			// This covers patterns like "**#debug-assistant** - " or "**#!new** - "
			helpBox = helpHeaderRegex.ReplaceAllString(helpBox, "")

			// Render side-by-side
			halfWidth := completionWidth // Already calculated

			leftStyle := lipgloss.NewStyle().
				Width(halfWidth).
				Height(availableHeight).
				MaxHeight(availableHeight)

			rightStyle := lipgloss.NewStyle().
				Width(halfWidth).
				Height(availableHeight).
				MaxHeight(availableHeight).
				PaddingLeft(1) // Add some spacing between columns

			// Render completion on left, help on right
			assistantContent = lipgloss.JoinHorizontal(lipgloss.Top,
				leftStyle.Render(completionBox),
				rightStyle.Render(helpBox))
			isPreformatted = true

		} else if completionBox != "" {
			assistantContent = completionBox
			isPreformatted = true
		} else if helpBox != "" {
			assistantContent = helpBox
		} else {
			assistantContent = m.explanation
		}
	}

	// Track if this is a coach tip for styling after word wrap
	isCoachTip := m.explanation == m.defaultExplanation && m.explanation != ""
	isIdleSummary := m.idleSummaryShown && isCoachTip

	// Add header and dismiss hint for idle summaries
	if isIdleSummary && assistantContent != "" {
		header := m.idleSummaryStyle.Render("💭 Idle summary ready")
		hint := m.idleSummaryHintStyle.Render("(Esc to dismiss)")
		assistantContent = header + "\n" + assistantContent + "\n" + hint
	}

	// Render Assistant Box with custom border that includes LLM indicators
	boxWidth := max(0, m.textInput.Width-2)
	borderColor := lipgloss.Color("62")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Word wrap content to fit box width, then split into lines
	innerWidth := max(0, boxWidth-2) // Account for left/right borders
	// Content area is innerWidth minus 2 spaces for left/right padding
	contentWidth := innerWidth - 2
	// Use custom word wrapping that uses GetRuneWidth for accurate Unicode/emoji width calculation
	// This ensures coach tips with emoji render correctly in the assistant box
	// Note: Skip word wrapping for completion/history boxes as they are already formatted with proper columns
	var wrappedContent string
	if isPreformatted {
		// Completion and history boxes are pre-formatted, don't word wrap
		wrappedContent = assistantContent
	} else {
		wrappedContent = WordwrapWithRuneWidth(assistantContent, contentWidth)
	}
	lines := strings.Split(wrappedContent, "\n")

	// Apply faded style to each line of coach tips after word wrapping
	// Skip styling for idle summaries as they have their own styling applied in header/content
	if isCoachTip && !isIdleSummary {
		for i, line := range lines {
			if line != "" {
				lines[i] = m.coachTipStyle.Render(line)
			}
		}
	}
	if len(lines) > availableHeight {
		lines = lines[:availableHeight]
	}

	// Vertically center all content in the box
	if len(lines) < availableHeight {
		// Calculate padding for vertical centering
		topPadding := (availableHeight - len(lines)) / 2
		bottomPadding := availableHeight - len(lines) - topPadding

		// Add empty lines at top
		centeredLines := make([]string, 0, availableHeight)
		for i := 0; i < topPadding; i++ {
			centeredLines = append(centeredLines, "")
		}
		centeredLines = append(centeredLines, lines...)
		// Add empty lines at bottom
		for i := 0; i < bottomPadding; i++ {
			centeredLines = append(centeredLines, "")
		}
		lines = centeredLines
	}

	// Top Border Logic
	// ╭[Badge][risk]──[context]╮
	topLeft := m.borderStatus.RenderTopLeft()
	// Calculate available space for top context
	// width - 2 (corners) - len(topLeft) - 2 (padding maybe?)

	// We construct top border in pieces
	// corner + topLeft + separator + context + ... + corner
	// Actually typical lipgloss border is uniform. We need to override.

	// We manually draw the top line.
	// "╭" + [Badge][Risk] + "──" + [Context] + "──" + "╮"

	// Let's compute exact widths.
	// Use TopLeftWidth() method which accounts for terminal-specific rendering
	// of emoji characters like 🤖, rather than lipgloss.Width() which may be incorrect
	topLeftWidth := m.borderStatus.TopLeftWidth()

	// Available width for middle
	middleWidth := innerWidth

	// If middleWidth is small, we might have issues.
	if middleWidth <= 0 {
		middleWidth = 0
	}

	topContentWidth := middleWidth
	// We need some lines between topLeft and Context?
	// Spec says: "Command kind badge immediately followed by the execution risk meter."
	// "Top edge: Prompt-style context stripes ... separated by line-continuation characters"
	// So: ╭[Badge][Risk]────[Context]────╮

	// Context
	// We want to fill the remaining space with context, right aligned or distributed?
	// Spec says: "Top edge (left-to-right): Prompt-style context stripes"
	// But Top-left is Badge/Risk.
	// So Badge/Risk comes first, then Context.
	// Should we pad with lines between them?

	// If we just concatenate: Badge Risk Context
	// And pad the rest with lines?
	// Or: Badge Risk ── Context ── ?

	// Let's assume: Badge Risk [context] ──────────
	// Or: Badge Risk ── [context] ── ?

	// Let's try to put context immediately after, separated by line.
	// But context stripes are variable width.

	// Render context with available width
	// Available = topContentWidth - topLeftWidth
	contextAvailableWidth := topContentWidth - topLeftWidth - 1 // -1 for separator
	if contextAvailableWidth < 0 {
		contextAvailableWidth = 0
	}

	topContext := m.borderStatus.RenderTopContext(contextAvailableWidth)
	topContextWidth := lipgloss.Width(topContext)

	// Line filler
	fillerWidth := topContentWidth - topLeftWidth - topContextWidth
	if fillerWidth < 0 {
		fillerWidth = 0
	}

	// Construction
	// ╭ + topLeft + [filler/separator] + topContext + [filler] + ╮
	// We prefer context to be visible.
	// The prompt context stripes usually sit on the line.

	// Design choice:
	// ╭[Badge][Risk]──[Context]──────────────╮

	// Note: We need to use border style for the line parts (╭, ─, ╮)
	// But Badge/Risk/Context have their own colors.

	var topBar strings.Builder
	topBar.WriteString(borderStyle.Render("╭"))
	topBar.WriteString(topLeft)

	if topContext != "" {
		// Separator line
		// Use Divider style from borderStatus? Or just border color?
		// Spec: "separated by line-continuation characters ... degrade to icon-only"
		// "Apply subtle color to the divider"
		// borderStatus handles internal dividers in Context.
		// Here we need divider between Risk and Context.
		topBar.WriteString(m.borderStatus.styles.Divider.Render("─"))
		topBar.WriteString(topContext)

		// Remaining filler
		if fillerWidth > 1 {
			topBar.WriteString(borderStyle.Render(strings.Repeat("─", fillerWidth-1)))
		}
	} else {
		// Just fill
		if fillerWidth > 0 {
			topBar.WriteString(borderStyle.Render(strings.Repeat("─", fillerWidth)))
		}
	}
	topBar.WriteString(borderStyle.Render("╮"))

	var result strings.Builder
	result.WriteString(topBar.String())
	result.WriteString("\n")

	// Content lines with left/right borders
	// Middle content - with one space padding on each side
	// Content is already wrapped at contentWidth
	for _, line := range lines {
		// Truncate or pad line to fit content width
		// Use stringWidthWithAnsi instead of lipgloss.Width to properly handle emoji
		lineWidth := stringWidthWithAnsi(line)
		if lineWidth > contentWidth {
			line = truncateWithAnsi(line, contentWidth)
			lineWidth = stringWidthWithAnsi(line)
		}
		padding := max(0, contentWidth-lineWidth)
		result.WriteString(borderStyle.Render("│"))
		result.WriteString(" ") // Left padding
		if isCoachTip && !isIdleSummary {
			// Right-justify coach tips (but not idle summaries)
			result.WriteString(strings.Repeat(" ", padding))
			result.WriteString(line)
		} else {
			// Left-justify other content including idle summaries
			result.WriteString(line)
			result.WriteString(strings.Repeat(" ", padding))
		}
		result.WriteString(" ") // Right padding
		result.WriteString(borderStyle.Render("│"))
		result.WriteString("\n")
	}

	// Bottom border with indicators: ╰[Res]──[User@Host]──[Res]── Fast:✓ Slow:○ ╯
	// Bottom-left: Resource Glance
	// Bottom-center: User@Host (centered) - suppressed if window too narrow
	// Bottom-right: LLM Indicator (preserved)

	bottomLeft := m.borderStatus.RenderBottomLeft()
	bottomCenter := m.borderStatus.RenderBottomCenter()
	bottomCenterWidth := lipgloss.Width(bottomCenter)
	bottomLeftWidth := lipgloss.Width(bottomLeft)

	indicatorStr := " " + m.llmIndicator.View() + " "
	// Use the indicator's Width() method which accounts for terminal-specific rendering
	// of the lightning bolt character, rather than lipgloss.Width() which may be incorrect
	indicatorLen := 2 + m.llmIndicator.Width() // 2 spaces + lightning bolt width

	// Calculate minimum required space for all elements
	minRequiredWidth := bottomLeftWidth + indicatorLen + 10 // 10 chars minimum for spacing

	// Use middleWidth to match the top bar's content width
	bottomContentWidth := middleWidth

	// Determine if we have enough space for user@hostname
	showUserHost := bottomContentWidth > minRequiredWidth && bottomCenter != ""

	// Calculate available space for centering
	var leftFillerWidth, rightFillerWidth, availableSpace int
	var totalUsedWidth int

	if showUserHost {
		totalUsedWidth = bottomLeftWidth + bottomCenterWidth + indicatorLen
		availableSpace = bottomContentWidth - totalUsedWidth

		if availableSpace < 0 {
			// Not enough space even with user@host, drop it
			showUserHost = false
			totalUsedWidth = bottomLeftWidth + indicatorLen
			availableSpace = bottomContentWidth - totalUsedWidth
		}

		// Distribute extra space to center the user@host
		leftFillerWidth = availableSpace / 2
		rightFillerWidth = availableSpace - leftFillerWidth
	} else {
		// User@host suppressed, just center between left and right
		totalUsedWidth = bottomLeftWidth + indicatorLen
		availableSpace = bottomContentWidth - totalUsedWidth

		leftFillerWidth = availableSpace / 2
		rightFillerWidth = availableSpace - leftFillerWidth
	}

	// Construction
	// ╰ + bottomLeft + leftFiller + center + rightFiller + indicator + ╯

	result.WriteString(borderStyle.Render("╰"))
	result.WriteString(bottomLeft)
	if showUserHost && leftFillerWidth > 0 {
		result.WriteString(borderStyle.Render(strings.Repeat("─", leftFillerWidth)))
	}
	if showUserHost && bottomCenter != "" {
		result.WriteString(bottomCenter)
	}
	if showUserHost && rightFillerWidth > 0 {
		result.WriteString(borderStyle.Render(strings.Repeat("─", rightFillerWidth)))
	}
	if !showUserHost && availableSpace > 0 {
		// User@host suppressed, just fill the space
		result.WriteString(borderStyle.Render(strings.Repeat("─", availableSpace)))
	}
	result.WriteString(indicatorStr)
	result.WriteString(borderStyle.Render("╯"))

	return inputStr + "\n" + result.String()
}

// stringWidthWithAnsi calculates the display width of a string, handling ANSI escape codes
// Uses terminal-specific probing for emoji characters to get accurate widths
func stringWidthWithAnsi(s string) int {
	width := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			continue
		}
		if inEscape {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		width += GetRuneWidth(r)
	}

	return width
}

// truncateWithAnsi truncates a string to maxWidth display columns, handling ANSI escape codes
// Uses terminal-specific probing for emoji characters to get accurate widths
func truncateWithAnsi(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	var result strings.Builder
	width := 0
	inEscape := false

	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			result.WriteRune(r)
			continue
		}
		if inEscape {
			result.WriteRune(r)
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEscape = false
			}
			continue
		}

		// Check if adding this rune would exceed maxWidth
		runeWidth := GetRuneWidth(r)
		if width+runeWidth > maxWidth {
			break
		}
		result.WriteRune(r)
		width += runeWidth
	}

	return result.String()
}

func (m appModel) getFinalOutput() string {
	m.textInput.SetValue(m.result)
	m.textInput.SetSuggestions([]string{})
	m.textInput.Blur()
	m.textInput.ShowSuggestions = false

	// Reset to original prompt for final output display
	m.textInput.Prompt = m.originalPrompt

	s := m.textInput.View()
	return s
}
