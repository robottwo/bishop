package core

import (
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// shellBuiltins is a list of shell builtins that should not trigger autocd
var shellBuiltins = map[string]bool{
	// Standard POSIX builtins
	"cd":       true,
	"exit":     true,
	"export":   true,
	"readonly": true,
	"unset":    true,
	"set":      true,
	"shift":    true,
	"return":   true,
	"break":    true,
	"continue": true,
	"eval":     true,
	"exec":     true,
	"source":   true,
	".":        true,
	"alias":    true,
	"unalias":  true,
	"read":     true,
	"trap":     true,
	"wait":     true,
	"jobs":     true,
	"fg":       true,
	"bg":       true,
	"kill":     true,
	"pwd":      true,
	"echo":     true,
	"printf":   true,
	"test":     true,
	"[":        true,
	"[[":       true,
	"true":     true,
	"false":    true,
	"type":     true,
	"command":  true,
	"builtin":  true,
	"local":    true,
	"declare":  true,
	"typeset":  true,
	"getopts":  true,
	"hash":     true,
	"ulimit":   true,
	"umask":    true,
	"times":    true,
	"let":      true,
	// Bishop-specific builtins
	"bish_cd":        true,
	"bish_typeset":   true,
	"bish_analytics": true,
	"bish_evaluate":  true,
	"history":        true,
	"complete":       true,
}

// isCompoundCommand checks if input contains shell operators that indicate
// it's a compound command rather than a simple word/path
func isCompoundCommand(input string) bool {
	// Track quote state
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		// Handle escape character
		if escaped {
			escaped = false
			continue
		}

		if r == '\\' && !inSingleQuote {
			escaped = true
			continue
		}

		// Handle quotes
		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			continue
		}
		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			continue
		}

		// Skip if inside quotes
		if inSingleQuote || inDoubleQuote {
			continue
		}

		// Check for shell operators outside of quotes
		switch r {
		case '|':
			return true // pipe
		case ';':
			return true // command separator
		case '&':
			return true // background or &&
		case '<', '>':
			return true // redirects
		case '(':
			return true // subshell
		case ')':
			return true // subshell
		case '`':
			return true // command substitution
		case '$':
			// Check for $(...) command substitution
			if i+1 < len(runes) && runes[i+1] == '(' {
				return true
			}
		}
	}

	return false
}

// hasArguments checks if the input has multiple space-separated words
// (i.e., command with arguments rather than just a path)
func hasArguments(input string) bool {
	// Track quote state
	inSingleQuote := false
	inDoubleQuote := false
	escaped := false
	wordCount := 0
	inWord := false

	for _, r := range input {
		// Handle escape character
		if escaped {
			escaped = false
			if !inWord {
				inWord = true
				wordCount++
			}
			continue
		}

		if r == '\\' && !inSingleQuote {
			escaped = true
			continue
		}

		// Handle quotes
		if r == '\'' && !inDoubleQuote {
			inSingleQuote = !inSingleQuote
			if !inWord {
				inWord = true
				wordCount++
			}
			continue
		}
		if r == '"' && !inSingleQuote {
			inDoubleQuote = !inDoubleQuote
			if !inWord {
				inWord = true
				wordCount++
			}
			continue
		}

		// Check for whitespace outside quotes
		if !inSingleQuote && !inDoubleQuote && (r == ' ' || r == '\t') {
			inWord = false
			continue
		}

		// Non-whitespace character
		if !inWord {
			inWord = true
			wordCount++
			if wordCount > 1 {
				return true
			}
		}
	}

	return wordCount > 1
}

// isCommandOrBuiltin checks if the word is a known command
// Returns true if the word is a builtin, function, alias, or found in PATH
func isCommandOrBuiltin(word string, runner *interp.Runner) bool {
	// Check if it's a builtin
	if shellBuiltins[word] {
		return true
	}

	// Check if it's a defined function
	if runner.Funcs[word] != nil {
		return true
	}

	// Check if it's in PATH
	_, err := exec.LookPath(word)
	return err == nil
}

