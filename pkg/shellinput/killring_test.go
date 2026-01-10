package shellinput

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBufferEditor implements bufferEditor for testing KillRing in isolation
type mockBufferEditor struct {
	value                []rune
	cursor               int
	err                  error
	suppressSuggCalled   bool
	clearSuggCalled      bool
	resetCompletionCalled bool
}

func (m *mockBufferEditor) insertRunesFromUserInput(runes []rune) {
	// Insert at cursor position
	newValue := make([]rune, 0, len(m.value)+len(runes))
	newValue = append(newValue, m.value[:m.cursor]...)
	newValue = append(newValue, runes...)
	newValue = append(newValue, m.value[m.cursor:]...)
	m.value = newValue
	m.cursor += len(runes)
}

func (m *mockBufferEditor) getCursorPosition() int {
	return m.cursor
}

func (m *mockBufferEditor) setValue(value []rune) {
	m.value = value
}

func (m *mockBufferEditor) getValue() []rune {
	return m.value
}

func (m *mockBufferEditor) setCursor(pos int) {
	m.cursor = pos
}

func (m *mockBufferEditor) validate(value []rune) error {
	return nil
}

func (m *mockBufferEditor) setError(err error) {
	m.err = err
}

func (m *mockBufferEditor) suppressSuggestions() {
	m.suppressSuggCalled = true
}

func (m *mockBufferEditor) clearSuggestions() {
	m.clearSuggCalled = true
}

func (m *mockBufferEditor) resetCompletion() {
	m.resetCompletionCalled = true
}

func TestKillRing_RecordKill_SingleKill(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{}

	killed := []rune("hello")
	kr.RecordKill(editor, killed, killDirectionForward)

	assert.Equal(t, 1, len(kr.ring), "Should have one entry in kill ring")
	assert.Equal(t, "hello", string(kr.ring[0]), "Kill ring should contain killed text")
	assert.True(t, kr.lastWasKill, "lastWasKill should be true")
	assert.Equal(t, killDirectionForward, kr.lastDirection, "lastDirection should be forward")
	assert.False(t, kr.yankActive, "yankActive should be false after kill")
	assert.True(t, editor.suppressSuggCalled, "Should suppress suggestions")
	assert.True(t, editor.clearSuggCalled, "Should clear suggestions")
	assert.True(t, editor.resetCompletionCalled, "Should reset completion")
}

func TestKillRing_RecordKill_AppendSameDirection(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{}

	// First kill
	kr.RecordKill(editor, []rune("hello"), killDirectionForward)
	require.Equal(t, 1, len(kr.ring))

	// Second kill in same direction should append
	kr.RecordKill(editor, []rune(" world"), killDirectionForward)

	assert.Equal(t, 1, len(kr.ring), "Should still have one entry")
	assert.Equal(t, "hello world", string(kr.ring[0]), "Should append to existing entry")
	assert.True(t, kr.lastWasKill, "lastWasKill should be true")
}

func TestKillRing_RecordKill_AppendBackward(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{}

	// First kill backward
	kr.RecordKill(editor, []rune("world"), killDirectionBackward)
	require.Equal(t, 1, len(kr.ring))

	// Second kill backward should prepend
	kr.RecordKill(editor, []rune("hello "), killDirectionBackward)

	assert.Equal(t, 1, len(kr.ring), "Should still have one entry")
	assert.Equal(t, "hello world", string(kr.ring[0]), "Should prepend to existing entry")
}

func TestKillRing_RecordKill_DifferentDirection(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{}

	// First kill forward
	kr.RecordKill(editor, []rune("hello"), killDirectionForward)
	require.Equal(t, 1, len(kr.ring))

	// Second kill backward should create new entry
	kr.RecordKill(editor, []rune("world"), killDirectionBackward)

	assert.Equal(t, 2, len(kr.ring), "Should have two entries")
	assert.Equal(t, "world", string(kr.ring[0]), "Most recent kill should be at index 0")
	assert.Equal(t, "hello", string(kr.ring[1]), "Previous kill should be at index 1")
}

func TestKillRing_RecordKill_EmptyKill(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{}

	// Record empty kill
	kr.RecordKill(editor, []rune{}, killDirectionForward)

	assert.Equal(t, 0, len(kr.ring), "Empty kill should not be recorded")
	assert.False(t, kr.lastWasKill, "lastWasKill should be false for empty kill")
	assert.Equal(t, killDirectionForward, kr.lastDirection, "Direction should still be set")
}

func TestKillRing_RecordKill_MaxSize(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{}

	// Record more than killRingMax entries
	for i := 0; i < killRingMax+5; i++ {
		kr.RecordKill(editor, []rune{rune('a' + i)}, killDirectionForward)
		// Force new entry by toggling lastWasKill
		kr.lastWasKill = false
	}

	assert.Equal(t, killRingMax, len(kr.ring), "Kill ring should not exceed max size")
}

func TestKillRing_YankKillBuffer_EmptyRing(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{
		value:  []rune("test"),
		cursor: 4,
	}

	kr.YankKillBuffer(editor)

	assert.Equal(t, "test", string(editor.value), "Value should be unchanged")
	assert.False(t, kr.yankActive, "yankActive should remain false")
}

func TestKillRing_YankKillBuffer_Basic(t *testing.T) {
	kr := &KillRing{
		ring: [][]rune{
			[]rune("yanked text"),
		},
	}
	editor := &mockBufferEditor{
		value:  []rune("before after"),
		cursor: 7, // Between "before " and "after"
	}

	kr.YankKillBuffer(editor)

	assert.Equal(t, "before yanked textafter", string(editor.value), "Should insert yanked text at cursor")
	assert.Equal(t, 18, editor.cursor, "Cursor should be after yanked text")
	assert.True(t, kr.yankActive, "yankActive should be true")
	assert.Equal(t, 7, kr.yankStart, "yankStart should track start position")
	assert.Equal(t, 18, kr.yankEnd, "yankEnd should track end position")
	assert.Equal(t, 0, kr.index, "index should be reset to 0")
	assert.False(t, kr.lastWasKill, "lastWasKill should be false after yank")
}

