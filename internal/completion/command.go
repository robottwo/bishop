package completion

import (
	"context"
	"fmt"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// For testing purposes
var printf = fmt.Printf

// completeUsage provides the usage summary for the complete command
const completeUsage = `Usage: complete [-pr] [-W wordlist] [-F function] [-C command] name
       complete -p [name]
       complete -r [name]

Options:
  -p          Print existing completion specifications
  -r          Remove completion specification for name
  -W wordlist Use wordlist (space-separated words) for completion
  -F function Call function for generating completions
  -C command  Execute command for generating completions
  -h, --help  Show this help message

Examples:
  complete -W "start stop restart" service
  complete -F _git_completion git
  complete -p git
  complete -r git`

// usageError wraps an error with a usage hint
type usageError struct {
	msg string
}

func (e *usageError) Error() string {
	return fmt.Sprintf("%s\n\nRun 'complete -h' for usage information.", e.msg)
}

// newUsageError creates a new error that hints at -h/--help
func newUsageError(format string, args ...any) error {
	return &usageError{msg: fmt.Sprintf(format, args...)}
}

// NewCompleteCommandHandler creates a new ExecHandler for the complete command
func NewCompleteCommandHandler(completionManager *CompletionManager) func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 || args[0] != "complete" {
				return next(ctx, args)
			}

			// Handle the complete command
			return handleCompleteCommand(completionManager, args[1:])
		}
	}
}

func handleCompleteCommand(manager *CompletionManager, args []string) error {
	if len(args) == 0 {
		// No arguments - print all completion specs
		return printCompletionSpecs(manager, "")
	}

	// Parse options
	var (
		printMode  bool
		removeMode bool
		wordList   string
		function   string
		commandCmd string
		command    string
	)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "-h", "--help":
			_, _ = printf("%s\n", completeUsage)
			return nil
		case "-p":
			printMode = true
		case "-r":
			removeMode = true
		case "-W":
			if i+1 >= len(args) {
				return newUsageError("option -W requires a word list")
			}
			i++
			wordList = args[i]
		case "-F":
			if i+1 >= len(args) {
				return newUsageError("option -F requires a function name")
			}
			i++
			function = args[i]
		case "-C":
			if i+1 >= len(args) {
				return newUsageError("option -C requires a command")
			}
			i++
			commandCmd = args[i]
		default:
			if !strings.HasPrefix(arg, "-") {
				command = arg
				break
			}
			return newUsageError("unknown option: %s", arg)
		}
	}

	if command == "" && !printMode {
		return newUsageError("no command specified")
	}

	// Handle different modes
	if printMode {
		return printCompletionSpecs(manager, command)
	}

	if removeMode {
		manager.RemoveSpec(command)
		return nil
	}

	if wordList != "" {
		manager.AddSpec(CompletionSpec{
			Command: command,
			Type:    WordListCompletion,
			Value:   wordList,
		})
		return nil
	}

	if function != "" {
		manager.AddSpec(CompletionSpec{
			Command: command,
			Type:    FunctionCompletion,
			Value:   function,
		})
		return nil
	}

	if commandCmd != "" {
		manager.AddSpec(CompletionSpec{
			Command: command,
			Type:    CommandCompletion,
			Value:   commandCmd,
		})
		return nil
	}

	return newUsageError("missing completion action: use -W, -F, or -C")
}

func printCompletionSpecs(manager *CompletionManager, command string) error {
	if command != "" {
		// Print specific command
		if spec, ok := manager.GetSpec(command); ok {
			printCompletionSpec(spec)
		}
		return nil
	}

	// Print all specs
	for _, spec := range manager.ListSpecs() {
		printCompletionSpec(spec)
	}
	return nil
}

func printCompletionSpec(spec CompletionSpec) {
	switch spec.Type {
	case WordListCompletion:
		_, _ = printf("complete -W %q %s\n", spec.Value, spec.Command)
	case FunctionCompletion:
		_, _ = printf("complete -F %s %s\n", spec.Value, spec.Command)
	case CommandCompletion:
		_, _ = printf("complete -C %q %s\n", spec.Value, spec.Command)
	}
}
