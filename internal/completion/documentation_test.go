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
	err = os.MkdirAll(filepath.Join(manDir, "mann"), 0755)
	require.NoError(t, err)
	err = os.MkdirAll(infoDir, 0755)
	require.NoError(t, err)

	// Create dummy man pages
	createFile(t, filepath.Join(manDir, "man1", "ls.1.gz"))
	createFile(t, filepath.Join(manDir, "man1", "grep.1"))
	createFile(t, filepath.Join(manDir, "man3", "printf.3.gz"))
	createFile(t, filepath.Join(manDir, "man3", "dummy.3beta")) // complex extension
	createFile(t, filepath.Join(manDir, "mann", "tcl.n"))       // named section

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
		assert.Len(t, cands, 5) // ls, grep, printf, dummy, tcl

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

		// Named Section
		cands, _ = completer.GetCompletions("man", []string{"n", ""}, "", 0)
		assert.Len(t, cands, 1)
		assert.Equal(t, "tcl", cands[0].Value)
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

func TestGetEnvPaths(t *testing.T) {
	defaults := []string{"/default/1", "/default/2"}

	tests := []struct {
		name     string
		envVar   string
		envValue string
		want     []string
	}{
		{
			name:     "empty env",
			envVar:   "TEST_PATH_EMPTY",
			envValue: "",
			want:     defaults,
		},
		{
			name:     "no empty parts",
			envVar:   "TEST_PATH_FULL",
			envValue: "/custom/1:/custom/2",
			want:     []string{"/custom/1", "/custom/2"},
		},
		{
			name:     "empty start (defaults first)",
			envVar:   "TEST_PATH_START",
			envValue: ":/custom/1",
			want:     []string{"/default/1", "/default/2", "/custom/1"},
		},
		{
			name:     "empty end (defaults last)",
			envVar:   "TEST_PATH_END",
			envValue: "/custom/1:",
			want:     []string{"/custom/1", "/default/1", "/default/2"},
		},
		{
			name:     "empty middle (defaults middle)",
			envVar:   "TEST_PATH_MIDDLE",
			envValue: "/custom/1::/custom/2",
			want:     []string{"/custom/1", "/default/1", "/default/2", "/custom/2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.envVar, tt.envValue)
			got := getEnvPaths(tt.envVar, defaults)
			assert.Equal(t, tt.want, got)
		})
	}
}
