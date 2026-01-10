package shellinput

// clamp returns the value v constrained to the range [low, high].
// If high < low, the arguments are swapped.
func clamp(v, low, high int) int {
	if high < low {
		low, high = high, low
	}
	return min(high, max(low, v))
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// cloneRunes creates a deep copy of a rune slice.
func cloneRunes(r []rune) []rune {
	clone := make([]rune, len(r))
	copy(clone, r)
	return clone
}

// cloneConcatRunes creates a new rune slice containing the concatenation
// of r1 and r2.
func cloneConcatRunes(r1, r2 []rune) []rune {
	clone := make([]rune, len(r1)+len(r2))
	copy(clone, r1)
	copy(clone[len(r1):], r2)
	return clone
}
