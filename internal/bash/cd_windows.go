//go:build windows
// +build windows

package bash

import "strings"

// reconstructWindowsPath attempts to reconstruct a Windows path that has lost its backslashes
func reconstructWindowsPath(malformedPath string) string {
	// Split by drive letter
	parts := strings.Split(malformedPath, ":")
	if len(parts) < 2 {
		return malformedPath
	}

	// Don't include backslash in drive - strings.Join will add separators
	drive := parts[0] + ":"
	remainingPath := parts[1]

	// Common Windows directory patterns in order of typical appearance
	patterns := []string{
		"Users",
		"Program Files",
		"Windows",
		"AppData",
		"Local",
		"Temp",
		"ProgramData",
		"Roaming",
		"system32",
		"SysWOW64",
	}

	// Build a more comprehensive reconstruction
	segments := []string{drive}
	currentPos := 0

	// First pass: identify and mark all known patterns

	for currentPos < len(remainingPath) {
		found := false

		// Try to match known patterns
		for _, pattern := range patterns {
			if strings.HasPrefix(remainingPath[currentPos:], pattern) {
				segments = append(segments, pattern)
				currentPos += len(pattern)
				found = true
				break
			}
		}

		if !found {
			// Look for the next pattern
			nextPatternPos := len(remainingPath)

			for _, pattern := range patterns {
				if pos := strings.Index(remainingPath[currentPos:], pattern); pos >= 0 {
					if currentPos+pos < nextPatternPos {
						nextPatternPos = currentPos + pos
					}
				}
			}

			if nextPatternPos < len(remainingPath) {
				// Everything before the next pattern should be a directory name
				dirName := remainingPath[currentPos:nextPatternPos]
				if dirName != "" {
					segments = append(segments, dirName)
				}
				currentPos = nextPatternPos
			} else {
				// No more patterns - add the rest as final segments
				finalPart := remainingPath[currentPos:]
				if finalPart != "" {
					// Try to split the final part into logical segments
					finalSegments := splitFinalPathPart(finalPart)
					segments = append(segments, finalSegments...)
				}
				break
			}
		}
	}

	// Join all segments with backslashes
	return strings.Join(segments, "\\")
}

// splitFinalPathPart splits a concatenated final path part into logical segments
func splitFinalPathPart(part string) []string {
	if len(part) < 4 {
		return []string{part}
	}

	var segments []string
	currentPos := 0

	// Common patterns to look for - order matters!
	// Longer patterns must come before shorter patterns that could be substrings
	// e.g., "subdir" must come before "dir" to avoid splitting "subdir" into "sub\dir"
	patterns := []string{"subdir", "bin", "lib", "src", "test", "temp", "tmp", "data", "app", "exe", "dll", "dir"}

	for currentPos < len(part) {
		found := false

		// Look for known ending patterns
		for _, pattern := range patterns {
			if strings.HasSuffix(part[currentPos:], pattern) && len(part[currentPos:]) > len(pattern) {
				// Found a pattern at the end
				beforePattern := part[currentPos : len(part)-len(pattern)]
				afterPattern := part[len(part)-len(pattern):]

				if beforePattern != "" {
					segments = append(segments, beforePattern)
				}
				segments = append(segments, afterPattern)
				currentPos = len(part)
				found = true
				break
			}
		}

		if !found {
			// Look for number-to-letter transitions
			for i := currentPos + 1; i < len(part); i++ {
				if i > 0 && part[i-1] >= '0' && part[i-1] <= '9' && part[i] >= 'a' && part[i] <= 'z' {
					// Split after the number
					segments = append(segments, part[currentPos:i])
					currentPos = i
					found = true
					break
				}
			}
		}

		if !found {
			// No more splits found, add the rest
			if currentPos < len(part) {
				segments = append(segments, part[currentPos:])
			}
			break
		}
	}

	if len(segments) == 0 {
		return []string{part}
	}

	return segments
}

// findLogicalSplitPoint tries to find a logical place to split a directory name
// For example: "MyAppbin" -> split between "MyApp" and "bin"
// "bish-cd-test1695569554subdir" -> split between "bish-cd-test1695569554" and "subdir"
func findLogicalSplitPoint(s string) int {
	// Common patterns:
	// 1. camelCase to lowercase: "MyAppbin" -> split before "bin"
	// 2. Number+letter: "test1695569554subdir" -> split before "subdir"
	// 3. Hyphen+letter: "bish-cd-test1695569554subdir" -> split before "subdir"

	if len(s) < 4 {
		return 0 // Too short to split meaningfully
	}

	// Look for transitions from numbers to letters
	for i := 1; i < len(s)-1; i++ {
		if i+1 < len(s) && s[i] >= '0' && s[i] <= '9' && s[i+1] >= 'a' && s[i+1] <= 'z' {
			return i + 1 // Split after the number
		}
	}

	// Look for common directory names at the end
	commonEndings := []string{"bin", "lib", "src", "test", "temp", "tmp", "dir", "data", "app", "exe", "dll", "subdir"}
	for _, ending := range commonEndings {
		if strings.HasSuffix(s, ending) && len(s) > len(ending) {
			// Check if what comes before the ending looks like a directory name
			beforeEnding := s[:len(s)-len(ending)]
			if len(beforeEnding) >= 2 && !strings.HasSuffix(beforeEnding, "\\") {
				return len(beforeEnding) // Split before the ending
			}
		}
	}

	return 0 // No good split point found
}
