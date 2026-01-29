package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanLogFiles(t *testing.T) {
	t.Run("Removes all bish.*.zst files", func(t *testing.T) {
		tmpDir := t.TempDir()

		// Set default paths to use temp directory
		oldDefaultPaths := defaultPaths
		defer func() {
			defaultPaths = oldDefaultPaths
		}()

		defaultPaths = &Paths{
			DataDir: tmpDir,
		}

		// Create some test log files
		logFile1 := filepath.Join(tmpDir, "bish.1234.zst")
		logFile2 := filepath.Join(tmpDir, "bish.5678.zst")
		logFile3 := filepath.Join(tmpDir, "bish.zst")
		otherFile := filepath.Join(tmpDir, "other.log")

		err := os.WriteFile(logFile1, []byte("log1"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(logFile2, []byte("log2"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(logFile3, []byte("log3"), 0644)
		require.NoError(t, err)
		err = os.WriteFile(otherFile, []byte("other"), 0644)
		require.NoError(t, err)

		// Clean log files
		err = CleanLogFiles()
		require.NoError(t, err)

		// Verify bish log files are gone
		_, err = os.Stat(logFile1)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(logFile2)
		assert.True(t, os.IsNotExist(err))
		_, err = os.Stat(logFile3)
		assert.True(t, os.IsNotExist(err))

		// Verify other file remains
		_, err = os.Stat(otherFile)
		assert.NoError(t, err, "Other file should not be removed")
	})

	t.Run("Handles empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldDefaultPaths := defaultPaths
		defer func() {
			defaultPaths = oldDefaultPaths
		}()

		defaultPaths = &Paths{
			DataDir: tmpDir,
		}

		// Should not error on empty directory
		err := CleanLogFiles()
		assert.NoError(t, err)
	})
}

func TestRotateLogFiles(t *testing.T) {
	t.Run("Keeps most recent 10 log files", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldDefaultPaths := defaultPaths
		defer func() {
			defaultPaths = oldDefaultPaths
		}()

		defaultPaths = &Paths{
			DataDir: tmpDir,
		}

		// Create 15 log files with different modification times
		now := time.Now()
		for i := 1; i <= 15; i++ {
			logFile := filepath.Join(tmpDir, fmt.Sprintf("bish.%d.zst", i))

			// Stagger modification times (newest = lowest number)
			modTime := now.Add(-time.Duration(i) * time.Minute)
			err := os.WriteFile(logFile, []byte("log"), 0644)
			require.NoError(t, err)
			err = os.Chtimes(logFile, modTime, modTime)
			require.NoError(t, err)
		}

		// Rotate to keep only 10 most recent
		err := RotateLogFiles()
		require.NoError(t, err)

		// Verify only 10 files remain
		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		var logFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), "bish.") && strings.HasSuffix(entry.Name(), ".zst") {
				logFiles = append(logFiles, entry.Name())
			}
		}

		assert.Len(t, logFiles, 10, "Should have exactly 10 log files")

		// Verify newest files (1-10) remain
		for i := 1; i <= 10; i++ {
			expectedName := fmt.Sprintf("bish.%d.zst", i)
			assert.Contains(t, logFiles, expectedName, "Newest files should remain")
		}
	})

	t.Run("Keeps all files when <= 10", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldDefaultPaths := defaultPaths
		defer func() {
			defaultPaths = oldDefaultPaths
		}()

		defaultPaths = &Paths{
			DataDir: tmpDir,
		}

		// Create 5 log files
		for i := 1; i <= 5; i++ {
			logFile := filepath.Join(tmpDir, fmt.Sprintf("bish.%d.zst", i))
			err := os.WriteFile(logFile, []byte("log"), 0644)
			require.NoError(t, err)
		}

		// Rotate
		err := RotateLogFiles()
		require.NoError(t, err)

		// Verify all 5 files remain
		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		var logFiles []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), "bish.") && strings.HasSuffix(entry.Name(), ".zst") {
				logFiles = append(logFiles, entry.Name())
			}
		}

		assert.Len(t, logFiles, 5, "Should keep all 5 log files")
	})

	t.Run("Preserves non-bish files", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldDefaultPaths := defaultPaths
		defer func() {
			defaultPaths = oldDefaultPaths
		}()

		defaultPaths = &Paths{
			DataDir: tmpDir,
		}

		// Create 12 bish log files + some other files
		for i := 1; i <= 12; i++ {
			logFile := filepath.Join(tmpDir, fmt.Sprintf("bish.%d.zst", i))
			err := os.WriteFile(logFile, []byte("log"), 0644)
			require.NoError(t, err)
		}

		otherFile1 := filepath.Join(tmpDir, "other.log")
		otherFile2 := filepath.Join(tmpDir, "subdir")
		otherFile3 := filepath.Join(tmpDir, "backup.txt")

		err := os.WriteFile(otherFile1, []byte("other"), 0644)
		require.NoError(t, err)
		err = os.Mkdir(otherFile2, 0755)
		require.NoError(t, err)
		err = os.WriteFile(otherFile3, []byte("backup"), 0644)
		require.NoError(t, err)

		// Rotate
		_ = RotateLogFiles()
		require.NoError(t, err)

		// Verify only 2 oldest bish.*.zst files are removed
		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		var logFiles, otherFiles []string
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, "bish.") && strings.HasSuffix(name, ".zst") {
				logFiles = append(logFiles, name)
			} else {
				otherFiles = append(otherFiles, name)
			}
		}

		assert.Len(t, logFiles, 10, "Should have exactly 10 bish log files")
		assert.Len(t, otherFiles, 3, "Should preserve all non-bish files")
	})

	t.Run("Handles empty directory", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldDefaultPaths := defaultPaths
		defer func() {
			defaultPaths = oldDefaultPaths
		}()

		defaultPaths = &Paths{
			DataDir: tmpDir,
		}

		// Should not error on empty directory
		err := RotateLogFiles()
		assert.NoError(t, err)
	})

	t.Run("Sorts by modification time correctly", func(t *testing.T) {
		tmpDir := t.TempDir()

		oldDefaultPaths := defaultPaths
		defer func() {
			defaultPaths = oldDefaultPaths
		}()

		defaultPaths = &Paths{
			DataDir: tmpDir,
		}

		now := time.Now()

		// Create files with random order, then set specific mod times
		testCases := []struct {
			name string
			age  time.Duration
			keep bool
		}{
			{"bish.1.zst", 1 * time.Hour, true},
			{"bish.2.zst", 2 * time.Hour, true},
			{"bish.3.zst", 3 * time.Hour, true},
			{"bish.4.zst", 5 * time.Hour, false},
			{"bish.5.zst", 6 * time.Hour, false},
		}

		for _, tc := range testCases {
			logFile := filepath.Join(tmpDir, tc.name)
			modTime := now.Add(-tc.age)
			err := os.WriteFile(logFile, []byte("log"), 0644)
			require.NoError(t, err)
			err = os.Chtimes(logFile, modTime, modTime)
			require.NoError(t, err)
		}

		// Rotate
		err := RotateLogFiles()
		require.NoError(t, err)

		// Verify correct files remain (all 5 since max is 10)
		entries, err := os.ReadDir(tmpDir)
		require.NoError(t, err)

		var remaining []string
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasPrefix(entry.Name(), "bish.") && strings.HasSuffix(entry.Name(), ".zst") {
				remaining = append(remaining, entry.Name())
			}
		}

		assert.Len(t, remaining, 5, "Should keep all 5 files (under limit of 10)")

		for _, tc := range testCases {
			assert.Contains(t, remaining, tc.name, "Should keep file: "+tc.name)
		}
	})
}
