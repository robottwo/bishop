/*
This file is forked from the textinput component from
github.com/charmbracelet/bubbles

# MIT License

# Copyright (c) 2020-2023 Charmbracelet, Inc

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
package shellinput

import (
	"time"
	"unicode"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/cursor"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/runeutil"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ============================================================================
// Internal Messages
// ============================================================================

// Internal messages for clipboard operations.
type (
	pasteMsg    string
	pasteErrMsg struct{ error }
)

// ============================================================================
// Types and Configuration
// ============================================================================

// EchoMode sets the input behavior of the text input field.
type EchoMode int

const (
	// EchoNormal displays text as is. This is the default behavior.
	EchoNormal EchoMode = iota

	// EchoPassword displays the EchoCharacter mask instead of actual
	// characters. This is commonly used for password fields.
	EchoPassword

	// EchoNone displays nothing as characters are entered. This is commonly
	// seen for password fields on the command line.
	EchoNone
)

// ValidateFunc is a function that returns an error if the input is invalid.
type ValidateFunc func(string) error

// KeyMap is the key bindings for different actions within the textinput.
type KeyMap struct {
	CharacterForward        key.Binding
	CharacterBackward       key.Binding
	WordForward             key.Binding
	WordBackward            key.Binding
	DeleteWordBackward      key.Binding
	DeleteWordForward       key.Binding
	DeleteAfterCursor       key.Binding
	DeleteBeforeCursor      key.Binding
	DeleteCharacterBackward key.Binding
	DeleteCharacterForward  key.Binding
	LineStart               key.Binding
	LineEnd                 key.Binding
	Paste                   key.Binding
	Yank                    key.Binding
	YankPop                 key.Binding
	NextValue               key.Binding
	PrevValue               key.Binding
	Complete                key.Binding
	PrevSuggestion          key.Binding
	ClearScreen             key.Binding
	ReverseSearch           key.Binding
	HistorySort             key.Binding
	SwapCharacters          key.Binding
	SwapWords               key.Binding
	InsertLastArg           key.Binding
}

// DefaultKeyMap is the default set of key bindings for navigating and acting
// upon the textinput.
var DefaultKeyMap = KeyMap{
	CharacterForward:        key.NewBinding(key.WithKeys("right", "ctrl+f")),
	CharacterBackward:       key.NewBinding(key.WithKeys("left", "ctrl+b")),
	WordForward:             key.NewBinding(key.WithKeys("alt+right", "ctrl+right", "alt+f")),
	WordBackward:            key.NewBinding(key.WithKeys("alt+left", "ctrl+left", "alt+b")),
	DeleteWordBackward:      key.NewBinding(key.WithKeys("alt+backspace", "ctrl+w")),
	DeleteWordForward:       key.NewBinding(key.WithKeys("alt+delete", "alt+d")),
	DeleteAfterCursor:       key.NewBinding(key.WithKeys("ctrl+k")),
	DeleteBeforeCursor:      key.NewBinding(key.WithKeys("ctrl+u")),
	DeleteCharacterBackward: key.NewBinding(key.WithKeys("backspace", "ctrl+h")),
	Complete:                key.NewBinding(key.WithKeys("tab")),
	PrevSuggestion:          key.NewBinding(key.WithKeys("shift+tab")),
	DeleteCharacterForward:  key.NewBinding(key.WithKeys("delete", "ctrl+d")),
	LineStart:               key.NewBinding(key.WithKeys("home", "ctrl+a")),
	LineEnd:                 key.NewBinding(key.WithKeys("end", "ctrl+e")),
	Paste:                   key.NewBinding(key.WithKeys("ctrl+v")),
	Yank:                    key.NewBinding(key.WithKeys("ctrl+y")),
	YankPop:                 key.NewBinding(key.WithKeys("alt+y")),
	NextValue:               key.NewBinding(key.WithKeys("down", "ctrl+n")),
	PrevValue:               key.NewBinding(key.WithKeys("up", "ctrl+p")),
	ClearScreen:             key.NewBinding(key.WithKeys("ctrl+l")),
	ReverseSearch:           key.NewBinding(key.WithKeys("ctrl+r")),
	HistorySort:             key.NewBinding(key.WithKeys("ctrl+o")),
	SwapCharacters:          key.NewBinding(key.WithKeys("ctrl+t")),
	SwapWords:               key.NewBinding(key.WithKeys("alt+t")),
	InsertLastArg:           key.NewBinding(key.WithKeys("alt+.")),
}

// ============================================================================
// Model
// ============================================================================

// Model is the Bubble Tea model for this text input element.
type Model struct {
	Err error

	// General settings.
	Prompt        string
	EchoMode      EchoMode
	EchoCharacter rune
	Cursor        cursor.Model

	// Completion settings
	CompletionProvider CompletionProvider
	completion         completionState

	// Deprecated: use [cursor.BlinkSpeed] instead.
	BlinkSpeed time.Duration

	// Styles. These will be applied as inline styles.
	//
	// For an introduction to styling with Lip Gloss see:
	// https://github.com/charmbracelet/lipgloss
	PromptStyle              lipgloss.Style
	TextStyle                lipgloss.Style
	CompletionStyle          lipgloss.Style
	ReverseSearchPromptStyle lipgloss.Style

	// Deprecated: use Cursor.Style instead.
	CursorStyle lipgloss.Style

	// CharLimit is the maximum amount of characters this input element will
	// accept. If 0 or less, there's no limit.
	CharLimit int

	// Width marks the horizontal boundary for this component to render within.
	// Content that exceeds this width will be wrapped.
	// If 0 or less this setting is ignored.
	Width int

	// KeyMap encodes the keybindings recognized by the widget.
	KeyMap KeyMap

	// focus indicates whether user input focus should be on this input
	// component. When false, ignore keyboard input and hide the cursor.
	focus bool

	// Cursor position.
	pos int

	// killRing manages Emacs-style kill/yank operations
	killRing KillRing

	// State for Alt+. (Insert Last Argument)
	lastArgInsertionIndex   int
	lastCommandWasInsertArg bool
	lastInsertedArgLen      int

	// Validate is a function that checks whether or not the text within the
	// input is valid. If it is not valid, the `Err` field will be set to the
	// error returned by the function. If the function is not defined, all
	// input is considered valid.
	Validate ValidateFunc

	// rune sanitizer for input.
	rsan runeutil.Sanitizer

	// ShowSuggestions enables autocomplete suggestions. When true, the input
	// will display suggestions that match the current input value. This works
	// in conjunction with SetSuggestions() to provide tab-completion functionality.
	// The suggestion display can be customized via CompletionStyle.
	ShowSuggestions bool

	// suppressSuggestionsUntilInput temporarily disables autocomplete hints
	// until the user enters more text. This is used, for example, when the
	// user trims the line with Ctrl+K so that ghost text and help reflect
	// the truncated command until new input arrives. This flag is cleared
	// when the user types any character, restoring normal suggestion behavior.
	suppressSuggestionsUntilInput bool

	// Suggestion state fields (managed by methods in suggestions.go)
	//
	// The suggestion system provides tab-completion functionality through a
	// three-stage process:
	// 1. suggestions: Full list of available completions set via SetSuggestions()
	// 2. matchedSuggestions: Filtered subset matching current input (case-insensitive prefix)
	// 3. currentSuggestionIndex: Currently selected match when cycling with Tab/Shift+Tab
	//
	// Key methods (see suggestions.go):
	//   - SetSuggestions([]string): Sets available suggestions and triggers matching
	//   - updateSuggestions(): Refreshes matched list based on current input
	//   - AvailableSuggestions(): Returns all available suggestions
	//   - MatchedSuggestions(): Returns suggestions matching current input
	//   - CurrentSuggestion(): Returns the currently selected suggestion
	//   - CurrentSuggestionIndex(): Returns the selection index
	//   - canAcceptSuggestion(): Checks if a suggestion is available to accept

	// suggestions stores the full list of available suggestions that may be
	// used to autocomplete the input. Set via SetSuggestions() method.
	// Each suggestion is stored as a []rune for efficient text manipulation.
	suggestions [][]rune

	// matchedSuggestions stores the filtered list of suggestions that match
	// the current input value using case-insensitive prefix matching.
	// This list is automatically updated by updateSuggestions() whenever
	// the input value changes. Only suggestions from this list can be
	// cycled through and accepted by the user.
	matchedSuggestions [][]rune

	// currentSuggestionIndex tracks which matched suggestion is currently
	// selected when cycling through suggestions with Tab/Shift+Tab keys.
	// The index is reset to 0 when the matched suggestions list changes.
	// Valid range: [0, len(matchedSuggestions)-1]
	currentSuggestionIndex int

	// values[0] is the current value. other indices represent history values
	// that can be navigated with the up and down arrow keys.
	values             [][]rune
	selectedValueIndex int

	// Reverse search state
	inReverseSearch    bool
	reverseSearchQuery string

	// Rich history search
	historyItems       []HistoryItem
	historySearchState historySearchState
}

// ============================================================================
// Constructor
// ============================================================================

// New creates a new model with default settings.
func New() Model {
	return Model{
		Prompt:                   "> ",
		EchoCharacter:            '*',
		CharLimit:                0,
		ShowSuggestions:          false,
		CompletionStyle:          lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		ReverseSearchPromptStyle: lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
		Cursor:                   cursor.New(),
		KeyMap:                   DefaultKeyMap,

		suggestions: [][]rune{},
		focus:       false,
		pos:         0,

		values:             [][]rune{{}},
		selectedValueIndex: 0,
	}
}

// ============================================================================
// Value Management
// ============================================================================

// SetValue sets the value of the text input.
func (m *Model) SetValue(s string) {
	// Clean up any special characters in the input provided by the
	// caller. This avoids bugs due to e.g. tab characters and whatnot.
	runes := m.san().Sanitize([]rune(s))
	err := m.validate(runes)
	m.setValueInternal(runes, err)
}

func (m *Model) setValueInternal(runes []rune, err error) {
	m.Err = err
	m.killRing.lastWasKill = false
	m.killRing.yankActive = false

	empty := len(m.values[m.selectedValueIndex]) == 0

	if m.CharLimit > 0 && len(runes) > m.CharLimit {
		m.values[0] = runes[:m.CharLimit]
	} else {
		m.values[0] = runes
	}
	m.selectedValueIndex = 0
	if (m.pos == 0 && empty) || m.pos > len(m.values[0]) {
		m.SetCursor(len(m.values[0]))
	}
}

// Value returns the value of the text input.
func (m Model) Value() string {
	return string(m.values[m.selectedValueIndex])
}

// InReverseSearch returns true if the input is currently in reverse search mode.
func (m Model) InReverseSearch() bool {
	return m.inReverseSearch
}

// HistorySearchBoxView returns the rendered history search box if active.
// Note: This is a wrapper to allow the method to be called from the interface/package level if needed,
// but the actual implementation is in history_search.go.
// Go allows methods to be in different files of the same package.

// Position returns the cursor position.
func (m Model) Position() int {
	return m.pos
}

// ============================================================================
// Cursor and Position Management
// ============================================================================

// SetCursor moves the cursor to the given position. If the position is
// out of bounds the cursor will be moved to the start or end accordingly.
func (m *Model) SetCursor(pos int) {
	m.pos = clamp(pos, 0, len(m.values[m.selectedValueIndex]))
}

// getCursorPosition returns the current cursor position.
// This method implements the bufferEditor interface for KillRing operations.
func (m *Model) getCursorPosition() int {
	return m.pos
}

// setValue sets the input buffer value to the primary value slot.
// This method implements the bufferEditor interface for KillRing operations.
func (m *Model) setValue(value []rune) {
	m.values[0] = value
	m.selectedValueIndex = 0
}

// getValue returns the current input buffer value.
// This method implements the bufferEditor interface for KillRing operations.
func (m *Model) getValue() []rune {
	return m.values[m.selectedValueIndex]
}

// setCursor sets the cursor position.
// This method implements the bufferEditor interface for KillRing operations.
func (m *Model) setCursor(pos int) {
	m.SetCursor(pos)
}

// setError sets the validation error.
// This method implements the bufferEditor interface for KillRing operations.
func (m *Model) setError(err error) {
	m.Err = err
}

// suppressSuggestions suppresses suggestions until next input.
// This method implements the bufferEditor interface for KillRing operations.
func (m *Model) suppressSuggestions() {
	m.suppressSuggestionsUntilInput = true
}

// clearSuggestions clears matched suggestions.
// This method implements the bufferEditor interface for KillRing operations.
func (m *Model) clearSuggestions() {
	m.matchedSuggestions = [][]rune{}
	m.currentSuggestionIndex = 0
}

// CursorStart moves the cursor to the start of the input field.
func (m *Model) CursorStart() {
	m.SetCursor(0)
}

// CursorEnd moves the cursor to the end of the input field.
func (m *Model) CursorEnd() {
	m.SetCursor(len(m.values[m.selectedValueIndex]))
}

// ============================================================================
// Focus and Blur
// ============================================================================

// Focused returns the focus state on the model.
func (m Model) Focused() bool {
	return m.focus
}

// Focus sets the focus state on the model. When the model is in focus it can
// receive keyboard input and the cursor will be shown.
func (m *Model) Focus() tea.Cmd {
	m.focus = true
	return m.Cursor.Focus()
}

// Blur removes the focus state on the model.  When the model is blurred it can
// not receive keyboard input and the cursor will be hidden.
func (m *Model) Blur() {
	m.focus = false
	m.Cursor.Blur()
}

// ============================================================================
// State Management
// ============================================================================

// Reset sets the input to its default state with no input.
func (m *Model) Reset() {
	m.values = [][]rune{{}}
	m.selectedValueIndex = 0
	m.SetCursor(0)
}

// SetHistoryValues sets the suggestions for the input.
func (m *Model) SetHistoryValues(historyValues []string) {
	m.values = append([][]rune{m.values[0]}, make([][]rune, len(historyValues))...)

	for i, s := range historyValues {
		m.values[i+1] = m.san().Sanitize([]rune(s))
	}

	// reset value index if the selected index is out of bounds
	if m.selectedValueIndex >= len(m.values) {
		m.selectedValueIndex = 0
	}
}

// rsan initializes or retrieves the rune sanitizer.
func (m *Model) san() runeutil.Sanitizer {
	if m.rsan == nil {
		// Textinput has all its input on a single line so collapse
		// newlines/tabs to single spaces.
		m.rsan = runeutil.NewSanitizer(
			runeutil.ReplaceTabs(" "), runeutil.ReplaceNewlines(" "))
	}
	return m.rsan
}

// ============================================================================
// Input Handling
// ============================================================================

func (m *Model) insertRunesFromUserInput(v []rune) {
	m.suppressSuggestionsUntilInput = false
	m.killRing.lastWasKill = false
	m.killRing.yankActive = false
	// Only reset lastCommandWasInsertArg if it wasn't set by insertLastArg just before calling this
	// But insertLastArg calls this, so it will be reset.
	// So insertLastArg must re-set it to true.
	// However, if the user types a normal character, it calls this, and we WANT to reset it.
	// So I can't distinguish here easily.
	// Solution: insertRunesFromUserInput always resets it. insertLastArg re-sets it after calling this.
	m.lastCommandWasInsertArg = false

	// Clean up any special characters in the input provided by the
	// clipboard. This avoids bugs due to e.g. tab characters and
	// whatnot.
	paste := m.san().Sanitize(v)

	var availSpace int
	if m.CharLimit > 0 {
		availSpace = m.CharLimit - len(m.values[m.selectedValueIndex])

		// If the char limit's been reached, cancel.
		if availSpace <= 0 {
			return
		}

		// If there's not enough space to paste the whole thing cut the pasted
		// runes down so they'll fit.
		if availSpace < len(paste) {
			paste = paste[:availSpace]
		}
	}

	result := make([]rune, len(m.values[m.selectedValueIndex])+len(paste))

	copy(result, m.values[m.selectedValueIndex][:m.pos])
	copy(result[m.pos:], paste)
	copy(result[m.pos+len(paste):], m.values[m.selectedValueIndex][m.pos:])
	m.pos += len(paste)

	inputErr := m.validate(result)
	m.setValueInternal(result, inputErr)
}

// ============================================================================
// Kill Ring Integration
// ============================================================================

// recordKill captures killed text for yank operations and temporarily suppresses
// autocomplete hints until the user provides new input.
func (m *Model) recordKill(killed []rune, direction killDirection) {
	m.killRing.RecordKill(m, killed, direction)
}

// yankKillBuffer pastes the most recently killed text at the cursor position.
func (m *Model) yankKillBuffer() {
	m.killRing.YankKillBuffer(m)
}

// yankPop cycles through the kill ring after a yank, replacing the previously
// yanked text with the next entry.
func (m *Model) yankPop() {
	m.killRing.YankPop(m)
}

// ============================================================================
// Update Loop
// ============================================================================

// Update is the Bubble Tea update loop.
func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.focus {
		return m, nil
	}

	// Let's remember where the position of the cursor currently is so that if
	// the cursor position changes, we can reset the blink.
	oldPos := m.pos

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Reset lastCommandWasInsertArg unless InsertLastArg was pressed
		if !key.Matches(msg, m.KeyMap.InsertLastArg) {
			m.lastCommandWasInsertArg = false
		}

		// Handle reverse search specific keys
		if m.inReverseSearch {
			switch {
			case key.Matches(msg, m.KeyMap.ReverseSearch):
				// Toggle or exit? Standard Bash Ctrl+R cycles if there are matches,
				// but here we have a list. We can just keep focus or toggle off?
				// For now let's say Ctrl+R again toggles off or does nothing special if we just show a list.
				// Or maybe it cycles selection?
				// Let's make it toggle off for now, or maybe act as "Next" if we want.
				// The requirement says "Typing filters... Selection inserts...".
				// Let's allow Ctrl+R to exit if pressed again, or maybe cycle filters later.
				// For now: cancel.
				m.cancelReverseSearch()
				return m, nil
			case key.Matches(msg, m.KeyMap.PrevValue): // Up
				m.historySearchUp()
				return m, nil
			case key.Matches(msg, m.KeyMap.NextValue): // Down
				m.historySearchDown()
				return m, nil
			// Toggle Filter with Ctrl+F
			case msg.String() == "ctrl+f":
				m.toggleHistoryFilter()
				return m, nil
			// Toggle Sort with Ctrl+O
			case key.Matches(msg, m.KeyMap.HistorySort):
				m.toggleHistorySort()
				return m, nil
			// Left/Right: Accept and edit?
			case key.Matches(msg, m.KeyMap.CharacterBackward), key.Matches(msg, m.KeyMap.CharacterForward):
				m.acceptRichReverseSearch()
				return m, nil
			case msg.String() == "enter":
				m.acceptRichReverseSearch()
				return m, nil
			case msg.String() == "ctrl+g" || msg.String() == "ctrl+c" || msg.String() == "escape" || msg.String() == "esc":
				m.cancelReverseSearch()
				return m, nil
			case key.Matches(msg, m.KeyMap.DeleteCharacterBackward):
				if len(m.reverseSearchQuery) > 0 {
					runes := []rune(m.reverseSearchQuery)
					m.reverseSearchQuery = string(runes[:len(runes)-1])
					m.updateHistorySearch()
				}
				return m, nil
			case len(msg.Runes) > 0 && unicode.IsPrint(msg.Runes[0]):
				m.reverseSearchQuery += string(msg.Runes)
				m.updateHistorySearch()
				return m, nil
			default:
				// Ignore other keys in reverse search mode
				return m, nil
			}
		}

		// Handle completion-specific keys first
		if m.completion.active {
			switch msg.String() {
			case "escape":
				m.cancelCompletion()
				return m, nil
			case "enter":
				if m.completion.shouldShowInfoBox() && m.completion.selected >= 0 {
					// Accept the currently selected completion
					suggestion := m.completion.currentSuggestion()
					if suggestion != "" {
						m.applySuggestion(suggestion)
					}
					m.resetCompletion()
					return m, nil
				}
			}
		}

		// Reset completion state for any key except TAB, Shift+TAB, Escape, and Enter
		if !key.Matches(msg, m.KeyMap.Complete) && !key.Matches(msg, m.KeyMap.PrevSuggestion) &&
			msg.String() != "escape" && msg.String() != "enter" {
			m.resetCompletion()
		}

		killCommand := key.Matches(msg, m.KeyMap.DeleteBeforeCursor) || key.Matches(msg, m.KeyMap.DeleteAfterCursor) ||
			key.Matches(msg, m.KeyMap.DeleteWordBackward) || key.Matches(msg, m.KeyMap.DeleteWordForward)
		yankCommand := key.Matches(msg, m.KeyMap.Yank) || key.Matches(msg, m.KeyMap.YankPop)

		if m.suppressSuggestionsUntilInput && !killCommand {
			m.suppressSuggestionsUntilInput = false
		}

		switch {
		case key.Matches(msg, m.KeyMap.ReverseSearch):
			m.toggleReverseSearch()
			return m, nil
		case key.Matches(msg, m.KeyMap.Complete):
			m.handleCompletion()
			return m, nil
		case key.Matches(msg, m.KeyMap.PrevSuggestion) && m.completion.active:
			m.handleBackwardCompletion()
			return m, nil
		case key.Matches(msg, m.KeyMap.SwapCharacters):
			m.swapCharacters()
		case key.Matches(msg, m.KeyMap.SwapWords):
			m.swapWords()
		case key.Matches(msg, m.KeyMap.InsertLastArg):
			m.insertLastArg()
		case key.Matches(msg, m.KeyMap.DeleteWordBackward):
			m.deleteWordBackward()
		case key.Matches(msg, m.KeyMap.DeleteCharacterBackward):
			m.Err = nil
			if len(m.values[m.selectedValueIndex]) > 0 {
				newValue := cloneConcatRunes(m.values[m.selectedValueIndex][:max(0, m.pos-1)], m.values[m.selectedValueIndex][m.pos:])
				m.Err = m.validate(newValue)
				m.values[0] = newValue
				m.selectedValueIndex = 0
				if m.pos > 0 {
					m.SetCursor(m.pos - 1)
				}
			}
		case key.Matches(msg, m.KeyMap.WordBackward):
			m.wordBackward()
		case key.Matches(msg, m.KeyMap.CharacterBackward):
			if m.pos > 0 {
				m.SetCursor(m.pos - 1)
			}
		case key.Matches(msg, m.KeyMap.WordForward):
			m.wordForward()
		case key.Matches(msg, m.KeyMap.CharacterForward):
			if m.pos < len(m.values[m.selectedValueIndex]) {
				m.SetCursor(m.pos + 1)
			} else if m.canAcceptSuggestion() {
				newValue := cloneConcatRunes(
					m.values[m.selectedValueIndex],
					m.matchedSuggestions[m.currentSuggestionIndex][len(m.values[m.selectedValueIndex]):],
				)
				m.Err = m.validate(newValue)
				m.values[0] = newValue
				m.selectedValueIndex = 0
				m.CursorEnd()
			}
		case key.Matches(msg, m.KeyMap.LineStart):
			m.CursorStart()
		case key.Matches(msg, m.KeyMap.DeleteCharacterForward):
			if len(m.values[m.selectedValueIndex]) > 0 && m.pos < len(m.values[m.selectedValueIndex]) {
				newValue := cloneConcatRunes(m.values[m.selectedValueIndex][:m.pos], m.values[m.selectedValueIndex][m.pos+1:])
				m.Err = m.validate(newValue)
				m.values[0] = newValue
				m.selectedValueIndex = 0
			}
		case key.Matches(msg, m.KeyMap.LineEnd):
			m.CursorEnd()
		case key.Matches(msg, m.KeyMap.DeleteAfterCursor):
			m.deleteAfterCursor()
		case key.Matches(msg, m.KeyMap.DeleteBeforeCursor):
			m.deleteBeforeCursor()
		case key.Matches(msg, m.KeyMap.Paste):
			return m, Paste
		case key.Matches(msg, m.KeyMap.Yank):
			m.yankKillBuffer()
		case key.Matches(msg, m.KeyMap.YankPop):
			m.yankPop()
		case key.Matches(msg, m.KeyMap.DeleteWordForward):
			m.deleteWordForward()
		case key.Matches(msg, m.KeyMap.NextValue):
			m.nextValue()
		case key.Matches(msg, m.KeyMap.PrevValue):
			m.previousValue()
		case key.Matches(msg, m.KeyMap.ClearScreen):
			// Clear screen functionality will be handled by the gline package
			// Return the model unchanged to prevent default character input
			// The gline package will handle the actual screen clearing
			return m, nil
		default:
			// Input one or more regular characters.
			m.insertRunesFromUserInput(msg.Runes)
		}

		if !killCommand && !yankCommand {
			m.killRing.lastWasKill = false
		}

		if !yankCommand {
			m.killRing.yankActive = false
		}

		// Check again if can be completed
		// because value might be something that does not match the completion prefix
		m.updateSuggestions()

		// Update help info for special commands
		m.updateHelpInfo()

	case pasteMsg:
		m.insertRunesFromUserInput([]rune(msg))

	case pasteErrMsg:
		m.Err = msg
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.Cursor, cmd = m.Cursor.Update(msg)
	cmds = append(cmds, cmd)

	if oldPos != m.pos && m.Cursor.Mode() == cursor.CursorBlink {
		m.Cursor.Blink = false
		cmds = append(cmds, m.Cursor.BlinkCmd())
	}

	return m, tea.Batch(cmds...)
}

// ============================================================================
// Commands
// ============================================================================

// Blink is a command used to initialize cursor blinking.
func Blink() tea.Msg {
	return cursor.Blink()
}

// Paste is a command for pasting from the clipboard into the text input.
func Paste() tea.Msg {
	str, err := clipboard.ReadAll()
	if err != nil {
		return pasteErrMsg{err}
	}
	return pasteMsg(str)
}

// ============================================================================
// Deprecated Types and Methods
// ============================================================================

// Deprecated: use cursor.Mode.
type CursorMode int

const (
	// Deprecated: use cursor.CursorBlink.
	CursorBlink = CursorMode(cursor.CursorBlink)
	// Deprecated: use cursor.CursorStatic.
	CursorStatic = CursorMode(cursor.CursorStatic)
	// Deprecated: use cursor.CursorHide.
	CursorHide = CursorMode(cursor.CursorHide)
)

func (c CursorMode) String() string {
	return cursor.Mode(c).String()
}

// Deprecated: use cursor.Mode().
func (m Model) CursorMode() CursorMode {
	return CursorMode(m.Cursor.Mode())
}

// Deprecated: use cursor.SetMode().
func (m *Model) SetCursorMode(mode CursorMode) tea.Cmd {
	return m.Cursor.SetMode(cursor.Mode(mode))
}

// ============================================================================
// Helper Methods
// ============================================================================

// SuggestionsSuppressedUntilInput reports whether autocomplete hints are
// temporarily disabled until the user provides additional input (for example
// after a kill command like Ctrl+K).
func (m Model) SuggestionsSuppressedUntilInput() bool {
	return m.suppressSuggestionsUntilInput
}

func (m *Model) nextValue() {
	if len(m.values) == 1 {
		return
	}

	m.selectedValueIndex--
	if m.selectedValueIndex < 0 {
		m.selectedValueIndex = 0
	}
	m.SetCursor(len(m.values[m.selectedValueIndex]))
}

func (m *Model) previousValue() {
	if len(m.values) == 1 {
		return
	}

	m.selectedValueIndex++
	if m.selectedValueIndex >= len(m.values) {
		m.selectedValueIndex = len(m.values) - 1
	}
	m.SetCursor(len(m.values[m.selectedValueIndex]))
}

// validate runs the Model's Validate function on the given rune slice.
// Returns an error if validation fails, or nil if no validator is set or validation passes.
func (m Model) validate(v []rune) error {
	if m.Validate != nil {
		return m.Validate(string(v))
	}
	return nil
}

// ============================================================================
// Reverse Search
// ============================================================================

// toggleReverseSearch toggles the reverse search mode.
func (m *Model) toggleReverseSearch() {
	if m.inReverseSearch {
		m.inReverseSearch = false
	} else {
		m.inReverseSearch = true
		m.reverseSearchQuery = ""
		m.updateHistorySearch()
	}
}

// acceptRichReverseSearch accepts the currently selected history item.
func (m *Model) acceptRichReverseSearch() {
	if len(m.historySearchState.filteredIndices) > 0 {
		idx := m.historySearchState.selected
		if idx >= 0 && idx < len(m.historySearchState.filteredIndices) {
			originalIdx := m.historySearchState.filteredIndices[idx]
			if originalIdx >= 0 && originalIdx < len(m.historyItems) {
				// Use SetValue to properly handle sanitation and cursor positioning
				m.SetValue(m.historyItems[originalIdx].Command)
				m.CursorEnd()
			}
		}
	}
	m.inReverseSearch = false
}

// cancelReverseSearch cancels the reverse search and restores the original state.
func (m *Model) cancelReverseSearch() {
	m.inReverseSearch = false
	// Optionally restore original input? Bash restores the line you were on before Ctrl+R.
	// Since we modify selectedValueIndex only on accept, just exiting works effectively like cancel if we were editing.
	// But if we want to cancel effectively, we should just switch off the mode.
}
