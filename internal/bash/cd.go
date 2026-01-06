package bash

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"mvdan.cc/sh/v3/expand"
	"mvdan.cc/sh/v3/interp"
)

// Global runner reference for the cd command handler
// NOTE: This global variable pattern is intentionally used here due to the constraints
// of the interp.ExecHandlerFunc signature, which doesn't allow passing additional context.
// The runner must be available to update internal PWD/OLDPWD variables alongside OS env.
var cdRunner *interp.Runner

// SetCdRunner sets the global runner reference for the cd command handler.
// This enables the cd command to update both OS environment variables and
// the interpreter's internal PWD/OLDPWD variables for consistency.
func SetCdRunner(runner *interp.Runner) {
	cdRunner = runner
}

// NewCdCommandHandler creates a new ExecHandler middleware for the cd command
func NewCdCommandHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Handle 'cd' and 'bish_cd' commands on all platforms
			// This ensures runner.Dir and environment variables stay in sync
			commandName := args[0]
			if commandName != "bish_cd" && commandName != "cd" {
				return next(ctx, args)
			}

			// Handle cd command
			return handleCdCommand(ctx, args)
		}
	}
}

func handleCdCommand(ctx context.Context, args []string) error {
	// Get environment from context
	hc := interp.HandlerCtx(ctx)
	env := hc.Env

	// Determine the target directory
	var targetDir string
	if len(args) == 1 {
		// No argument provided, try HOME
		home := env.Get("HOME")
		if home.String() == "" {
			fmt.Fprintln(os.Stderr, "cd: HOME not set")
			return fmt.Errorf("no home directory")
		}
		targetDir = home.String()
	} else {
		targetDir = args[1]

		// Windows path recovery: minimal generic fallback for malformed paths
		// This addresses cases where backslashes are stripped from Windows paths
		// The real fix should be in the command parsing/escaping layer
		if runtime.GOOS == "windows" && strings.Contains(targetDir, ":") && !strings.Contains(targetDir, `\`) {
			// Use the sophisticated reconstruction function from cd_windows.go
			reconstructed := reconstructWindowsPath(targetDir)

			// Validate the reconstructed path exists before using it
			if _, err := os.Stat(reconstructed); err == nil {
				targetDir = reconstructed
			} else {
				fmt.Fprintf(os.Stderr, "cd: no such file or directory: %s\n", targetDir)
				return fmt.Errorf("directory not found")
			}
		}
	}

	// Track if we should print the path (for cd -)
	printPath := false

	// Expand any path variables and handle special cases
	switch targetDir {
	case "~":
		home := env.Get("HOME")
		if home.String() == "" {
			fmt.Fprintln(os.Stderr, "cd: HOME not set")
			return fmt.Errorf("no home directory")
		}
		targetDir = home.String()
	case "-":
		// Handle previous directory
		prevDir := env.Get("OLDPWD")
		if prevDir.String() == "" {
			fmt.Fprintln(os.Stderr, "cd: OLDPWD not set")
			return fmt.Errorf("no previous directory")
		}
		targetDir = prevDir.String()
		printPath = true
	}

	// Determine if path is absolute - be extra careful on Windows
	isAbs := filepath.IsAbs(targetDir)

	// On Windows, also check for drive letter pattern as additional validation
	// This fixes cases where filepath.IsAbs() might not recognize Windows paths correctly
	if runtime.GOOS == "windows" && !isAbs {
		// Check if it looks like a Windows absolute path (e.g., C:\path or C:/path)
		if len(targetDir) >= 3 && targetDir[1] == ':' && (targetDir[2] == '\\' || targetDir[2] == '/') {
			isAbs = true
		}
		// Also check for UNC paths (\\server\share)
		if len(targetDir) >= 2 && targetDir[0] == '\\' && targetDir[1] == '\\' {
			isAbs = true
		}
	}

	if isAbs {
		// For absolute paths, just clean but don't join with current directory
		// On Windows, ensure we use the correct path separator
		if runtime.GOOS == "windows" {
			targetDir = filepath.Clean(targetDir)
			// Convert forward slashes to backslashes for consistency on Windows
			targetDir = filepath.FromSlash(targetDir)
		} else {
			targetDir = filepath.Clean(targetDir)
		}
	} else {
		// For relative paths, resolve relative to current directory
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: unable to get current directory: %v\n", err)
			return err
		}

		// On Windows, ensure we use the correct path separator
		if runtime.GOOS == "windows" {
			// Convert forward slashes to backslashes in targetDir before joining
			targetDir = filepath.FromSlash(targetDir)
		}

		targetDir = filepath.Join(currentDir, targetDir)
		targetDir = filepath.Clean(targetDir)

		// Debug: Log final path after joining
	}

	// Check if the directory exists
	info, err := os.Stat(targetDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "cd: no such file or directory: %s\n", targetDir)
			return fmt.Errorf("directory not found")
		}
		fmt.Fprintf(os.Stderr, "cd: %s: %v\n", targetDir, err)
		return err
	}

	// Check if it's actually a directory
	if !info.IsDir() {
		fmt.Fprintf(os.Stderr, "cd: not a directory: %s\n", targetDir)
		return fmt.Errorf("directory not found") // Use consistent error message for cross-platform compatibility
	}

	// Check for read and execute permissions
	if err := os.Chdir(targetDir); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stderr, "cd: permission denied: %s\n", targetDir)
			return fmt.Errorf("permission denied")
		}
		fmt.Fprintf(os.Stderr, "cd: %s: %v\n", targetDir, err)
		return err
	}

	// Determine the old working directory for OLDPWD
	// Priority: runner.Dir > env.Get("PWD") > os.Getwd()
	var oldPwd string
	if cdRunner != nil && cdRunner.Dir != "" {
		oldPwd = cdRunner.Dir
	} else if pwd := env.Get("PWD").String(); pwd != "" {
		oldPwd = pwd
	} else if wd, err := os.Getwd(); err == nil {
		// Note: os.Getwd() returns the NEW directory since os.Chdir already happened
		// This is a fallback that shouldn't normally be needed
		oldPwd = wd
	}

	// Update OS environment variables (for child processes)
	if err := os.Setenv("OLDPWD", oldPwd); err != nil {
		fmt.Fprintf(os.Stderr, "cd: failed to set OLDPWD: %v\n", err)
	}
	if err := os.Setenv("PWD", targetDir); err != nil {
		fmt.Fprintf(os.Stderr, "cd: failed to set PWD: %v\n", err)
	}

	// Update the interpreter's internal state
	// This ensures that:
	// 1. runner.Dir reflects the new working directory (used by the interpreter)
	// 2. $PWD and $OLDPWD environment variables are correctly set for shell expansion
	if cdRunner != nil {
		// Update the interpreter's working directory - this is critical!
		// runner.Dir is what the interpreter uses as the current directory
		cdRunner.Dir = targetDir

		// Initialize runner.Vars if nil (can happen with fresh runner)
		if cdRunner.Vars == nil {
			cdRunner.Vars = make(map[string]expand.Variable)
		}

		// Update the environment variables for shell expansion
		cdRunner.Vars["OLDPWD"] = expand.Variable{Kind: expand.String, Str: oldPwd, Exported: true}
		cdRunner.Vars["PWD"] = expand.Variable{Kind: expand.String, Str: targetDir, Exported: true}
	}

	// Print the new directory path for cd - (matches bash behavior)
	if printPath {
		_, _ = fmt.Fprintln(hc.Stdout, targetDir)
	}

	return nil
}
