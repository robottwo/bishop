package predict

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPredictRouter_Predict_SkipsBlankInput(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"whitespace only", "   "},
		{"tabs only", "\t\t"},
		{"mixed whitespace", "  \t  \n  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := &PredictRouter{
				PrefixPredictor: nil, // Will panic if called
			}

			prediction, prompt, err := router.Predict(context.Background(), tt.input)

			assert.NoError(t, err)
			assert.Empty(t, prediction)
			assert.Empty(t, prompt)
		})
	}
}

func TestPredictRouter_UpdateContext_NilPredictors(t *testing.T) {
	// Should not panic when predictors are nil
	router := &PredictRouter{
		PrefixPredictor:    nil,
		NullStatePredictor: nil,
	}

	ctx := map[string]string{"key": "value"}

	// Should not panic
	assert.NotPanics(t, func() {
		router.UpdateContext(&ctx)
	})
}
