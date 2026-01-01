package filesystem

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultFileSystem_Implements_FileSystem(t *testing.T) {
	var _ FileSystem = DefaultFileSystem{}
}

func TestDefaultFileSystem_ReadFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	expectedContent := "Hello, World!"

	err := os.WriteFile(tmpFile, []byte(expectedContent), 0644)
	require.NoError(t, err)

	fs := DefaultFileSystem{}
	content, err := fs.ReadFile(tmpFile)

	assert.NoError(t, err)
	assert.Equal(t, expectedContent, content)
}

func TestDefaultFileSystem_ReadFile_NonExistent(t *testing.T) {
	fs := DefaultFileSystem{}
	_, err := fs.ReadFile("/nonexistent/file/path.txt")

	assert.Error(t, err)
}

func TestDefaultFileSystem_WriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "output.txt")
	content := "Test content"

	fs := DefaultFileSystem{}
	err := fs.WriteFile(tmpFile, content)

	require.NoError(t, err)

	// Verify the file was written correctly
	data, err := os.ReadFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestDefaultFileSystem_Create(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "created.txt")

	fs := DefaultFileSystem{}
	file, err := fs.Create(tmpFile)

	require.NoError(t, err)
	require.NotNil(t, file)
	defer func() { _ = file.Close() }()

	// Verify file exists
	_, err = os.Stat(tmpFile)
	assert.NoError(t, err)
}

func TestDefaultFileSystem_Open(t *testing.T) {
	// Create a temporary file first
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "open_test.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0644)
	require.NoError(t, err)

	fs := DefaultFileSystem{}
	file, err := fs.Open(tmpFile)

	require.NoError(t, err)
	require.NotNil(t, file)
	defer func() { _ = file.Close() }()
}

func TestDefaultFileSystem_Open_NonExistent(t *testing.T) {
	fs := DefaultFileSystem{}
	_, err := fs.Open("/nonexistent/file.txt")

	assert.Error(t, err)
}

func TestDefaultFileSystem_OpenFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "openfile_test.txt")

	fs := DefaultFileSystem{}
	file, err := fs.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY, 0644)

	require.NoError(t, err)
	require.NotNil(t, file)
	defer func() { _ = file.Close() }()

	// Write some data
	_, err = file.WriteString("test data")
	assert.NoError(t, err)
}

func TestDefaultFileSystem_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "roundtrip.txt")
	originalContent := "Round trip content\nWith multiple lines\n"

	fs := DefaultFileSystem{}

	// Write
	err := fs.WriteFile(tmpFile, originalContent)
	require.NoError(t, err)

	// Read back
	readContent, err := fs.ReadFile(tmpFile)
	require.NoError(t, err)

	assert.Equal(t, originalContent, readContent)
}
