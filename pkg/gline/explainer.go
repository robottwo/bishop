package gline

import "context"

type Explainer interface {
	Explain(ctx context.Context, input string) (string, error)
}

type NoopExplainer struct{}

func (e *NoopExplainer) Explain(ctx context.Context, input string) (string, error) {
	return "", nil
}
