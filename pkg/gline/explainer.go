package gline

type Explanation struct {
	Text  string
	Error string // concise error message, empty if none
}

type Explainer interface {
	Explain(input string) (*Explanation, error)
}

type NoopExplainer struct{}

func (e *NoopExplainer) Explain(input string) (*Explanation, error) {
	return nil, nil
}
