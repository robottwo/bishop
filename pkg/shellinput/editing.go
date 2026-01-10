package shellinput

import "unicode"

// wordBackward moves the cursor one word to the left. If input is masked, move
// input to the start so as not to reveal word breaks in the masked input.
func (m *Model) wordBackward() {
	if m.pos == 0 || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.CursorStart()
		return
	}

	i := m.pos - 1
	for i >= 0 {
		if unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos - 1)
			i--
		} else {
			break
		}
	}

	for i >= 0 {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos - 1)
			i--
		} else {
			break
		}
	}
}

// wordForward moves the cursor one word to the right. If the input is masked,
// move input to the end so as not to reveal word breaks in the masked input.
func (m *Model) wordForward() {
	if m.pos >= len(m.values[m.selectedValueIndex]) || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.CursorEnd()
		return
	}

	i := m.pos
	for i < len(m.values[m.selectedValueIndex]) {
		if unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos + 1)
			i++
		} else {
			break
		}
	}

	for i < len(m.values[m.selectedValueIndex]) {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][i]) {
			m.SetCursor(m.pos + 1)
			i++
		} else {
			break
		}
	}
}

// deleteWordBackward deletes the word left to the cursor. If the input is masked
// delete everything before the cursor so as not to reveal word breaks in the
// masked input.
func (m *Model) deleteWordBackward() {
	if m.pos == 0 || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.deleteBeforeCursor()
		return
	}

	// Linter note: it's critical that we acquire the initial cursor position
	// here prior to altering it via SetCursor() below. As such, moving this
	// call into the corresponding if clause does not apply here.
	oldPos := m.pos //nolint:ifshort

	m.SetCursor(m.pos - 1)
	for unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
		if m.pos <= 0 {
			break
		}
		// ignore series of whitespace before cursor
		m.SetCursor(m.pos - 1)
	}

	for m.pos > 0 {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
			m.SetCursor(m.pos - 1)
		} else {
			if m.pos > 0 {
				// keep the previous space
				m.SetCursor(m.pos + 1)
			}
			break
		}
	}

	var newValue []rune
	if oldPos > len(m.values[m.selectedValueIndex]) {
		newValue = cloneRunes(m.values[m.selectedValueIndex][:m.pos])
	} else {
		newValue = cloneConcatRunes(m.values[m.selectedValueIndex][:m.pos], m.values[m.selectedValueIndex][oldPos:])
	}

	m.recordKill(m.values[m.selectedValueIndex][m.pos:oldPos], killDirectionBackward)

	m.Err = m.validate(newValue)
	m.values[0] = newValue
	m.selectedValueIndex = 0
}

// deleteWordForward deletes the word right to the cursor. If input is masked
// delete everything after the cursor so as not to reveal word breaks in the
// masked input.
func (m *Model) deleteWordForward() {
	if m.pos >= len(m.values[m.selectedValueIndex]) || len(m.values[m.selectedValueIndex]) == 0 {
		return
	}

	if m.EchoMode != EchoNormal {
		m.deleteAfterCursor()
		return
	}

	oldPos := m.pos
	m.SetCursor(m.pos + 1)
	for unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
		// ignore series of whitespace after cursor
		m.SetCursor(m.pos + 1)

		if m.pos >= len(m.values[m.selectedValueIndex]) {
			break
		}
	}

	for m.pos < len(m.values[m.selectedValueIndex]) {
		if !unicode.IsSpace(m.values[m.selectedValueIndex][m.pos]) {
			m.SetCursor(m.pos + 1)
		} else {
			break
		}
	}

	var newValue []rune
	if m.pos > len(m.values[m.selectedValueIndex]) {
		newValue = cloneRunes(m.values[m.selectedValueIndex][:oldPos])
	} else {
		newValue = cloneConcatRunes(m.values[m.selectedValueIndex][:oldPos], m.values[m.selectedValueIndex][m.pos:])
	}

	killEnd := min(m.pos, len(m.values[m.selectedValueIndex]))
	m.recordKill(m.values[m.selectedValueIndex][oldPos:killEnd], killDirectionForward)
	m.Err = m.validate(newValue)
	m.values[0] = newValue
	m.selectedValueIndex = 0
	m.SetCursor(oldPos)
}
