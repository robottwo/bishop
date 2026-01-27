package main

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/klauspost/compress/zstd"
	"github.com/robottwo/bishop/internal/analytics"
	"github.com/robottwo/bishop/internal/bash"
	"github.com/robottwo/bishop/internal/coach"
	"github.com/robottwo/bishop/internal/completion"
	"github.com/robottwo/bishop/internal/config"
	"github.com/robottwo/bishop/internal/core"
	"github.com/robottwo/bishop/internal/environment"
	"github.com/robottwo/bishop/internal/evaluate"
	"github.com/robottwo/bishop/internal/history"
	"github.com/robottwo/bishop/internal/styles"
	"go.uber.org/zap"
	"golang.org/x/term"
	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

var BUILD_VERSION = "dev"

//go:embed .bishrc.default
var DEFAULT_VARS []byte

var command = flag.String("c", "", "run a command")
var loginShell = flag.Bool("l", false, "run as a login shell")
var rcFile = flag.String("rcfile", "", "use a custom rc file instead of ~/.bishrc")
var strictConfig = flag.Bool("strict-config", false, "fail fast if configuration files contain errors (like bash 'set -e')")

var helpFlag bool
var versionFlag bool

func init() {
	// Register help flags: -h and --help
	flag.BoolVar(&helpFlag, "h", false, "display help information")
	flag.BoolVar(&helpFlag, "help", false, "display help information")

	// Register version flags: -v, -ver, and --version
	flag.BoolVar(&versionFlag, "v", false, "display build version")
	flag.BoolVar(&versionFlag, "ver", false, "display build version")
	flag.BoolVar(&versionFlag, "version", false, "display build version")

	// Register custom zstd sink for compressed logging
	if err := zap.RegisterSink("zstd", newCompressedSink); err != nil {
		panic(fmt.Sprintf("failed to register zstd sink: %v", err))
	}
}

// main is the entry point of the bish shell program.
// It initializes all the core components including:
// - Command-line flag parsing for version (-v), help (-h), and execution modes
// - History manager for command history tracking
// - Analytics manager for usage analytics
// - Completion manager for tab completion
// - Shell interpreter runner with stderr capture
// - Logger for debugging and monitoring
// - Coach manager for AI-powered assistance (optional)
//
// The function supports multiple execution modes:
// 1. Version display: bish -v
// 2. Help display: bish -h
// 3. Command execution: bish -c "command"
// 4. Interactive shell: bish (when stdin is a terminal)
// 5. Script execution: bish script.sh
//
// After initialization, it delegates to the run() function which handles
// the actual execution based on the detected mode and handles exit codes.
func main() {
	flag.Parse()

	if versionFlag {
		fmt.Println(BUILD_VERSION)
		return
	}

	if helpFlag {
		printUsage()
		return
	}

	// Initialize the history manager
	historyManager, err := initializeHistoryManager()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize history manager: %v", err))
	}
	defer func() {
		if err := historyManager.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close history manager: %v\n", err)
		}
	}()

	// Initialize the analytics manager
	analyticsManager, err := initializeAnalyticsManager()
	if err != nil {
		panic(fmt.Sprintf("failed to initialize analytics manager: %v", err))
	}
	defer func() {
		if err := analyticsManager.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close analytics manager: %v\n", err)
		}
	}()

	// Initialize the completion manager
	completionManager := initializeCompletionManager()

	// Initialize the stderr capturer
	stderrCapturer := core.NewStderrCapturer(os.Stderr)

	// Initialize the shell interpreter
	runner, err := initializeRunner(analyticsManager, historyManager, completionManager, stderrCapturer)
	if err != nil {
		panic(err)
	}

	// Register session config override getter so environment package can access config overrides
	environment.SetSessionConfigOverrideGetter(config.GetSessionOverride)

	// Initialize the logger
	logger, err := initializeLogger(runner)
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = logger.Sync() // Flush any buffered log entries
	}()

	analyticsManager.Logger = logger

	logger.Info("-------- new bish session --------", zap.Any("args", os.Args))

	// Initialize the coach manager (uses same database as history)
	coachManager, err := coach.NewCoachManager(historyManager.GetDB(), historyManager, runner, logger)
	if err != nil {
		logger.Warn("failed to initialize coach manager", zap.Error(err))
		// Coach is optional, continue without it
		coachManager = nil
	}

	// Start running
	err = run(runner, historyManager, analyticsManager, completionManager, coachManager, logger, stderrCapturer)

	// Handle exit status
	if code, ok := interp.IsExitStatus(err); ok {
		os.Exit(int(code))
	}

	if err != nil {
		logger.Error("unhandled error", zap.Error(err))
		os.Exit(1)
	}
}

