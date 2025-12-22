package bash

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"mvdan.cc/sh/v3/interp"
)

// NewCdCommandHandler creates a new ExecHandler middleware for the cd command
func NewCdCommandHandler() func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
	return func(next interp.ExecHandlerFunc) interp.ExecHandlerFunc {
		return func(ctx context.Context, args []string) error {
			if len(args) == 0 {
				return next(ctx, args)
			}

			// Handle only the 'bish_cd' command
			if args[0] != "bish_cd" {
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
	}

	// Expand any path variables and handle special cases
	if targetDir == "~" {
		home := env.Get("HOME")
		if home.String() == "" {
			fmt.Fprintln(os.Stderr, "cd: HOME not set")
			return fmt.Errorf("no home directory")
		}
		targetDir = home.String()
	} else if targetDir == "-" {
		// Handle previous directory
		prevDir := env.Get("OLDPWD")
		if prevDir.String() == "" {
			fmt.Fprintln(os.Stderr, "cd: OLDPWD not set")
			return fmt.Errorf("no previous directory")
		}
		targetDir = prevDir.String()
	}

	// Resolve relative paths to absolute path
	if !filepath.IsAbs(targetDir) {
		currentDir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "cd: unable to get current directory: %v\n", err)
			return err
		}
		targetDir = filepath.Join(currentDir, targetDir)
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
		return fmt.Errorf("not a directory")
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
	os.Setenv("OLDPWD", oldPwd)
	os.Setenv("PWD", targetDir)

	return nil
}
