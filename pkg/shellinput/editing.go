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

// swapCharacters swaps the character before the cursor with the character at the cursor.
// If at end of line, swap the two characters before the cursor (Emacs-style transposition).
func (m *Model) swapCharacters() {
	if m.pos == 0 || len(m.values[m.selectedValueIndex]) < 2 {
		return
	}

	// If at end of line, swap the two characters before the cursor
	idx := m.pos
	if idx == len(m.values[m.selectedValueIndex]) {
		// Swap idx-1 and idx-2
		m.values[m.selectedValueIndex][idx-1], m.values[m.selectedValueIndex][idx-2] = m.values[m.selectedValueIndex][idx-2], m.values[m.selectedValueIndex][idx-1]
		m.values[0] = m.values[m.selectedValueIndex]
		// Cursor stays at end
		return
	}

	// Swap idx-1 and idx
	m.values[m.selectedValueIndex][idx-1], m.values[m.selectedValueIndex][idx] = m.values[m.selectedValueIndex][idx], m.values[m.selectedValueIndex][idx-1]
	m.values[0] = m.values[m.selectedValueIndex]
	m.SetCursor(m.pos + 1)
}

// swapWords swaps the word before the cursor with the word before that.
func (m *Model) swapWords() {
	v := m.values[m.selectedValueIndex]
	if len(v) == 0 {
		return
	}

	// Step 1: Check if there is a word at or after pos.
	hasWordAfter := false
	temp := m.pos
	for temp < len(v) {
		if !unicode.IsSpace(v[temp]) {
			hasWordAfter = true
			break
		}
		temp++
	}
	w2Start := temp
	var w2End int

	if hasWordAfter {
		// Word 2 starts at w2Start. Find its end.
		w2End = w2Start
		for w2End < len(v) && !unicode.IsSpace(v[w2End]) {
			w2End++
		}
	} else {
		// No word after. Treat word before cursor (or EOL) as Word 2.
		// Scan back from EOL to find end of Word 2.
		w2End = len(v)
		for w2End > 0 && unicode.IsSpace(v[w2End-1]) {
			w2End--
		}
		if w2End == 0 {
			return
		} // No words

		// Find start of Word 2
		w2Start = w2End
		for w2Start > 0 && !unicode.IsSpace(v[w2Start-1]) {
			w2Start--
		}
	}

	// Now we have Word 2 (w2Start, w2End).
	// Find Word 1 before Word 2.
	w1End := w2Start
	// Skip spaces backwards
	for w1End > 0 && unicode.IsSpace(v[w1End-1]) {
		w1End--
	}
	if w1End == 0 {
		return
	} // No Word 1

	// Find start of Word 1
	w1Start := w1End
	for w1Start > 0 && !unicode.IsSpace(v[w1Start-1]) {
		w1Start--
	}

	// Construct new value
	// ... w1 ... w2 ...
	// ... w1Start ... w1End ... w2Start ... w2End ...

	// We need to preserve text between words (usually spaces)
	// part1: 0 to w1Start
	// part2: w1 (w1Start to w1End)
	// part3: w1End to w2Start (separator)
	// part4: w2 (w2Start to w2End)
	// part5: w2End to end

	// New: part1 + part4 + part3 + part2 + part5

	part1 := v[:w1Start]
	part2 := v[w1Start:w1End]
	part3 := v[w1End:w2Start]
	part4 := v[w2Start:w2End]
	part5 := v[w2End:]

	var newValue []rune
	newValue = append(newValue, part1...)
	newValue = append(newValue, part4...)
	newValue = append(newValue, part3...)
	newValue = append(newValue, part2...)
	newValue = append(newValue, part5...)

	m.values[0] = newValue
	m.values[m.selectedValueIndex] = newValue

	// Cursor should move to end of inserted word2 (which is now in second position? No, Bash says "moving point over that word as well")
	// If "one two|", result "two one|". Cursor moves to end of "one" (which is now last).
	// So cursor should be at: len(part1) + len(part4) + len(part3) + len(part2)

	m.SetCursor(len(part1) + len(part4) + len(part3) + len(part2))
}
