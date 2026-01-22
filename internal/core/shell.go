package core

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/robottwo/bishop/internal/agent"
	"github.com/robottwo/bishop/internal/analytics"
	"github.com/robottwo/bishop/internal/bash"
	"github.com/robottwo/bishop/internal/coach"
	"github.com/robottwo/bishop/internal/completion"
	"github.com/robottwo/bishop/internal/config"
	"github.com/robottwo/bishop/internal/environment"
	"github.com/robottwo/bishop/internal/history"
	"github.com/robottwo/bishop/internal/idle"
	"github.com/robottwo/bishop/internal/predict"
	"github.com/robottwo/bishop/internal/rag"
	"github.com/robottwo/bishop/internal/rag/retrievers"
	"github.com/robottwo/bishop/internal/styles"
	"github.com/robottwo/bishop/internal/subagent"
	"github.com/robottwo/bishop/internal/termtitle"
	"github.com/robottwo/bishop/pkg/gline"
	"github.com/robottwo/bishop/pkg/shellinput"
	"go.uber.org/zap"
	"golang.org/x/term"
	"mvdan.cc/sh/v3/interp"
	"mvdan.cc/sh/v3/syntax"
)

func RunInteractiveShell(
	ctx context.Context,
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	analyticsManager *analytics.AnalyticsManager,
	completionManager *completion.CompletionManager,
	coachManager *coach.CoachManager,
	logger *zap.Logger,
	stderrCapturer *StderrCapturer,
) error {
	// Generate session ID
	sessionID := uuid.New().String()

	state := &ShellState{}
	contextProvider := &rag.ContextProvider{
		Logger: logger,
		Retrievers: []rag.ContextRetriever{
			retrievers.SystemInfoContextRetriever{Runner: runner},
			retrievers.WorkingDirectoryContextRetriever{Runner: runner},
			retrievers.GitStatusContextRetriever{Runner: runner, Logger: logger},
			retrievers.ConciseHistoryContextRetriever{Runner: runner, Logger: logger, HistoryManager: historyManager},
			retrievers.VerboseHistoryContextRetriever{Runner: runner, Logger: logger, HistoryManager: historyManager},
		},
	}
	predictor := &predict.PredictRouter{
		PrefixPredictor:    predict.NewLLMPrefixPredictor(runner, historyManager, logger),
		NullStatePredictor: predict.NewLLMNullStatePredictor(runner, logger),
	}
	explainer := predict.NewLLMExplainer(runner, logger)
	agent := agent.NewAgent(runner, historyManager, logger, sessionID)

	// Set up subagent integration
	subagentIntegration := subagent.NewSubagentIntegration(runner, historyManager, logger, sessionID)

	// Set up completion
	completionProvider := completion.NewShellCompletionProvider(completionManager, runner)
	completionProvider.SetSubagentProvider(subagentIntegration.GetCompletionProvider())

	// Set up idle summary generator
	idleSummaryGenerator := idle.NewSummaryGenerator(runner, historyManager, logger)

	// Set up terminal title manager
	termTitleManager := termtitle.NewManager(runner, logger)

	chanSIGINT := make(chan os.Signal, 1)
	signal.Notify(chanSIGINT, os.Interrupt)

	go func() {
		for {
			// ignore SIGINT
			<-chanSIGINT
		}
	}()

	// Initialize cached prompt before entering the loop
	cachedPrompt := environment.GetPrompt(context.Background(), runner, logger)
	logger.Debug("initial prompt cached", zap.String("prompt", cachedPrompt))

	for {
		ragContext := contextProvider.GetContext()
		logger.Debug("context updated", zap.Any("context", ragContext))

		predictor.UpdateContext(ragContext)
		explainer.UpdateContext(ragContext)
		agent.UpdateContext(ragContext)

		// Fetch recent entries for standard history (Up/Down) - scoped to current directory for now, or generally recent
		// Note: GetRecentEntries reverses the list (oldest first) so standard history navigation works correctly
		historySize := environment.GetHistorySize(runner, logger)
		historyEntries, err := historyManager.GetRecentEntries(environment.GetPwd(runner), historySize)
		if err != nil {
			logger.Warn("error getting recent history entries", zap.Error(err))
			historyEntries = []history.HistoryEntry{}
		}

		historyCommands := make([]string, len(historyEntries))
		for i := len(historyEntries) - 1; i >= 0; i-- {
			historyCommands[len(historyEntries)-1-i] = historyEntries[i].Command
		}

		// Fetch all entries for rich search (Ctrl+R)
		allHistoryEntries, err := historyManager.GetAllEntries()
		if err != nil {
			logger.Warn("error getting all history entries", zap.Error(err))
			allHistoryEntries = []history.HistoryEntry{}
		}

		richHistory := make([]shellinput.HistoryItem, len(allHistoryEntries))
		for i, entry := range allHistoryEntries {
			richHistory[i] = shellinput.HistoryItem{
				Command:   entry.Command,
				Directory: entry.Directory,
				Timestamp: entry.CreatedAt,
				SessionID: entry.SessionID,
			}
		}

		// Read input
		options := gline.NewOptions()
		options.AssistantHeight = environment.GetAssistantHeight(runner, logger)
		options.CompletionProvider = completionProvider
		options.RichHistory = richHistory
		options.CurrentDirectory = environment.GetPwd(runner)
		options.CurrentSessionID = sessionID

		// Populate context for border status
		options.User = environment.GetUser(runner)
		options.Host, _ = os.Hostname()

		// Configure idle summary
		idleTimeout := environment.GetIdleSummaryTimeout(runner, logger)
		options.IdleSummaryTimeout = idleTimeout
		if idleTimeout > 0 {
			options.IdleSummaryGenerator = idleSummaryGenerator.GenerateSummary
		}

		// Configure async prompt generation (follows IdleSummaryGenerator pattern above)
		options.PromptGenerator = func(ctx context.Context) string {
			return environment.GetPrompt(ctx, runner, logger)
		}

		// Get coach startup content for the Assistant Box
		var coachContent string
		if coachManager != nil {
			if content := coachManager.GetDisplayContent(); content != nil {
				coachContent = content.Icon + " " + content.Title
				if content.Content != "" {
					coachContent += "\n" + content.Content
				}
				if content.Action != "" {
					coachContent += "\n" + content.Action
				}
			}
		}

		line, newPrompt, err := gline.Gline(cachedPrompt, historyCommands, coachContent, predictor, explainer, analyticsManager, logger, options)

		logger.Debug("received command", zap.String("line", line))

		if err != nil {
			if err == gline.ErrInterrupted {
				// User pressed Ctrl+C, restart loop with fresh prompt
				logger.Debug("input interrupted by user")
				// Store the returned prompt for next iteration
				cachedPrompt = newPrompt
				continue
			}
			logger.Error("error reading input through gline", zap.Error(err))
			return err
		}

		// Store the returned prompt for next iteration
		cachedPrompt = newPrompt

		// Handle agent chat and macros
		if strings.HasPrefix(line, "#") {
			chatMessage := strings.TrimSpace(line[1:])

			// Handle agent controls
			if strings.HasPrefix(chatMessage, "!") {
				control := strings.TrimSpace(strings.TrimPrefix(chatMessage, "!"))

				// Try subagent controls first
				if subagentIntegration.HandleAgentControl(control) {
					continue
				}

				// Handle built-in agent controls
				switch control {
				case "help":
					printHelp()
					continue
				case "fix":
					// Alias for #? - trigger magic fix
					if state.LastExitCode == 0 {
						fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Last command succeeded.\n") + gline.RESET_CURSOR_COLUMN)
						continue
					}
					// Fall through to magic fix handling by setting chatMessage
					chatMessage = "?"
				case "new":
					agent.ResetChat()
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Chat session reset.\n") + gline.RESET_CURSOR_COLUMN)
					continue
				case "tokens":
					agent.PrintTokenStats()
					continue
				case "config":
					if err := config.RunConfigUI(runner); err != nil {
						logger.Error("error running config UI", zap.Error(err))
						fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("bish: Error running config: "+err.Error()+"\n") + gline.RESET_CURSOR_COLUMN)
					}
					// Sync any gsh variables that were changed in the config UI
					environment.SyncVariablesToEnv(runner)
					continue
				default:
					// Handle coach command with subcommands
					if strings.HasPrefix(control, "coach") {
						if coachManager == nil {
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("bish: Coach not initialized\n") + gline.RESET_CURSOR_COLUMN)
							continue
						}

						// Parse subcommand (e.g., "coach tips" -> "tips")
						coachArgs := strings.TrimSpace(strings.TrimPrefix(control, "coach"))

						switch coachArgs {
						case "", "dashboard":
							fmt.Print(coachManager.RenderDashboard())
						case "stats":
							fmt.Print(coachManager.RenderStats())
						case "achievements":
							fmt.Print(coachManager.RenderAchievements())
						case "challenges":
							fmt.Print(coachManager.RenderChallenges())
						case "tips":
							fmt.Print(coachManager.RenderAllTips())
						case "reset-tips":
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("Resetting tips and generating new ones from your history...\nThis may take a moment.\n\n") + gline.RESET_CURSOR_COLUMN)
							result := coachManager.ResetAndRegenerateTips()
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(result+"\n") + gline.RESET_CURSOR_COLUMN)
						default:
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Unknown coach command: "+coachArgs+"\n") + gline.RESET_CURSOR_COLUMN)
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("Available: #!coach [stats|achievements|challenges|tips|reset-tips]\n") + gline.RESET_CURSOR_COLUMN)
						}
						continue
					}
					logger.Warn("unknown agent control", zap.String("control", control))
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Unknown agent control: "+control+"\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}
			}

			// Handle magic fix
			if chatMessage == "?" {
				if state.LastExitCode == 0 {
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Last command succeeded.\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}

				prompt := fmt.Sprintf("The command `%s` failed with exit code %d.\nThe stderr output was:\n%s\n\nExplain why it failed and suggest a fix. Do not execute the fix yet. Provide the fixed command in a markdown code block.", state.LastCommand, state.LastExitCode, state.LastStderr)

				chatChannel, err := agent.Chat(prompt)
				if err != nil {
					logger.Error("error chatting with agent", zap.Error(err))
					continue
				}

				var fullResponse strings.Builder
				for message := range chatChannel {
					fullResponse.WriteString(message)
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: "+message+"\n") + gline.RESET_CURSOR_COLUMN)
				}

				// Display token usage summary
				if tokenSummary := agent.GetTokenSummary(); tokenSummary != "" {
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(tokenSummary+"\n") + gline.RESET_CURSOR_COLUMN)
				}

				// Extract code block
				responseStr := fullResponse.String()
				codeBlockRegex := regexp.MustCompile("(?s)```(?:bash|sh|zsh)?\\s+(.*?)\\s+```")
				matches := codeBlockRegex.FindAllStringSubmatch(responseStr, -1)

				var fixedCmd string
				if len(matches) > 0 {
					fixedCmd = strings.TrimSpace(matches[len(matches)-1][1])
				}

				if fixedCmd != "" {
					defaultToYes := environment.GetDefaultToYes(runner)

					// Loop to allow editing before execution
				magicFixLoop:
					for {
						promptText := "Run this fix? [y/N/e/i] "
						if defaultToYes {
							promptText = "Run this fix? [Y/n/e/i] "
						}

						fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("\nCommand: "+fixedCmd+"\n") + gline.RESET_CURSOR_COLUMN)
						fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(promptText) + gline.RESET_CURSOR_COLUMN)

						// Read single key in raw mode (terminal state restored via defer)
						char, err := readSingleKey(logger)
						if err != nil {
							logger.Error("failed to read key", zap.Error(err))
							break magicFixLoop
						}
						// Echo the character and newline
						if char == '\r' || char == '\n' {
							fmt.Println()
						} else {
							fmt.Printf("%c\n", char)
						}

						// Handle 'e' - edit in external editor
						if char == 'e' || char == 'E' {
							editedCmd, err := openInEditor(fixedCmd)
							if err != nil {
								logger.Error("failed to open editor", zap.Error(err))
								fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("bish: Failed to open editor: "+err.Error()+"\n") + gline.RESET_CURSOR_COLUMN)
								continue
							}
							if editedCmd == "" {
								fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Edit cancelled (empty command)\n") + gline.RESET_CURSOR_COLUMN)
								break magicFixLoop
							}
							fixedCmd = editedCmd
							continue // Show the updated command and prompt again
						}

						// Handle 'i' - insert into prompt for inline editing
						if char == 'i' || char == 'I' {
							fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Edit the command and press Enter to run:\n") + gline.RESET_CURSOR_COLUMN)

							// Create options with the fix command pre-filled
							editOptions := gline.NewOptions()
							editOptions.AssistantHeight = environment.GetAssistantHeight(runner, logger)
							editOptions.CompletionProvider = completionProvider
							editOptions.RichHistory = richHistory
							editOptions.CurrentDirectory = environment.GetPwd(runner)
							editOptions.CurrentSessionID = sessionID
							editOptions.User = environment.GetUser(runner)
							editOptions.Host, _ = os.Hostname()
							editOptions.InitialValue = fixedCmd

							shellPrompt := environment.GetPrompt(context.Background(), runner, logger)
							editedLine, _, editErr := gline.Gline(shellPrompt, historyCommands, "", predictor, explainer, analyticsManager, logger, editOptions)
							if editErr != nil {
								if editErr == gline.ErrInterrupted {
									fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Edit cancelled\n") + gline.RESET_CURSOR_COLUMN)
									break magicFixLoop
								}
								logger.Error("error reading edited command", zap.Error(editErr))
								break magicFixLoop
							}
							if strings.TrimSpace(editedLine) == "" {
								fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Edit cancelled (empty command)\n") + gline.RESET_CURSOR_COLUMN)
								break magicFixLoop
							}
							fixedCmd = editedLine
							// Execute the edited command directly
							fmt.Println()
							shouldExit, err := executeCommand(ctx, fixedCmd, historyManager, coachManager, runner, logger, state, stderrCapturer, sessionID)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
							}
							termTitleManager.RecordCommand(fixedCmd)
							environment.SyncVariablesToEnv(runner)
							if shouldExit {
								logger.Debug("exiting...")
								return nil
							}
							break magicFixLoop
						}

						// Determine if confirmed based on default setting
						confirmed := char == 'y' || char == 'Y'
						if defaultToYes && (char == '\r' || char == '\n') {
							confirmed = true
						}

						if confirmed {
							fmt.Println()
							shouldExit, err := executeCommand(ctx, fixedCmd, historyManager, coachManager, runner, logger, state, stderrCapturer, sessionID)
							if err != nil {
								fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
							}

							// Record command for terminal title updates
							termTitleManager.RecordCommand(fixedCmd)

							// Sync any gsh variables that might have been changed during command execution
							environment.SyncVariablesToEnv(runner)

							if shouldExit {
								logger.Debug("exiting...")
								return nil
							}
							break magicFixLoop
						}

						// Any other key (n, N, Escape, etc.) cancels
						break magicFixLoop
					}
				}
				continue
			}

			// Handle macros
			if strings.HasPrefix(chatMessage, "/") {
				macroName := strings.TrimSpace(strings.TrimPrefix(chatMessage, "/"))
				macros := environment.GetAgentMacros(runner, logger)
				if message, ok := macros[macroName]; ok {
					chatMessage = message
				} else {
					logger.Warn("macro not found", zap.String("macro", macroName))
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: Macro not found: "+macroName+"\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}
			}

			// Check for subagent commands first
			handled, chatChannel, subagent, err := subagentIntegration.HandleCommand(chatMessage)
			if handled {
				if err != nil {
					logger.Error("error with subagent command", zap.Error(err))
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.ERROR("bish: "+err.Error()+"\n") + gline.RESET_CURSOR_COLUMN)
					continue
				}

				// Print notification: "Asking <subagent_name> to assist with this task"
				fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(fmt.Sprintf("Asking %s to assist with this task\n", subagent.Name)) + gline.RESET_CURSOR_COLUMN)

				// Handle subagent response with subagent identification
				for message := range chatChannel {
					prefix := fmt.Sprintf("bish [%s]: ", subagent.Name)
					fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(prefix+message+"\n") + gline.RESET_CURSOR_COLUMN)
				}
				continue
			}

			// Fall back to regular agent chat
			chatChannel, err = agent.Chat(chatMessage)
			if err != nil {
				logger.Error("error chatting with agent", zap.Error(err))
				continue
			}

			for message := range chatChannel {
				fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("bish: "+message+"\n") + gline.RESET_CURSOR_COLUMN)
			}

			// Display token usage summary
			if tokenSummary := agent.GetTokenSummary(); tokenSummary != "" {
				fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(tokenSummary+"\n") + gline.RESET_CURSOR_COLUMN)
			}

			continue
		}

		// Handle empty input
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Note: Autocd is now handled by the AutocdExecHandler in the command execution chain
		// This allows builtins and commands to take precedence naturally

		// Execute the command
		shouldExit, err := executeCommand(ctx, line, historyManager, coachManager, runner, logger, state, stderrCapturer, sessionID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error executing command: %v\n", err)
		}

		// Show helpful hint when command fails (only once per session)
		if state.LastExitCode != 0 && !state.FixHintShown {
			state.FixHintShown = true
			fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE("Tip: Use #? or #!fix to ask the AI to help fix this error\n") + gline.RESET_CURSOR_COLUMN)
		}

		// Record command for terminal title updates
		termTitleManager.RecordCommand(line)

		// Sync any gsh variables that might have been changed during command execution
		environment.SyncVariablesToEnv(runner)

		if shouldExit {
			logger.Debug("exiting...")
			break
		}
	}

	return nil
}