func run(
	runner *interp.Runner,
	historyManager *history.HistoryManager,
	analyticsManager *analytics.AnalyticsManager,
	completionManager *completion.CompletionManager,
	coachManager *coach.CoachManager,
	logger *zap.Logger,
	stderrCapturer *core.StderrCapturer,
) error {
	ctx := context.Background()

	// bish -c "echo hello"
	if *command != "" {
		return bash.RunBashScriptFromReader(ctx, runner, strings.NewReader(*command), "bish")
	}

	// bish
	if flag.NArg() == 0 {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return core.RunInteractiveShell(ctx, runner, historyManager, analyticsManager, completionManager, coachManager, logger, stderrCapturer)
		}

		return bash.RunBashScriptFromReader(ctx, runner, os.Stdin, "bish")
	}

	// bish script.sh
	for _, filePath := range flag.Args() {
		if err := bash.RunBashScriptFromFile(ctx, runner, filePath); err != nil {
			return err
		}
	}

	return nil
}

func printUsage() {
	// Header
	fmt.Println(styles.AGENT_QUESTION("Usage:") + " bish [flags] [script]")
	fmt.Println("\nA modern, POSIX-compatible, Generative Shell.")
	fmt.Println()

	// Flags
	fmt.Println(styles.AGENT_QUESTION("Options:"))

	// We want to group aliases like -h and -help together
	// Map to track which flags we've already printed
	printed := make(map[string]bool)

	flag.VisitAll(func(f *flag.Flag) {
		if printed[f.Name] {
			return
		}

		// Identify aliases based on shared usage strings.
		aliases := []string{f.Name}
		flag.VisitAll(func(p *flag.Flag) {
			if p.Name == f.Name {
				return
			}
			if p.Usage == f.Usage {
				aliases = append(aliases, p.Name)
				printed[p.Name] = true
			}
		})
		printed[f.Name] = true

		// Separate short and long flags
		var shortFlags, longFlags []string
		for _, name := range aliases {
			if len(name) == 1 {
				shortFlags = append(shortFlags, "-"+name)
			} else {
				longFlags = append(longFlags, "-"+name)
			}
		}

		// Construct the flag string: short flags first, then long flags
		flagStr := ""
		if len(shortFlags) > 0 {
			flagStr = strings.Join(shortFlags, ", ")
		}
		if len(longFlags) > 0 {
			if flagStr != "" {
				flagStr += ", "
			}
			flagStr += strings.Join(longFlags, ", ")
		}

		// Check if the flag takes an argument
		argName, usage := flag.UnquoteUsage(f)
		if argName != "" {
			flagStr += " <" + argName + ">"
		}

		fmt.Printf("  %-28s %s\n", flagStr, usage)
	})

	fmt.Println()
	fmt.Println(styles.AGENT_QUESTION("Key Features:"))
	fmt.Printf("  %-28s %s\n", "# <message>", "Chat with the agent")
	fmt.Printf("  %-28s %s\n", "#!<control>", "Agent controls (e.g., #!config, #!new)")
	fmt.Printf("  %-28s %s\n", "#?", "Magic Fix: Analyze and fix the last error")
	fmt.Printf("  %-28s %s\n", "#/<macro>", "Run a chat macro (e.g., #/gitdiff)")
}

