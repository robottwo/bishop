package docs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractCommands(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"ls -la", []string{"ls"}},
		{"git commit -m 'msg'", []string{"git"}},
		{"ls | grep foo", []string{"ls", "grep"}},
		{"echo hello; cat file", []string{"echo", "cat"}},
		{"if true; then echo yes; fi", []string{"true", "echo"}},
	}

	for _, tt := range tests {
		got := extractCommands(tt.input)
		assert.ElementsMatch(t, tt.expected, got)
	}
}