// readSingleKey reads a single key from stdin in raw mode.
// It ensures the terminal state is always restored, even on panic.
func readSingleKey(logger *zap.Logger) (byte, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return 0, err
	}
	defer func() {
		if restoreErr := term.Restore(fd, oldState); restoreErr != nil {
			logger.Error("failed to restore terminal state", zap.Error(restoreErr))
		}
	}()

	var buf [1]byte
	_, err = os.Stdin.Read(buf[:])
	if err != nil {
		return 0, err
	}
	return buf[0], nil
}

// openInEditor opens the given command in an external editor and returns the edited result.
// It uses $EDITOR, $VISUAL, or falls back to vi/vim/nano.
func openInEditor(command string) (string, error) {
	// Determine editor to use
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		// Try common editors
		for _, e := range []string{"vi", "vim", "nano"} {
			if _, err := exec.LookPath(e); err == nil {
				editor = e
				break
			}
		}
	}
	if editor == "" {
		return "", fmt.Errorf("no editor found (set $EDITOR)")
	}

	// Create temp file with the command
	tmpFile, err := os.CreateTemp("", "bish-fix-*.sh")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.WriteString(command); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("failed to write to temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}

	// Run the editor
	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor exited with error: %w", err)
	}

	// Read the edited content
	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("failed to read edited file: %w", err)
	}

	// Return trimmed content (remove trailing newlines but preserve internal structure)
	return strings.TrimSpace(string(content)), nil
}

