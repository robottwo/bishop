package predict

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPREDICTED_COMMAND_SCHEMA_Generated(t *testing.T) {
	// Verify schema is generated and not nil
	require.NotNil(t, PREDICTED_COMMAND_SCHEMA)

	// Verify it can be marshaled to JSON
	jsonBytes, err := PREDICTED_COMMAND_SCHEMA.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Verify it contains expected field
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "predicted_command")
}

func TestEXPLAINED_COMMAND_SCHEMA_Generated(t *testing.T) {
	// Verify schema is generated and not nil
	require.NotNil(t, EXPLAINED_COMMAND_SCHEMA)

	// Verify it can be marshaled to JSON
	jsonBytes, err := EXPLAINED_COMMAND_SCHEMA.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Verify it contains expected field
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "explanation")
}

func TestCOMPLETION_CANDIDATES_SCHEMA_Generated(t *testing.T) {
	// Verify schema is generated and not nil
	require.NotNil(t, COMPLETION_CANDIDATES_SCHEMA)

	// Verify it can be marshaled to JSON
	jsonBytes, err := COMPLETION_CANDIDATES_SCHEMA.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)

	// Verify it contains expected field
	jsonStr := string(jsonBytes)
	assert.Contains(t, jsonStr, "candidates")
}

func TestPredictedCommand_Struct(t *testing.T) {
	cmd := PredictedCommand{
		PredictedCommand: "ls -la",
	}
	assert.Equal(t, "ls -la", cmd.PredictedCommand)
}

func TestExplainedCommand_Struct(t *testing.T) {
	cmd := explainedCommand{
		Explanation: "Lists all files in the current directory",
	}
	assert.Equal(t, "Lists all files in the current directory", cmd.Explanation)
}

func TestCompletionCandidates_Struct(t *testing.T) {
	candidates := CompletionCandidates{
		Candidates: []string{"ls", "ls -la", "ls -lh"},
	}
	assert.Len(t, candidates.Candidates, 3)
	assert.Equal(t, "ls", candidates.Candidates[0])
}
