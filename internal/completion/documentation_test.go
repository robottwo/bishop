package completion

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocumentationCompleter(t *testing.T) {
	// Create mock filesystem
	rootDir := t.TempDir()
	manDir := filepath.Join(rootDir, "share", "man")
	infoDir := filepath.Join(rootDir, "share", "info")

	err := os.MkdirAll(filepath.Join(manDir, "man1"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(filepath.Join(manDir, "man3"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(infoDir, 0755)
	require.NoError(t, err)

	// Create dummy man pages
	createFile(t, filepath.Join(manDir, "man1", "ls.1.gz"))
	createFile(t, filepath.Join(manDir, "man1", "grep.1"))
	createFile(t, filepath.Join(manDir, "man3", "printf.3.gz"))
	createFile(t, filepath.Join(manDir, "man3", "dummy.3beta")) // complex extension

	// Create dummy info pages
	createFile(t, filepath.Join(infoDir, "grep.info.gz"))
	createFile(t, filepath.Join(infoDir, "sed.info-1.gz")) // split info file
	createFile(t, filepath.Join(infoDir, "sed.info-2.gz"))

	// Set env vars
	t.Setenv("MANPATH", manDir)
	t.Setenv("INFOPATH", infoDir)

	completer := NewDocumentationCompleter()

	// Test MAN completion
	t.Run("man completion", func(t *testing.T) {
		// All man pages
		cands, found := completer.GetCompletions("man", []string{}, "", 0)
		assert.True(t, found)
		assert.Len(t, cands, 4) // ls, grep, printf, dummy

		// Filtering
		cands, _ = completer.GetCompletions("man", []string{"gr"}, "", 0)
		assert.Len(t, cands, 1)
		assert.Equal(t, "grep", cands[0].Value)

		// Sections
		cands, _ = completer.GetCompletions("man", []string{"3", ""}, "", 0)
		assert.Len(t, cands, 2) // printf, dummy
		assert.Equal(t, "dummy", cands[0].Value)
		assert.Equal(t, "printf", cands[1].Value)

		// Section mismatch
		cands, _ = completer.GetCompletions("man", []string{"1", "pr"}, "", 0)
		assert.Len(t, cands, 0)
	})

	// Test INFO completion
	t.Run("info completion", func(t *testing.T) {
		cands, found := completer.GetCompletions("info", []string{}, "", 0)
		assert.True(t, found)
		assert.Len(t, cands, 2) // grep, sed (deduplicated)

		match := false
		for _, c := range cands {
			if c.Value == "sed" {
				match = true
				break
			}
		}
		assert.True(t, match, "sed should be in info completions")
	})

	// Test HELP completion
	t.Run("help completion", func(t *testing.T) {
		cands, found := completer.GetCompletions("help", []string{}, "", 0)
		assert.True(t, found)

		// Should contain builtins
		hasBuiltin := false
		for _, c := range cands {
			if c.Value == "cd" {
				hasBuiltin = true
				break
			}
		}
		assert.True(t, hasBuiltin, "help should contain builtins")

		// Should contain man pages
		hasMan := false
		for _, c := range cands {
			if c.Value == "ls" {
				hasMan = true
				break
			}
		}
		assert.True(t, hasMan, "help should contain man pages")

		// Should contain info pages
		hasInfo := false
		for _, c := range cands {
			if c.Value == "sed" {
				hasInfo = true
				break
			}
		}
		assert.True(t, hasInfo, "help should contain info pages")
	})
}

func createFile(t *testing.T, path string) {
	f, err := os.Create(path)
	require.NoError(t, err)
	f.Close()
}