// newCompressedSink creates a new compressed sink from a URL.
// The URL path should point to the log file location.
// Implements proper zstd frame continuation by checking if the existing file
// contains valid zstd frames and appending new frames appropriately.
func newCompressedSink(u *url.URL) (zap.Sink, error) {
	filePath := u.Path

	flags := os.O_CREATE | os.O_WRONLY

	fileInfo, err := os.Stat(filePath)
	if err == nil && fileInfo.Size() > 0 {
		if isValidZstdFile(filePath) {
			flags |= os.O_APPEND
		} else {
			flags |= os.O_TRUNC
		}
	}

	file, err := os.OpenFile(filePath, flags, 0644)
	if err != nil {
		return nil, err
	}

	encoder, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedDefault))
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	return &compressedSink{
		file:    file,
		encoder: encoder,
	}, nil
}

// isValidZstdFile checks if a file starts with a valid zstd magic number.
// Returns false if file doesn't exist, is empty, or has invalid header.
func isValidZstdFile(filePath string) bool {
	file, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func() {
		_ = file.Close()
	}()

	buf := make([]byte, 4)
	n, err := file.Read(buf)
	if err != nil || n < 4 {
		return false
	}

	return buf[0] == 0x28 && buf[1] == 0xB5 && buf[2] == 0x2F && buf[3] == 0xFD
}

// compressedSink wraps a zstd encoder to provide compressed log file writing.
// It implements the WriteSyncer interface required by zap's custom sinks.
type compressedSink struct {
	file    *os.File
	encoder *zstd.Encoder
}

