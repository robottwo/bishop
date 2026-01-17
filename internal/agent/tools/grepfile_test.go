package tools

import (
	"bytes"
	"os"
	"strings"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

func TestGrepFileToolDefinition(t *testing.T) {
	assert.Equal(t, openai.ToolType("function"), GrepFileToolDefinition.Type)
	assert.Equal(t, "grep_file", GrepFileToolDefinition.Function.Name)
	assert.Equal(
		t,
		"Search for a pattern (regex) in a file and return matching lines with line numbers. Similar to grep command.",
		GrepFileToolDefinition.Function.Description,
	)
	parameters, ok := GrepFileToolDefinition.Function.Parameters.(*jsonschema.Definition)
	assert.True(t, ok, "Parameters should be of type *jsonschema.Definition")
	assert.Equal(t, jsonschema.DataType("object"), parameters.Type)
	assert.Equal(t, "Absolute path to the file to search", parameters.Properties["path"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["path"].Type)
	assert.Equal(t, "Regular expression pattern to search for", parameters.Properties["pattern"].Description)
	assert.Equal(t, jsonschema.DataType("string"), parameters.Properties["pattern"].Type)
	assert.Equal(
		t,
		"Optional. Number of lines to show before and after each match (like grep -C). Default is 0.",
		parameters.Properties["context_lines"].Description,
	)
	assert.Equal(t, jsonschema.DataType("integer"), parameters.Properties["context_lines"].Type)
	assert.Equal(t, []string{"path", "pattern"}, parameters.Required)
}

func TestGrepFileTool(t *testing.T) {
	tempFile, err := os.CreateTemp("", "testfile*.txt")
	assert.NoError(t, err)
	t.Cleanup(func() {
		assert.NoError(t, os.Remove(tempFile.Name()))
	})

	content := "Line 1\nLine 2 error\nLine 3\nLine 4 error\nLine 5\nLine 6\nLine 7 ERROR"
	_, err = tempFile.WriteString(content)
	assert.NoError(t, err)
	_ = tempFile.Close()

	runner, _ := interp.New()
	logger := zap.NewNop()

	t.Run("Valid file path with matching pattern", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "error"}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "2:Line 2 error")
		assert.Contains(t, result, "4:Line 4 error")
		assert.NotContains(t, result, "Line 1")
		assert.NotContains(t, result, "Line 3")
	})

	t.Run("Valid file path with no matches", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "nomatch"}
		result := GrepFileTool(runner, logger, params)
		assert.Equal(t, "No matches found.", result)
	})

	t.Run("Invalid file path", func(t *testing.T) {
		params := map[string]any{"path": "nonexistent.txt", "pattern": "error"}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "Error opening file")
	})

	t.Run("Invalid regex pattern", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "[invalid("}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "Invalid regex pattern")
	})

	t.Run("With context lines", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "Line 4 error", "context_lines": 1.0}
		result := GrepFileTool(runner, logger, params)
		// Should include line 3 (before), line 4 (match), and line 5 (after)
		assert.Contains(t, result, "3-Line 3")
		assert.Contains(t, result, "4:Line 4 error")
		assert.Contains(t, result, "5-Line 5")
		// Should not include line 1 or 2
		assert.NotContains(t, result, "Line 1")
		assert.NotContains(t, result, "Line 2")
	})

	t.Run("With context lines showing separator", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "error", "context_lines": 1.0}
		result := GrepFileTool(runner, logger, params)
		// Should show separator between non-adjacent matches
		// Line 2 error with context: 1-3
		// Line 4 error with context: 3-5
		// These overlap, so no separator
		assert.Contains(t, result, "1-Line 1")
		assert.Contains(t, result, "2:Line 2 error")
		assert.Contains(t, result, "3-Line 3")
		assert.Contains(t, result, "4:Line 4 error")
		assert.Contains(t, result, "5-Line 5")
	})

	t.Run("Context lines at file boundaries", func(t *testing.T) {
		smallFile, err := os.CreateTemp("", "smallfile*.txt")
		assert.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, os.Remove(smallFile.Name()))
		})

		content := "First line\nSecond line"
		_, err = smallFile.WriteString(content)
		assert.NoError(t, err)
		_ = smallFile.Close()

		// Match first line with context_lines=5 should not cause issues
		params := map[string]any{"path": smallFile.Name(), "pattern": "First", "context_lines": 5.0}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "1:First line")
		assert.Contains(t, result, "2-Second line")
	})

	t.Run("Case sensitive pattern", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "ERROR"}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "7:Line 7 ERROR")
		assert.NotContains(t, result, "Line 2 error")
		assert.NotContains(t, result, "Line 4 error")
	})

	t.Run("Case insensitive pattern using regex", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "(?i)error"}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "2:Line 2 error")
		assert.Contains(t, result, "4:Line 4 error")
		assert.Contains(t, result, "7:Line 7 ERROR")
	})

	t.Run("File content exceeding max view size", func(t *testing.T) {
		largeFile, err := os.CreateTemp("", "largefile*.txt")
		assert.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, os.Remove(largeFile.Name()))
		})

		// Create content that will exceed MAX_VIEW_SIZE when grep results are returned
		var largeContent bytes.Buffer
		// Write many matching lines to ensure output exceeds MAX_VIEW_SIZE
		for i := 0; i < 10000; i++ {
			largeContent.WriteString(strings.Repeat("A", 100) + " MATCH\n")
		}
		_, err = largeFile.Write(largeContent.Bytes())
		assert.NoError(t, err)
		_ = largeFile.Close()

		params := map[string]any{"path": largeFile.Name(), "pattern": "MATCH"}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "<bish:truncated />")
	})

	t.Run("Missing path parameter", func(t *testing.T) {
		params := map[string]any{"pattern": "error"}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "failed to parse parameter 'path'")
	})

	t.Run("Missing pattern parameter", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name()}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "failed to parse parameter 'pattern'")
	})

	t.Run("Invalid context_lines parameter type", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "error", "context_lines": "invalid"}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "failed to parse parameter 'context_lines'")
	})

	t.Run("Zero context lines", func(t *testing.T) {
		params := map[string]any{"path": tempFile.Name(), "pattern": "Line 4 error", "context_lines": 0.0}
		result := GrepFileTool(runner, logger, params)
		assert.Contains(t, result, "4:Line 4 error")
		assert.NotContains(t, result, "Line 3")
		assert.NotContains(t, result, "Line 5")
	})

	t.Run("Multiple matches with gaps and separator", func(t *testing.T) {
		gapFile, err := os.CreateTemp("", "gapfile*.txt")
		assert.NoError(t, err)
		t.Cleanup(func() {
			assert.NoError(t, os.Remove(gapFile.Name()))
		})

		content := "Line 1\nMatch line 2\nLine 3\nLine 4\nLine 5\nMatch line 6\nLine 7"
		_, err = gapFile.WriteString(content)
		assert.NoError(t, err)
		_ = gapFile.Close()

		params := map[string]any{"path": gapFile.Name(), "pattern": "Match", "context_lines": 0.0}
		result := GrepFileTool(runner, logger, params)
		// Should have separator between non-adjacent matches
		assert.Contains(t, result, "2:Match line 2")
		assert.Contains(t, result, "--")
		assert.Contains(t, result, "6:Match line 6")
	})
}
