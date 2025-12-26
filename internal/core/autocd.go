package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/robottwo/bishop/internal/environment"
	"mvdan.cc/sh/v3/interp"
)

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

// isExternalCommand checks if the word is a command found in PATH or a defined function
// Returns true if the word is a function or found in PATH
// Note: This does NOT check for builtins - those are handled by the interpreter
func isExternalCommand(word string, runner *interp.Runner) bool {
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
	// The key is: if the first word is a command in PATH or a function, we shouldn't treat the rest as a path
	// Note: builtins are handled by the AutocdExecHandler fallback mechanism
	firstWord := strings.Fields(input)[0]
	if isExternalCommand(firstWord, runner) {
		return input, false
	}

	// Expand the path
	expandedPath := expandPath(input, runner)

	// If it has arguments and the first word is not a command, check if it might be a path with spaces
	// For now, we only autocd on single-word inputs or quoted paths
	if hasArguments(input) {
		// Check if the whole input (possibly a path with spaces that wasn't quoted) is a directory
		if !isDirectory(expandedPath) {
			return input, false
		}
	}

	// Check if it's a directory
	if !isDirectory(expandedPath) {
		return input, false
	}

	// Return the cd command
	return "cd " + shellQuote(input), true
}

// autocdRunner stores the runner for autocd verbose output
var autocdRunner *interp.Runner

// SetAutocdRunner sets the runner for the autocd handler
func SetAutocdRunner(runner *interp.Runner) {
	autocdRunner = runner
}

// NewAutocdExecHandler creates an ExecHandler that implements autocd.
// It checks if path-like inputs are directories and executes cd instead.
// This allows builtins and commands to take precedence naturally without
// needing to maintain a hardcoded list of builtin names.
func NewAutocdExecHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Check if autocd is enabled
			if autocdRunner == nil || !environment.IsAutocdEnabled(autocdRunner) {
				return next(ctx, args)
			}

			// Only process single-word commands (no arguments)
			if len(args) > 1 {
				return next(ctx, args)
			}

			// Get the command name (first argument)
			cmdName := args[0]

			// Quick check: if it's clearly not a path, skip autocd logic
			// This avoids overhead for normal commands
			if !mightBePath(cmdName) {
				return next(ctx, args)
			}

			// Check if it's a command in PATH or a defined function
			// If so, let it execute normally
			if isExternalCommand(cmdName, autocdRunner) {
				return next(ctx, args)
			}

			// Expand the path and check if it's a directory
			expandedPath := expandPath(cmdName, autocdRunner)
			if isDirectory(expandedPath) {
				// It's a directory! Execute cd instead
				hc := interp.HandlerCtx(ctx)
				if environment.IsAutocdVerbose(autocdRunner) {
					fmt.Fprintln(hc.Stderr, "cd "+shellQuote(cmdName))
				}

				// Execute cd with the original path (let cd handle expansion)
				return next(ctx, []string{"bish_cd", cmdName})
			}

			// Not a directory, let normal command execution happen
			return next(ctx, args)
		}
	}
}

// mightBePath does a quick check to see if the string might be a filesystem path
// This is used to avoid unnecessary directory checks for obvious non-paths
func mightBePath(s string) bool {
	// Paths typically start with /, ~, or .
	// Or contain / somewhere
	if len(s) == 0 {
		return false
	}

	first := s[0]
	if first == '/' || first == '~' || first == '.' {
		return true
	}

	// Check for embedded slashes (relative paths like foo/bar)
	return strings.Contains(s, "/")
}