// Write writes compressed data to the underlying file via the zstd encoder.
// Returns len(p) on success to satisfy io.Writer contract, regardless of
// how many compressed bytes were written.
func (s *compressedSink) Write(p []byte) (int, error) {
	_, err := s.encoder.Write(p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Sync flushes the encoder buffer and syncs the file to disk.
func (s *compressedSink) Sync() error {
	if err := s.encoder.Flush(); err != nil {
		return err
	}
	return s.file.Sync()
}

// Close closes the encoder and then closes the underlying file.
// Always closes the file to prevent file descriptor leaks, even if
// encoder close fails.
func (s *compressedSink) Close() error {
	encErr := s.encoder.Close()
	fileErr := s.file.Close()

	if encErr != nil {
		return encErr
	}
	return fileErr
}

func initializeLogger(runner *interp.Runner) (*zap.Logger, error) {
	logLevel := environment.GetLogLevel(runner)
	if BUILD_VERSION == "dev" {
		logLevel = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	if environment.ShouldCleanLogFile(runner) {
		_ = os.Remove(core.LogFile())
	}

	// Initialize the logger
	loggerConfig := zap.NewProductionConfig()
	loggerConfig.Level = logLevel
	loggerConfig.OutputPaths = []string{
		"zstd://" + core.LogFile(),
	}
	logger, err := loggerConfig.Build()
	if err != nil {
		return nil, err
	}

	return logger, nil
}

func initializeHistoryManager() (*history.HistoryManager, error) {
	historyManager, err := history.NewHistoryManager(core.HistoryFile())
	if err != nil {
		return nil, err
	}

	return historyManager, nil
}

func initializeAnalyticsManager() (*analytics.AnalyticsManager, error) {
	analyticsManager, err := analytics.NewAnalyticsManager(core.AnalyticsFile())
	if err != nil {
		return nil, err
	}

	return analyticsManager, nil
}

func initializeCompletionManager() *completion.CompletionManager {
	return completion.NewCompletionManager()
}

// initializeRunner loads the shell configuration files and sets up the interpreter.
func initializeRunner(analyticsManager *analytics.AnalyticsManager, historyManager *history.HistoryManager, completionManager *completion.CompletionManager, stderrCapturer *core.StderrCapturer) (*interp.Runner, error) {
	shellPath, err := os.Executable()
	if err != nil {
		panic(err)
	}
	// Create a dynamic environment that can include GSH variables
	dynamicEnv := environment.NewDynamicEnviron()
	// Set initial system environment variables
	dynamicEnv.UpdateSystemEnv()
	// Add BISH-specific environment variables
	dynamicEnv.UpdateBishVar("SHELL", shellPath)
	dynamicEnv.UpdateBishVar("BISH_BUILD_VERSION", BUILD_VERSION)
	env := expand.Environ(dynamicEnv)

	var runner *interp.Runner

	// Create interpreter with all necessary configuration in a single call
	runner, err = interp.New(
		interp.Interactive(true),
		interp.Env(env),
		interp.StdIO(os.Stdin, os.Stdout, stderrCapturer),
		interp.ExecHandlers(
			core.NewAutocdExecHandler(), // Must be first to intercept path-like commands
			bash.NewCdCommandHandler(),
			bash.NewTypesetCommandHandler(),
			bash.SetBuiltinHandler(),
			analytics.NewAnalyticsCommandHandler(analyticsManager),
			evaluate.NewEvaluateCommandHandler(analyticsManager),
			history.NewHistoryCommandHandler(historyManager),
			completion.NewCompleteCommandHandler(completionManager),
		),
	)
	if err != nil {
		panic(err)
	}

	// Set the runner for the autocd handler
	core.SetAutocdRunner(runner)

	// load default vars
	if err := bash.RunBashScriptFromReader(
		context.Background(),
		runner,
		bytes.NewReader(DEFAULT_VARS),
		"bish",
	); err != nil {
		panic(err)
	}

	// Override cd command to run builtin cd first, then sync our state
	// The builtin cd updates the interpreter's internal directory tracking
	// The bish_cd_hook syncs os.Chdir(), runner.Dir, os.Setenv(PWD), etc.
	// We use $PWD which is set by builtin cd after it changes the directory
	if _, _, err := bash.RunBashCommand(context.Background(), runner, `function cd() { builtin cd "$@" && bish_cd_hook "$PWD"; }`); err != nil {
		panic(err)
	}

	var configFiles []string

	// If custom rcfile is provided, use it instead of the default ones
	if *rcFile != "" {
		configFiles = []string{*rcFile}
	} else {
		configFiles = []string{
			filepath.Join(core.HomeDir(), ".bishrc"),
			filepath.Join(core.HomeDir(), ".bishenv"),
		}

		// Check if this is a login shell
		if *loginShell || strings.HasPrefix(os.Args[0], "-") {
			// Prepend .bish_profile to the list of config files
			configFiles = append(
				[]string{
					"/etc/profile",
					filepath.Join(core.HomeDir(), ".bish_profile"),
				},
				configFiles...,
			)
		}
	}

	for _, configFile := range configFiles {
		if stat, err := os.Stat(configFile); err == nil && stat.Size() > 0 {
			if err := bash.RunBashScriptFromFile(context.Background(), runner, configFile); err != nil {
				// Enhanced error reporting with context
				fmt.Fprintf(os.Stderr, "Configuration file %s contains errors: %v\n", configFile, err)

				if *strictConfig {
					// In strict mode (like bash 'set -e'), fail fast on configuration errors
					return nil, fmt.Errorf("aborting due to configuration error in %s: %w", configFile, err)
				}
				// In permissive mode (default), continue despite configuration errors
				// This maintains backward compatibility while providing better visibility
			}
			// Configuration loaded successfully in permissive mode
		}
		// File not found or empty - this is normal behavior, not an error
	}

	// Sync gsh variables to system environment so they're visible to 'env' command
	environment.SyncVariablesToEnv(runner)

	analyticsManager.Runner = runner

	// Set the global runner for command handlers that need to update interpreter state
	bash.SetTypesetRunner(runner)
	bash.SetCdRunner(runner)

	return runner, nil
}
