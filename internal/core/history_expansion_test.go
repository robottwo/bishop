package core

import (
	"path/filepath"
	"testing"

	"github.com/robottwo/bishop/internal/history"
	"github.com/stretchr/testify/assert"
)

func TestExpandHistory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "history.db")
	hm, err := history.NewHistoryManager(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := hm.Close(); err != nil {
			t.Logf("Error closing history manager: %v", err)
		}
	}()

	// Add history
	// entry 0 (newest): "echo hello"
	// StartCommand creates entry with created_at.
	// GetAllEntries orders by created_at desc.
	// We need to ensure we add them.
	_, err = hm.StartCommand("echo hello", "/tmp")
	if err != nil {
		t.Fatal(err)
	}

	// Test !!
	out, expanded := expandHistory("!!", hm)
	assert.True(t, expanded)
	assert.Equal(t, "echo hello", out)

	// Test !$
	out, expanded = expandHistory("!$", hm)
	assert.True(t, expanded)
	assert.Equal(t, "hello", out)

	// Test mixed
	out, expanded = expandHistory("echo !!", hm)
	assert.True(t, expanded)
	assert.Equal(t, "echo echo hello", out)

	// Test quotes
	out, expanded = expandHistory("'!!'", hm)
	assert.False(t, expanded)
	assert.Equal(t, "'!!'", out)

	// Test double quotes (should expand in our simplified logic)
	out, expanded = expandHistory("\"!!\"", hm)
	assert.True(t, expanded)
	assert.Equal(t, "\"echo hello\"", out)

	// Test escaped
	out, expanded = expandHistory("\\!!", hm)
	assert.False(t, expanded)
	assert.Equal(t, "\\!!", out)

	// Test !$ with multiple args
	_, err = hm.StartCommand("ls -la /tmp", "/tmp")
	if err != nil {
		t.Fatal(err)
	}
	// Now last command is "ls -la /tmp"
	out, expanded = expandHistory("!$", hm)
	assert.True(t, expanded)
	assert.Equal(t, "/tmp", out)
}