func executeCommand(ctx context.Context, input string, historyManager *history.HistoryManager, coachManager *coach.CoachManager, runner *interp.Runner, logger *zap.Logger, state *ShellState, stderrCapturer *StderrCapturer, sessionID string) (bool, error) {
	// History expansion
	expandedInput, expanded := expandHistory(input, historyManager)
	if expanded {
		input = expandedInput
		fmt.Fprintln(os.Stderr, input)
	}

	// Pre-process input to transform typeset/declare -f/-F/-p commands to bish_typeset
	logger.Debug("preprocessing input", zap.String("original_input", input), zap.Int("input_length", len(input)))

	// Validate input before preprocessing
	if input == "" {
		logger.Warn("empty input received for preprocessing")
		return false, nil
	}

	// Add timeout protection for preprocessing
	preprocessStart := time.Now()
	logger.Debug("calling bash.PreprocessTypesetCommands", zap.String("input", input))
	processedInput := bash.PreprocessTypesetCommands(input)
	logger.Debug("bash.PreprocessTypesetCommands completed", zap.String("output", processedInput))
	preprocessDuration := time.Since(preprocessStart)

	logger.Debug("preprocessing completed",
		zap.String("processed_input", processedInput),
		zap.Int("processed_length", len(processedInput)),
		zap.Duration("preprocess_duration", preprocessDuration),
		zap.Bool("input_changed", input != processedInput))

	// Check if preprocessing took too long (potential resource exhaustion)
	if preprocessDuration > 100*time.Millisecond {
		logger.Warn("preprocessing took unusually long",
			zap.Duration("duration", preprocessDuration),
			zap.Int("input_length", len(input)))
	}

	input = processedInput

	var prog *syntax.Stmt
	err := syntax.NewParser().Stmts(strings.NewReader(input), func(stmt *syntax.Stmt) bool {
		prog = stmt
		return false
	})
	if prog == nil {
		logger.Error("invalid command", zap.String("command", input))
		return false, nil
	}
	if err != nil {
		logger.Error("error parsing command", zap.String("command", input), zap.Error(err))
		return false, err
	}

	historyEntry, _ := historyManager.StartCommand(input, environment.GetPwd(runner), sessionID)

	state.LastCommand = input
	if stderrCapturer != nil {
		stderrCapturer.StartCapture()
	}

	startTime := time.Now()
	err = runner.Run(ctx, prog)
	exited := runner.Exited()

	if stderrCapturer != nil {
		state.LastStderr = stderrCapturer.StopCapture()
	}

	endTime := time.Now()

	durationMs := endTime.Sub(startTime).Milliseconds()
	_, _, _ = bash.RunBashCommand(ctx, runner, fmt.Sprintf("BISH_LAST_COMMAND_DURATION_MS=%d", durationMs))

	var exitCode int
	if err != nil {
		status, ok := interp.IsExitStatus(err)
		if !ok {
			exitCode = -1
		} else {
			exitCode = int(status)
		}
	} else {
		exitCode = 0
	}

	state.LastExitCode = exitCode

	_, _ = historyManager.FinishCommand(historyEntry, exitCode)
	_, _, _ = bash.RunBashCommand(ctx, runner, fmt.Sprintf("BISH_LAST_COMMAND_EXIT_CODE=%d", exitCode))

	// Record command for coach gamification
	if coachManager != nil {
		coachManager.RecordCommand(input, exitCode, durationMs)
	}

	return exited, nil
}

