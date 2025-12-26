package bash

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"mvdan.cc/sh/v3/interp"
)

// NewCdCommandHandler creates a new ExecHandler middleware for the cd command
func NewCdCommandHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Handle the 'bish_cd' command on all platforms
			// Handle the native 'cd' command on Windows only
			commandName := args[0]
			if runtime.GOOS == "windows" {
				if commandName != "bish_cd" && commandName != "cd" {
					return next(ctx, args)
				}
			} else {
				// On non-Windows platforms, only handle 'bish_cd'
				if commandName != "bish_cd" {
					return next(ctx, args)
				}
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

	// Update PWD and OLDPWD environment variables
	oldPwd := env.Get("PWD").String()

	// Update OS environment variables (for child processes)
	if err := os.Setenv("OLDPWD", oldPwd); err != nil {
		fmt.Fprintf(os.Stderr, "cd: failed to set OLDPWD: %v\n", err)
	}
	if err := os.Setenv("PWD", targetDir); err != nil {
		fmt.Fprintf(os.Stderr, "cd: failed to set PWD: %v\n", err)
	}

	// NOTE: The current implementation only updates the OS environment variables via os.Setenv()
	// but does not update the interpreter's internal environment. This means that:
	// 1. Child processes will see the updated PWD/OLDPWD
	// 2. The shell's internal PWD/OLDPWD variables remain stale
	// 3. Subsequent calls to env.Get("PWD") will return the old value

	// TODO: Implement proper interpreter environment update
	// This requires accessing the underlying runner.Vars or finding the correct interface
	// to update the interpreter's internal environment variables.

	return nil
}
