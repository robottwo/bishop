package predict

import (
	"context"
	"strings"
)

type PredictRouter struct {
	PrefixPredictor    *LLMPrefixPredictor
	NullStatePredictor *LLMNullStatePredictor
}

func (p *PredictRouter) UpdateContext(ctx *map[string]string) {
	if p.PrefixPredictor != nil {
		p.PrefixPredictor.UpdateContext(ctx)
	}

	if p.NullStatePredictor != nil {
		p.NullStatePredictor.UpdateContext(ctx)
	}
}

func (p *PredictRouter) Predict(ctx context.Context, input string) (string, string, error) {
	// Skip LLM prediction when input is blank (empty or whitespace only)
	if strings.TrimSpace(input) == "" {
		return "", "", nil
	}
	return p.PrefixPredictor.Predict(ctx, input)
}
