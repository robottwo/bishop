package predict

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBEST_PRACTICES_NotEmpty(t *testing.T) {
	assert.NotEmpty(t, BEST_PRACTICES)
}

func TestBEST_PRACTICES_ContainsGitConventions(t *testing.T) {
	assert.Contains(t, BEST_PRACTICES, "Git commit")
	assert.Contains(t, BEST_PRACTICES, "conventional commit")
}
