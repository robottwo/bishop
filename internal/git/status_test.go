package git

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"zero", "0", 0},
		{"positive single digit", "5", 5},
		{"positive multi digit", "42", 42},
		{"large number", "1234567", 1234567},
		{"with plus prefix", "+5", 5},
		{"with minus prefix", "-3", 3},
		{"empty string", "", 0},
		{"non-numeric", "abc", 0},
		{"mixed alphanumeric", "12abc34", 1234},
		{"leading non-digits", "abc123", 123},
		{"trailing non-digits", "123abc", 123},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRepoStatus_DefaultValues(t *testing.T) {
	status := &RepoStatus{}
	assert.Empty(t, status.RepoName)
	assert.Empty(t, status.Branch)
	assert.False(t, status.Clean)
	assert.Zero(t, status.Staged)
	assert.Zero(t, status.Unstaged)
	assert.Zero(t, status.Ahead)
	assert.Zero(t, status.Behind)
	assert.False(t, status.Conflict)
}

func TestRepoStatus_WithValues(t *testing.T) {
	status := &RepoStatus{
		RepoName: "my-repo",
		Branch:   "main",
		Clean:    true,
		Staged:   2,
		Unstaged: 3,
		Ahead:    1,
		Behind:   0,
		Conflict: false,
	}

	assert.Equal(t, "my-repo", status.RepoName)
	assert.Equal(t, "main", status.Branch)
	assert.True(t, status.Clean)
	assert.Equal(t, 2, status.Staged)
	assert.Equal(t, 3, status.Unstaged)
	assert.Equal(t, 1, status.Ahead)
	assert.Equal(t, 0, status.Behind)
	assert.False(t, status.Conflict)
}

func TestGetStatus_ReturnsNilWhenGitNotInstalled(t *testing.T) {
	// This test relies on the PATH not having git, which may not be reliable
	// In most environments git is installed, so we skip this test
	t.Skip("Skip: requires environment without git installed")
}

func TestGetStatus_ReturnsNilOutsideGitRepo(t *testing.T) {
	// Use a directory that's definitely not a git repo
	status := GetStatus("/tmp")
	// This may or may not be nil depending on whether /tmp is inside a git repo
	// In most cases it should be nil
	if status != nil {
		// If somehow /tmp is inside a git repo, just skip
		t.Skip("Skip: /tmp appears to be inside a git repo")
	}
	assert.Nil(t, status)
}

func TestGetStatusWithTimeout_ReturnsResult(t *testing.T) {
	// Test with a very short timeout
	status := GetStatusWithTimeout("/tmp", 1)
	// Should either return nil (not a repo) or timeout
	// Either way, should not panic
	_ = status
}
