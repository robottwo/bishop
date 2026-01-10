package styles

import (
	"os"

	"github.com/muesli/termenv"
)

var (
	stdout = termenv.NewOutput(os.Stdout)

	ERROR = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("9")).
			String()
	}
	AGENT_MESSAGE = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("12")).
			String()
	}
	AGENT_QUESTION = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("11")).
			Bold().
			String()
	}
	// PROMPT_OPTION styles a regular prompt option (e.g., "(y)es", "(n)o", "(m)anage")
	PROMPT_OPTION = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("11")).
			String()
	}
	// PROMPT_DEFAULT styles the default prompt option with bold emphasis (e.g., "[N]o", "[Y]es")
	PROMPT_DEFAULT = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("11")).
			Bold().
			String()
	}
	// PROMPT_HINT styles hint text with dimmed appearance (e.g., "[or type feedback]")
	PROMPT_HINT = func(s string) string {
		return stdout.String(s).
			Foreground(stdout.Color("244")).
			String()
	}
)
