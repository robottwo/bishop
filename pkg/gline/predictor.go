package gline

import "context"

type Predictor interface {
	Predict(ctx context.Context, input string) (string, string, error)
}

type NoopPredictor struct{}

func (p *NoopPredictor) Predict(ctx context.Context, input string) (string, string, error) {
	return "", "", nil
}
