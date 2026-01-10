package shellinput

const (
	killRingMax = 30
)

// killDirection represents the direction of a kill operation
type killDirection int

const (
	killDirectionUnknown killDirection = iota
	killDirectionForward
	killDirectionBackward
)

// KillRing manages the Emacs-style kill ring for cut/paste operations.
// It stores recently killed (cut) text and supports yank (paste) and yank-pop
// (cycling through previous kills) operations.
type KillRing struct {
	// ring stores recently killed text. The head (index 0) is the most recent kill.
	ring [][]rune
	// index is used when cycling through the ring with yank-pop
	index int
	// lastDirection tracks the direction of the previous kill to support
	// Bash/zsh-style kill ring appending semantics
	lastDirection killDirection
	// lastWasKill tracks whether the last command was a kill operation
	lastWasKill bool
	// yankActive indicates whether a yank operation is currently active
	yankActive bool
	// yankStart is the start position of the last yank in the input buffer
	yankStart int
	// yankEnd is the end position of the last yank in the input buffer
	yankEnd int
}

// bufferEditor defines the interface for editing the input buffer.
// This allows the KillRing to perform yank operations without tightly
// coupling to the Model implementation.
type bufferEditor interface {
	// insertRunesFromUserInput inserts runes at the current cursor position
	insertRunesFromUserInput(runes []rune)
	// getCursorPosition returns the current cursor position
	getCursorPosition() int
	// setValue sets the input buffer value
	setValue(value []rune)
	// getValue returns the current input buffer value
	getValue() []rune
	// setCursor sets the cursor position
	setCursor(pos int)
	// validate validates the input and returns an error if invalid
	validate(value []rune) error
	// setError sets the validation error
	setError(err error)
	// suppressSuggestions suppresses suggestions until next input
	suppressSuggestions()
	// clearSuggestions clears matched suggestions
	clearSuggestions()
	// resetCompletion resets completion state
	resetCompletion()
}

// RecordKill records killed text in the kill ring. If the last command was
// also a kill in the same direction, the text is appended to the most recent
// kill entry. Otherwise, a new entry is created.
func (kr *KillRing) RecordKill(editor bufferEditor, killed []rune, direction killDirection) {
	if len(killed) > 0 {
		cleaned := cloneRunes(killed)

		if kr.lastWasKill && direction == kr.lastDirection && len(kr.ring) > 0 {
			if direction == killDirectionForward {
				kr.ring[0] = append(kr.ring[0], cleaned...)
			} else {
				kr.ring[0] = append(cleaned, kr.ring[0]...)
			}
		} else {
			kr.ring = append([][]rune{cleaned}, kr.ring...)
			if len(kr.ring) > killRingMax {
				kr.ring = kr.ring[:killRingMax]
			}
			kr.index = 0
		}
		kr.lastWasKill = true
	} else {
		kr.lastWasKill = false
	}

	kr.lastDirection = direction
	kr.yankActive = false
	editor.suppressSuggestions()
	editor.clearSuggestions()
	editor.resetCompletion()
}

// YankKillBuffer pastes the most recently killed text at the cursor position.
func (kr *KillRing) YankKillBuffer(editor bufferEditor) {
	if len(kr.ring) == 0 {
		return
	}

	killed := cloneRunes(kr.ring[0])
	editor.insertRunesFromUserInput(killed)
	kr.yankStart = editor.getCursorPosition() - len(killed)
	kr.yankEnd = editor.getCursorPosition()
	kr.index = 0
	kr.yankActive = true
	kr.lastWasKill = false
}

// YankPop cycles through the kill ring after a yank, replacing the previously
// yanked text with the next entry.
func (kr *KillRing) YankPop(editor bufferEditor) {
	if !kr.yankActive || len(kr.ring) == 0 {
		return
	}

	if len(kr.ring) == 1 {
		return
	}

	kr.index = (kr.index + 1) % len(kr.ring)

	value := editor.getValue()
	start := clamp(kr.yankStart, 0, len(value))
	end := clamp(kr.yankEnd, start, len(value))

	replacement := cloneRunes(kr.ring[kr.index])
	newValue := make([]rune, 0, len(value)-end+start+len(replacement))
	newValue = append(newValue, value[:start]...)
	newValue = append(newValue, replacement...)
	newValue = append(newValue, value[end:]...)

	err := editor.validate(newValue)
	editor.setError(err)
	editor.setValue(newValue)
	editor.setCursor(start + len(replacement))

	kr.yankStart = start
	kr.yankEnd = start + len(replacement)
	kr.yankActive = true
	kr.lastWasKill = false
}
