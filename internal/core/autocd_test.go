package core

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

func TestIsCompoundCommand(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Simple commands - not compound
		{"ls", false},
		{"/etc", false},
		{"path/to/dir", false},
		{"..", false},
		{".", false},
		{"~", false},
		{"~/Documents", false},
		{"ls -la", false}, // args don't make it compound
		{"/path/with spaces", false},

		// Compound commands
		{"ls | grep foo", true},
		{"ls; pwd", true},
		{"ls && pwd", true},
		{"ls || pwd", true},
		{"ls > file", true},
		{"ls >> file", true},
		{"cat < file", true},
		{"echo $(pwd)", true},
		{"echo `pwd`", true},
		{"(ls)", true},
		{"ls &", true},

		// Quoted content - pipes/operators inside quotes should NOT make it compound
		{"echo 'hello | world'", false},
		{"echo \"hello | world\"", false},
		{"echo 'hello; world'", false},
		{"echo \"hello && world\"", false},

		// Edge cases
		{"", false},
		{"   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isCompoundCommand(tt.input)
			assert.Equal(t, tt.expected, result, "isCompoundCommand(%q)", tt.input)
		})
	}
}

func TestHasArguments(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// Single words - no arguments
		{"ls", false},
		{"/etc", false},
		{"path/to/dir", false},
		{"..", false},
		{"~", false},

		// Commands with arguments
		{"ls -la", true},
		{"cd /tmp", true},
		{"echo hello", true},
		{"git status", true},

		// Quoted content - treated as single word
		{"'path with spaces'", false},
		{"\"path with spaces\"", false},

		// Mixed
		{"echo 'hello world'", true}, // echo + one quoted arg = 2 words
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := hasArguments(tt.input)
			assert.Equal(t, tt.expected, result, "hasArguments(%q)", tt.input)
		})
	}
}

func createTestRunner(t *testing.T) *interp.Runner {
	t.Helper()

	home, err := os.UserHomeDir()
	require.NoError(t, err)

	// Create a simple environment
	env := expand.ListEnviron(
		"HOME="+home,
		"PWD="+os.TempDir(),
		"PATH=/usr/bin:/bin",
		"TEST_VAR=/test/path",
	)

	runner, err := interp.New(interp.Env(env))
	require.NoError(t, err)

	return runner
}

func TestIsCommandOrBuiltin(t *testing.T) {
	runner := createTestRunner(t)

	tests := []struct {
		input    string
		expected bool
	}{
		// Builtins
		{"cd", true},
		{"exit", true},
		{"echo", true},
		{"export", true},
		{"pwd", true},
		{"true", true},
		{"false", true},
		{"history", true},

		// Common commands (should be in PATH)
		{"ls", true},
		{"cat", true},

		// Not commands
		{"/etc", false},
		{"..", false},
		{"nonexistent_command_12345", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := isCommandOrBuiltin(tt.input, runner)
			assert.Equal(t, tt.expected, result, "isCommandOrBuiltin(%q)", tt.input)
		})
	}
}

func TestExpandPath(t *testing.T) {
	home, _ := os.UserHomeDir()

	// Save and restore environment variables (needed for cross-platform compatibility)
	origHome := os.Getenv("HOME")
	origTestVar := os.Getenv("TEST_VAR")
	os.Setenv("HOME", home)
	os.Setenv("TEST_VAR", "/test/path")
	defer func() {
		if origHome != "" {
			os.Setenv("HOME", origHome)
		}
		if origTestVar != "" {
			os.Setenv("TEST_VAR", origTestVar)
		} else {
			os.Unsetenv("TEST_VAR")
		}
	}()

	runner := createTestRunner(t)

	tests := []struct {
		input    string
		expected string
	}{
		// Tilde expansion
		{"~", home},
		{"~/Documents", filepath.Join(home, "Documents")},
		{"~/a/b/c", filepath.Join(home, "a", "b", "c")},

		// Environment variable expansion (from OS env)
		{"$HOME", home},
		{"$HOME/test", home + "/test"},
		{"$TEST_VAR", "/test/path"},
		{"$TEST_VAR/sub", "/test/path/sub"},

		// No expansion needed
		{"/etc", "/etc"},
		{"/tmp/foo", "/tmp/foo"},
		{".", "."},
		{"..", ".."},
		{"relative/path", "relative/path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := expandPath(tt.input, runner)
			assert.Equal(t, tt.expected, result, "expandPath(%q)", tt.input)
		})
	}
}