func expandHistory(input string, historyManager *history.HistoryManager) (string, bool) {
	// Quick check
	if !strings.Contains(input, "!") {
		return input, false
	}

	entries, err := historyManager.GetAllEntries()
	if err != nil || len(entries) == 0 {
		return input, false
	}
	lastEntry := entries[0]
	lastCmd := lastEntry.Command

	// Get last argument
	lastArg := shellinput.GetLastArgument(lastCmd)

	var sb strings.Builder
	expanded := false
	inSingleQuote := false

	runes := []rune(input)
	for i := 0; i < len(runes); i++ {
		r := runes[i]

		if r == '\'' {
			inSingleQuote = !inSingleQuote
			sb.WriteRune(r)
			continue
		}

		if inSingleQuote {
			sb.WriteRune(r)
			continue
		}

		if r == '\\' {
			sb.WriteRune(r)
			if i+1 < len(runes) {
				sb.WriteRune(runes[i+1])
				i++
			}
			continue
		}

		// Check for !!
		if r == '!' && i+1 < len(runes) && runes[i+1] == '!' {
			sb.WriteString(lastCmd)
			expanded = true
			i++ // Skip next !
			continue
		}

		// Check for !$
		if r == '!' && i+1 < len(runes) && runes[i+1] == '$' {
			sb.WriteString(lastArg)
			expanded = true
			i++ // Skip next $
			continue
		}

		sb.WriteRune(r)
	}

	return sb.String(), expanded
}

