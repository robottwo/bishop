package gline

import (
	"context"

	"github.com/robottwo/bishop/pkg/shellinput"
)

// IdleSummaryGenerator is a function that generates an idle summary
type IdleSummaryGenerator func(ctx context.Context) (string, error)

// PromptGenerator is a function that generates the prompt string
type PromptGenerator func(ctx context.Context) string

type Options struct {
	// Deprecated: use AssistantHeight instead
	MinHeight          int
	AssistantHeight    int
	CompletionProvider shellinput.CompletionProvider
	RichHistory        []shellinput.HistoryItem
	CurrentDirectory   string
	CurrentSessionID   string
	User               string
	Host               string

	// InitialValue is the initial text to populate in the input field.
	// Used for features like editing a suggested fix before execution.
	InitialValue string

	// IdleSummaryTimeout is the number of seconds of idle time before generating a summary.
	// Set to 0 to disable idle summaries.
	IdleSummaryTimeout int
	// IdleSummaryGenerator is called when the user is idle to generate a summary
	IdleSummaryGenerator IdleSummaryGenerator

	// ResourceUpdateInterval is the interval in seconds between resource (CPU/RAM) updates.
	// Higher values reduce energy consumption. Default is 5 seconds.
	// Set to 0 to disable resource monitoring.
	ResourceUpdateInterval int

	// PromptGenerator is called asynchronously to generate the prompt string.
	// If nil, prompt fetching is disabled.
	PromptGenerator PromptGenerator
}

func NewOptions() Options {
	return Options{
		AssistantHeight:        3,
		ResourceUpdateInterval: 5, // 5 seconds default to reduce energy consumption
	}
}