func TestIsDirectory(t *testing.T) {
	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "autocd_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a file in the temp directory
	tmpFile := filepath.Join(tmpDir, "testfile")
	err = os.WriteFile(tmpFile, []byte("test"), 0644)
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected bool
		unixOnly bool // skip on Windows
	}{
		{tmpDir, true, false},
		{tmpFile, false, false},
		{"/nonexistent/path/12345", false, false},
		{"/etc", true, true},  // Unix only
		{"/tmp", true, true},  // Unix only
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("Skipping Unix-specific path test on Windows")
			}
			result := isDirectory(tt.input)
			assert.Equal(t, tt.expected, result, "isDirectory(%q)", tt.input)
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		// No quoting needed
		{"simple", "simple"},
		{"/etc/passwd", "/etc/passwd"},
		{"path/to/file", "path/to/file"},

		// Quoting needed
		{"path with spaces", "'path with spaces'"},
		{"path\twith\ttabs", "'path\twith\ttabs'"},
		{"path$var", "'path$var'"},
		{"path`cmd`", "'path`cmd`'"},
		{"path*glob", "'path*glob'"},
		{"path?glob", "'path?glob'"},

		// Single quotes in path
		{"it's", "'it'\\''s'"},
		{"path'quote", "'path'\\''quote'"},

		// Empty string
		{"", "''"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := shellQuote(tt.input)
			assert.Equal(t, tt.expected, result, "shellQuote(%q)", tt.input)
		})
	}
}

func TestTryAutocd(t *testing.T) {
	runner := createTestRunner(t)

	// Create a temp directory for testing
	tmpDir, err := os.MkdirTemp("", "autocd_test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	tests := []struct {
		name            string
		input           string
		expectTriggered bool
		expectCd        bool // if triggered, expect "cd " prefix
		unixOnly        bool // skip on Windows
	}{
		// Commands should NOT trigger autocd
		{"ls command", "ls", false, false, false},
		{"pwd command", "pwd", false, false, false},
		{"cd command", "cd /tmp", false, false, false},

		// Existing directories should trigger autocd
		{"/tmp directory", "/tmp", true, true, true},  // Unix only
		{"/etc directory", "/etc", true, true, true},  // Unix only
		{"temp dir", tmpDir, true, true, false},
		{"sub dir", subDir, true, true, false},

		// Special paths (note: "." is a shell builtin for source, so it won't trigger)
		{".. parent", "..", true, true, false},

		// Compound commands should NOT trigger
		{"pipe", "ls | grep foo", false, false, false},
		{"semicolon", "ls; pwd", false, false, false},
		{"and", "ls && pwd", false, false, false},

		// Non-existent paths should NOT trigger
		{"nonexistent", "/nonexistent/path/12345", false, false, false},

		// Empty input
		{"empty", "", false, false, false},
		{"whitespace", "   ", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("Skipping Unix-specific path test on Windows")
			}

			result, triggered := TryAutocd(tt.input, runner)

			assert.Equal(t, tt.expectTriggered, triggered, "TryAutocd(%q) triggered", tt.input)

			if tt.expectCd && triggered {
				assert.True(t, len(result) > 3 && result[:3] == "cd ", "Expected 'cd ' prefix, got %q", result)
			}

			if !triggered && tt.input != "" && tt.input != "   " {
				// For non-empty, non-whitespace input that doesn't trigger, expect original
				assert.Equal(t, tt.input, result, "When not triggered, should return original input")
			}
		})
	}
}

func TestTryAutocd_HomeTilde(t *testing.T) {
	runner := createTestRunner(t)
	home, _ := os.UserHomeDir()

	// Only test if home directory exists
	if _, err := os.Stat(home); err != nil {
		t.Skip("Home directory not accessible")
	}

	result, triggered := TryAutocd("~", runner)
	assert.True(t, triggered, "~ should trigger autocd")
	assert.Equal(t, "cd ~", result, "should produce cd ~ command")

	// Test ~/subdir if it exists
	docs := filepath.Join(home, "Documents")
	if _, err := os.Stat(docs); err == nil {
		result, triggered = TryAutocd("~/Documents", runner)
		assert.True(t, triggered, "~/Documents should trigger autocd")
		assert.Equal(t, "cd ~/Documents", result)
	}
}