// printHelp displays help information about Bishop shell commands
func printHelp() {
	helpText := `
Bishop Shell - AI-Powered Command Line

AGENT COMMANDS
  # <message>       Chat with the AI agent
  #? or #!fix       Ask AI to explain and fix the last failed command
  #/<macro>         Invoke a predefined agent macro

AGENT CONTROLS
  #!help            Show this help message
  #!new             Reset the current chat session
  #!tokens          Display token usage statistics
  #!config          Open interactive configuration menu
  #!coach           Open the coaching dashboard
    #!coach stats        View your command statistics
    #!coach achievements View your achievements
    #!coach challenges   View active challenges
    #!coach tips         View personalized tips
    #!coach reset-tips   Regenerate tips from history

SUBAGENTS
  ##<name> <prompt> Chat with a specific subagent (e.g., ##git commit this)
  ## <prompt>       Auto-select best subagent for your prompt
  #:<mode> <prompt> Roo Code style invocation
  Type '##' and press Tab to see available subagents

MAGIC FIX OPTIONS
  y/Y               Run the suggested fix
  n/N               Cancel (any other key also cancels)
  e/E               Edit the fix in your $EDITOR
  i/I               Insert the fix into the prompt to edit inline

HISTORY EXPANSION
  !!                Repeat the last command
  !$                Use the last argument from previous command

KEYBOARD SHORTCUTS
  Ctrl+R            Search command history
  Ctrl+L            Clear screen
  Ctrl+C            Cancel current input
  Ctrl+D            Exit shell (on empty line)
  Tab               Autocomplete commands/paths

For more information, see the documentation at:
  https://github.com/robottwo/bishop
`
	fmt.Print(gline.RESET_CURSOR_COLUMN + styles.AGENT_MESSAGE(helpText) + gline.RESET_CURSOR_COLUMN)
}
