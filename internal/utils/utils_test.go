package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestComposeContextText(t *testing.T) {
	// Mock logger
	logger, _ := zap.NewDevelopment(zap.IncreaseLevel(zapcore.WarnLevel))

	context := map[string]string{
		"type1": "This is type 1",
		"type2": "This is type 2",
	}

	// Test with valid keys
	result := ComposeContextText(&context, []string{"type1", "type2"}, logger)
	assert.Equal(t, "\nThis is type 1\n\nThis is type 2\n", result, "Should concatenate values for valid keys")

	// Test with a missing key
	result = ComposeContextText(&context, []string{"type1", "type3"}, logger)
	assert.Equal(t, "\nThis is type 1\n", result, "Should skip missing keys and log a warning")

	// Test with empty contextTypes
	result = ComposeContextText(&context, []string{}, logger)
	assert.Equal(t, "", result, "Should return empty string for empty contextTypes")

	// Test with nil context
	result = ComposeContextText(nil, []string{"type1"}, logger)
	assert.Equal(t, "", result, "Should return empty string for nil context")
}

func TestGenerateJsonSchema(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	schema := GenerateJsonSchema(TestStruct{})
	require.NotNil(t, schema)

	// Verify schema can be marshaled
	jsonBytes, err := schema.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, jsonBytes)
}

func TestGenerateJsonSchema_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}

	schema := GenerateJsonSchema(EmptyStruct{})
	require.NotNil(t, schema)
}

func TestGenerateJsonSchema_NestedStruct(t *testing.T) {
	type Inner struct {
		ID int `json:"id"`
	}
	type Outer struct {
		Inner Inner  `json:"inner"`
		Name  string `json:"name"`
	}

	schema := GenerateJsonSchema(Outer{})
	require.NotNil(t, schema)
}

func TestLLMModelType_Constants(t *testing.T) {
	assert.Equal(t, LLMModelType("FAST"), FastModel)
	assert.Equal(t, LLMModelType("SLOW"), SlowModel)
}

func TestLLMModelConfig_Struct(t *testing.T) {
	temp := 0.7
	parallel := true
	config := LLMModelConfig{
		ModelId:           "gpt-4",
		Temperature:       &temp,
		ParallelToolCalls: &parallel,
	}

	assert.Equal(t, "gpt-4", config.ModelId)
	require.NotNil(t, config.Temperature)
	assert.Equal(t, 0.7, *config.Temperature)
	require.NotNil(t, config.ParallelToolCalls)
	assert.True(t, *config.ParallelToolCalls)
}

func TestLLMModelConfig_NilOptionals(t *testing.T) {
	config := LLMModelConfig{
		ModelId: "claude-3",
	}

	assert.Equal(t, "claude-3", config.ModelId)
	assert.Nil(t, config.Temperature)
	assert.Nil(t, config.ParallelToolCalls)
}