func TestKillRing_YankPop_NotActive(t *testing.T) {
	kr := &KillRing{
		ring: [][]rune{
			[]rune("first"),
			[]rune("second"),
		},
		yankActive: false,
	}
	editor := &mockBufferEditor{
		value:  []rune("test"),
		cursor: 4,
	}

	kr.YankPop(editor)

	assert.Equal(t, "test", string(editor.value), "Value should be unchanged")
	assert.Equal(t, 0, kr.index, "Index should be unchanged")
}

func TestKillRing_YankPop_EmptyRing(t *testing.T) {
	kr := &KillRing{
		yankActive: true,
	}
	editor := &mockBufferEditor{
		value:  []rune("test"),
		cursor: 4,
	}

	kr.YankPop(editor)

	assert.Equal(t, "test", string(editor.value), "Value should be unchanged")
}

func TestKillRing_YankPop_SingleEntry(t *testing.T) {
	kr := &KillRing{
		ring: [][]rune{
			[]rune("only"),
		},
		yankActive: true,
		yankStart:  0,
		yankEnd:    4,
	}
	editor := &mockBufferEditor{
		value:  []rune("only"),
		cursor: 4,
	}

	kr.YankPop(editor)

	// With single entry, nothing should change
	assert.Equal(t, "only", string(editor.value), "Value should be unchanged with single entry")
}

func TestKillRing_YankPop_Cycling(t *testing.T) {
	kr := &KillRing{
		ring: [][]rune{
			[]rune("first"),
			[]rune("second"),
			[]rune("third"),
		},
		yankActive: true,
		yankStart:  0,
		yankEnd:    5, // Length of "first"
		index:      0,
	}
	editor := &mockBufferEditor{
		value:  []rune("first"),
		cursor: 5,
	}

	// First yank-pop should replace with "second"
	kr.YankPop(editor)

	assert.Equal(t, "second", string(editor.value), "Should replace with second entry")
	assert.Equal(t, 1, kr.index, "Index should advance to 1")
	assert.Equal(t, 0, kr.yankStart, "yankStart should be updated")
	assert.Equal(t, 6, kr.yankEnd, "yankEnd should be updated")
	assert.True(t, kr.yankActive, "yankActive should remain true")

	// Second yank-pop should replace with "third"
	kr.YankPop(editor)

	assert.Equal(t, "third", string(editor.value), "Should replace with third entry")
	assert.Equal(t, 2, kr.index, "Index should advance to 2")

	// Third yank-pop should cycle back to "first"
	kr.YankPop(editor)

	assert.Equal(t, "first", string(editor.value), "Should cycle back to first entry")
	assert.Equal(t, 0, kr.index, "Index should cycle back to 0")
}

func TestKillRing_YankPop_WithSurroundingText(t *testing.T) {
	kr := &KillRing{
		ring: [][]rune{
			[]rune("AAA"),
			[]rune("BBBBB"),
		},
		yankActive: true,
		yankStart:  7,  // Position after "before "
		yankEnd:    10, // Position after "before AAA"
		index:      0,
	}
	editor := &mockBufferEditor{
		value:  []rune("before AAA after"),
		cursor: 10,
	}

	kr.YankPop(editor)

	assert.Equal(t, "before BBBBB after", string(editor.value), "Should replace yanked region")
	assert.Equal(t, 7, kr.yankStart, "yankStart should be unchanged")
	assert.Equal(t, 12, kr.yankEnd, "yankEnd should be updated for new length")
	assert.Equal(t, 12, editor.cursor, "Cursor should be at end of replacement")
}

func TestKillRing_RecordKill_ResetsYankActive(t *testing.T) {
	kr := &KillRing{
		yankActive: true,
		yankStart:  0,
		yankEnd:    5,
	}
	editor := &mockBufferEditor{}

	kr.RecordKill(editor, []rune("killed"), killDirectionForward)

	assert.False(t, kr.yankActive, "yankActive should be reset by RecordKill")
}

func TestKillRing_YankKillBuffer_ResetsLastWasKill(t *testing.T) {
	kr := &KillRing{
		ring: [][]rune{
			[]rune("text"),
		},
		lastWasKill: true,
	}
	editor := &mockBufferEditor{}

	kr.YankKillBuffer(editor)

	assert.False(t, kr.lastWasKill, "lastWasKill should be reset by yank")
}

func TestKillRing_RecordKill_IsolatesData(t *testing.T) {
	kr := &KillRing{}
	editor := &mockBufferEditor{}

	killed := []rune("test")
	kr.RecordKill(editor, killed, killDirectionForward)

	// Modify original slice
	killed[0] = 'X'

	assert.Equal(t, "test", string(kr.ring[0]), "Kill ring should not be affected by modification of original slice")
}

func TestKillRing_YankKillBuffer_IsolatesData(t *testing.T) {
	original := []rune("test")
	kr := &KillRing{
		ring: [][]rune{original},
	}
	editor := &mockBufferEditor{}

	kr.YankKillBuffer(editor)

	// Modify original slice
	original[0] = 'X'

	// Yank again - should get original value
	editor2 := &mockBufferEditor{}
	kr.YankKillBuffer(editor2)

	assert.Equal(t, "test", string(editor2.value), "Yanked text should not be affected by modification of original slice")
}
