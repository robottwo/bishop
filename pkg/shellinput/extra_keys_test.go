package shellinput

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func TestSwapCharacters(t *testing.T) {
	model := New()
	model.Focus()

	// Case 1: Middle
	// "ab|cde" (cursor at 2).
	// Before cursor: 'b'. At cursor: 'c'.
	// Result: "ac|bde". Cursor at 3.
	model.SetValue("abcde")
	model.SetCursor(2)
	msg := tea.KeyMsg{Type: tea.KeyCtrlT}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "acbde", updatedModel.Value(), "Middle swap failed")
	assert.Equal(t, 3, updatedModel.Position(), "Cursor should move forward")

	// Case 2: End
	// "abcde|" (cursor at 5).
	// Two before point: 'd', 'e'.
	// Result: "abced|". Cursor at 5.
	model.SetValue("abcde")
	model.SetCursor(5)
	msg = tea.KeyMsg{Type: tea.KeyCtrlT}
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "abced", updatedModel.Value(), "End swap failed")
	assert.Equal(t, 5, updatedModel.Position(), "Cursor should stay at end")

	// Case 3: Start (should do nothing)
	model.SetValue("abcde")
	model.SetCursor(0)
	msg = tea.KeyMsg{Type: tea.KeyCtrlT}
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "abcde", updatedModel.Value(), "Start swap should do nothing")

	// Case 4: Index 1 (swaps 0 and 1, moves to 2?)
	// "a|bcde" (cursor at 1).
	// Before: 'a'. At: 'b'.
	// Result: "ba|cde". Cursor at 2.
	model.SetValue("abcde")
	model.SetCursor(1)
	msg = tea.KeyMsg{Type: tea.KeyCtrlT}
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "bacde", updatedModel.Value(), "Swap at index 1 failed")
	assert.Equal(t, 2, updatedModel.Position(), "Cursor should move forward")
}

func TestSwapWords(t *testing.T) {
	model := New()
	model.Focus()

	// Case 1: End
	// "one two three|" -> "one three two|".
	model.SetValue("one two three")
	model.SetCursor(13) // End
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: true}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "one three two", updatedModel.Value(), "End word swap failed")

	// Case 2: Between words
	// "one |two three"
	// Word before: "one". Word after: "two".
	// Result: "two one| three".
	model.SetValue("one two three")
	model.SetCursor(4) // "one |two three"
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}, Alt: true}
	updatedModel, _ = model.Update(msg)
	assert.Equal(t, "two one three", updatedModel.Value(), "Middle word swap failed")
}

func TestInsertLastArg(t *testing.T) {
	model := New()
	model.Focus()

	// Set up history
	// m.values = [current, history1, history2]
	model.SetHistoryValues([]string{"echo one", "ls -la /tmp"})

	// "echo one" is index 1 (most recent). "ls -la /tmp" is index 2.

	// Test 1: First press inserts last arg of most recent ("one")
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'.'}, Alt: true}
	updatedModel, _ := model.Update(msg)
	assert.Equal(t, "one", updatedModel.Value(), "First Alt+. failed")

	// Test 2: Second press cycles to next history ("ls -la /tmp" -> "/tmp")
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "/tmp", updatedModel.Value(), "Cycling Alt+. failed")

	// Test 3: Third press cycles back to start (most recent) -> "one"
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "one", updatedModel.Value(), "Cycling back failed")

	// Test 4: Typing resets cycling
	// Type ' '
	msgSpace := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}}
	updatedModel, _ = updatedModel.Update(msgSpace)
	assert.Equal(t, "one ", updatedModel.Value())

	// Now InsertLastArg should start from fresh (index 1 -> "one")
	updatedModel, _ = updatedModel.Update(msg)
	assert.Equal(t, "one one", updatedModel.Value(), "Reset cycling failed")
}