// expandPath expands ~ and environment variables in a path
func expandPath(path string, runner *interp.Runner) string {
	// Handle tilde expansion
	if strings.HasPrefix(path, "~/") {
		home := runner.Vars["HOME"].String()
		if home == "" {
			if usr, err := user.Current(); err == nil {
				home = usr.HomeDir
			}
		}
		if home != "" {
			path = filepath.Join(home, path[2:])
		}
	} else if path == "~" {
		home := runner.Vars["HOME"].String()
		if home == "" {
			if usr, err := user.Current(); err == nil {
				home = usr.HomeDir
			}
		}
		if home != "" {
			path = home
		}
	} else if strings.HasPrefix(path, "~") {
		// Handle ~username expansion
		// Find the end of the username (first slash or end of string)
		slashIdx := strings.Index(path[1:], "/")
		var username string
		var rest string
		if slashIdx == -1 {
			username = path[1:]
			rest = ""
		} else {
			username = path[1 : slashIdx+1]
			rest = path[slashIdx+2:] // +2 to skip both the offset and the slash
		}
		if username != "" {
			if usr, err := user.Lookup(username); err == nil {
				path = filepath.Join(usr.HomeDir, rest)
			}
		}
	}

	// Handle environment variable expansion
	path = os.Expand(path, func(key string) string {
		if val := runner.Vars[key].String(); val != "" {
			return val
		}
		return os.Getenv(key)
	})

	return path
}

// isDirectory checks if path exists and is a directory
func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// shellQuote properly quotes a path for shell execution
// This handles paths with spaces and special characters
func shellQuote(s string) string {
	// If the string is empty, return empty quotes
	if s == "" {
		return "''"
	}

	// Check if quoting is needed
	needsQuoting := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\'' || r == '"' ||
			r == '\\' || r == '$' || r == '`' || r == '!' || r == '*' ||
			r == '?' || r == '[' || r == ']' || r == '(' || r == ')' ||
			r == '{' || r == '}' || r == '|' || r == '&' || r == ';' ||
			r == '<' || r == '>' || r == '#' {
			needsQuoting = true
			break
		}
	}

	if !needsQuoting {
		return s
	}

	// Use single quotes, escaping any single quotes within
	var sb strings.Builder
	sb.WriteByte('\'')
	for _, r := range s {
		if r == '\'' {
			// End single quote, add escaped single quote, start single quote again
			sb.WriteString("'\\''")
		} else {
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('\'')
	return sb.String()
}

// TryAutocd checks if the input should trigger autocd and returns the modified command
// Returns the original input if autocd should not trigger, or "cd <path>" if it should
// Also returns a boolean indicating whether autocd was triggered
func TryAutocd(input string, runner *interp.Runner) (string, bool) {
	input = strings.TrimSpace(input)

	// Don't process empty input
	if input == "" {
		return input, false
	}

	// Don't process compound commands
	if isCompoundCommand(input) {
		return input, false
	}

	// Don't process commands with arguments (but allow paths with spaces)
	// We need to be careful here: "ls -la" has arguments, but "/path/to/dir" is just a path
	// The key is: if the first word is a command, we shouldn't treat the rest as a path
	firstWord := strings.Fields(input)[0]
	if isCommandOrBuiltin(firstWord, runner) {
		return input, false
	}

	// If it has arguments and the first word is not a command, check if it might be a path with spaces
	// For now, we only autocd on single-word inputs or quoted paths
	if hasArguments(input) {
		// Check if the whole input (possibly a path with spaces that wasn't quoted) is a directory
		expandedPath := expandPath(input, runner)
		if !isDirectory(expandedPath) {
			return input, false
		}
	}

	// Expand the path
	expandedPath := expandPath(input, runner)

	// Check if it's a directory
	if !isDirectory(expandedPath) {
		return input, false
	}

	// Return the cd command
	return "cd " + shellQuote(input), true
}
