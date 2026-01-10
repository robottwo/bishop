package tools

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/robottwo/bishop/internal/environment"
	"github.com/robottwo/bishop/internal/styles"
	"github.com/robottwo/bishop/pkg/gline"
	"go.uber.org/zap"
	"mvdan.cc/sh/v3/interp"
)

func failedToolResponse(errorMessage string) string {
	return fmt.Sprintf("<bish_tool_call_error>%s</bish_tool_call_error>", errorMessage)
}

func printToolMessage(message string) {
	// Suppress output during tests
	if flag.Lookup("test.v") != nil {
		return
	}
	fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_QUESTION(message) + "\n")
}

func printToolPath(path string) {
	// Suppress output during tests
	if flag.Lookup("test.v") != nil {
		return
	}
	fmt.Print(gline.RESET_CURSOR_COLUMN + path + "\n")
}

func printDiff(diff string) {
	// Suppress output during tests
	if flag.Lookup("test.v") != nil {
		return
	}
	fmt.Print(gline.RESET_CURSOR_COLUMN + diff + "\n" + gline.RESET_CURSOR_COLUMN)
}

func printCommandPrompt(prompt string) {
	// Suppress output during tests
	if flag.Lookup("test.v") != nil {
		return
	}
	fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(prompt) + "\n")
}

// defaultUserConfirmation is the default implementation that calls gline.Gline
var defaultUserConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string, showManage bool) string {
	defaultToYes := false
	if runner != nil {
		defaultToYes = environment.GetDefaultToYes(runner)
	}

	// Build the prompt with styled components using style functions
	// Format: (y)es  [N]o - default  (m)anage menu  [or type feedback]: (with manage)
	// Format: (y)es  [N]o - default  [or type feedback]: (without manage)
	var promptSuffix string
	if defaultToYes {
		// When default is yes: [Y]es - default  (n)o  (m)anage menu  [or type feedback]:
		yesOption := styles.PROMPT_DEFAULT("[Y]es - default")
		noOption := styles.PROMPT_OPTION("(n)o")
		hint := styles.PROMPT_HINT("[or type feedback]")
		if showManage {
			manageOption := styles.PROMPT_OPTION("(m)anage menu")
			promptSuffix = " " + yesOption + "  " + noOption + "  " + manageOption + "  " + hint + ": "
		} else {
			promptSuffix = " " + yesOption + "  " + noOption + "  " + hint + ": "
		}
	} else {
		// When default is no: (y)es  [N]o - default  (m)anage menu  [or type feedback]:
		yesOption := styles.PROMPT_OPTION("(y)es")
		noOption := styles.PROMPT_DEFAULT("[N]o - default")
		hint := styles.PROMPT_HINT("[or type feedback]")
		if showManage {
			manageOption := styles.PROMPT_OPTION("(m)anage menu")
			promptSuffix = " " + yesOption + "  " + noOption + "  " + manageOption + "  " + hint + ": "
		} else {
			promptSuffix = " " + yesOption + "  " + noOption + "  " + hint + ": "
		}
	}
	prompt := styles.AGENT_QUESTION(question) + promptSuffix

	line, err := gline.Gline(prompt, []string{}, explanation, nil, nil, nil, logger, gline.NewOptions())
	if err != nil {
		// Check if the error is specifically from Ctrl+C interruption
		if err == gline.ErrInterrupted {
			logger.Debug("User pressed Ctrl+C, treating as 'n' response")
			return "n"
		}

		// Log the error and return default response based on setting
		logger.Error("gline.Gline returned error during user confirmation",
			zap.Error(err),
			zap.String("question", question))
		if defaultToYes {
			return "y"
		}
		return "n"
	}

	// Handle empty input based on default setting
	if strings.TrimSpace(line) == "" {
		if defaultToYes {
			return "y"
		}
		return "n"
	}

	lowerLine := strings.ToLower(line)

	if lowerLine == "y" || lowerLine == "yes" {
		return "y"
	}

	if lowerLine == "n" || lowerLine == "no" {
		return "n"
	}

	if lowerLine == "m" || lowerLine == "manage" {
		return "m"
	}

	return line
}

// userConfirmation is a wrapper that checks for test mode before calling the real implementation
var userConfirmation = func(logger *zap.Logger, runner *interp.Runner, question string, explanation string, showManage bool) string {
	// Check if we're in test mode and this function hasn't been mocked
	// We detect if it's been mocked by checking if the function pointer has changed
	if flag.Lookup("test.v") != nil {
		// In test mode, return "n" to avoid blocking on gline.Gline
		// Tests that need different behavior should mock this function
		if logger != nil {
			logger.Debug("userConfirmation called in test mode without mock, returning 'n'")
		}
		return "n"
	}

	return defaultUserConfirmation(logger, runner, question, explanation, showManage)
}
