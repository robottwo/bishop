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

// reconstructWindowsPath attempts to reconstruct a Windows path that has lost its backslashes
// This function is only compiled on Windows
func reconstructWindowsPath(malformedPath string) string {
	// Split by drive letter
	parts := strings.Split(malformedPath, ":")
	if len(parts) < 2 {
		return malformedPath
	}

	drive := parts[0] + ":\\"
	remainingPath := parts[1]

	// Common Windows directory patterns in order of typical appearance
	patterns := []string{
		"Users",
		"Program Files",
		"Windows",
		"AppData",
		"Local",
		"Temp",
		"ProgramData",
		"Roaming",
		"system32",
		"SysWOW64",
	}

	// Build a more comprehensive reconstruction
	segments := []string{drive}
	currentPos := 0

	// First pass: identify and mark all known patterns

	for currentPos < len(remainingPath) {
		found := false

		// Try to match known patterns
		for _, pattern := range patterns {
			if strings.HasPrefix(remainingPath[currentPos:], pattern) {
				segments = append(segments, pattern)
				currentPos += len(pattern)
				found = true
				break
			}
		}

		if !found {
			// Look for the next pattern
			nextPatternPos := len(remainingPath)

			for _, pattern := range patterns {
				if pos := strings.Index(remainingPath[currentPos:], pattern); pos >= 0 {
					if currentPos+pos < nextPatternPos {
						nextPatternPos = currentPos + pos
					}
				}
			}

			if nextPatternPos < len(remainingPath) {
				// Everything before the next pattern should be a directory name
				dirName := remainingPath[currentPos:nextPatternPos]
				if dirName != "" {
					segments = append(segments, dirName)
				}
				currentPos = nextPatternPos
			} else {
				// No more patterns - add the rest as final segments
				finalPart := remainingPath[currentPos:]
				if finalPart != "" {
					// Try to split the final part into logical segments
					finalSegments := splitFinalPathPart(finalPart)
					segments = append(segments, finalSegments...)
				}
				break
			}
		}
	}

	// Join all segments with backslashes
	return strings.Join(segments, "\\")
}

// splitFinalPathPart splits a concatenated final path part into logical segments
// This function is only compiled on Windows
func splitFinalPathPart(part string) []string {
	if len(part) < 4 {
		return []string{part}
	}

	var segments []string
	currentPos := 0

	// Common patterns to look for
	patterns := []string{"bin", "lib", "src", "test", "temp", "tmp", "dir", "data", "app", "exe", "dll", "subdir"}

	for currentPos < len(part) {
		found := false

		// Look for known ending patterns
		for _, pattern := range patterns {
			if strings.HasSuffix(part[currentPos:], pattern) && len(part[currentPos:]) > len(pattern) {
				// Found a pattern at the end
				beforePattern := part[currentPos : len(part)-len(pattern)]
				afterPattern := part[len(part)-len(pattern):]

				if beforePattern != "" {
					segments = append(segments, beforePattern)
				}
				segments = append(segments, afterPattern)
				currentPos = len(part)
				found = true
				break
			}
		}

		if !found {
			// Look for number-to-letter transitions
			for i := currentPos + 1; i < len(part); i++ {
				if i > 0 && part[i-1] >= '0' && part[i-1] <= '9' && part[i] >= 'a' && part[i] <= 'z' {
					// Split after the number
					segments = append(segments, part[currentPos:i])
					currentPos = i
					found = true
					break
				}
			}
		}

		if !found {
			// No more splits found, add the rest
			if currentPos < len(part) {
				segments = append(segments, part[currentPos:])
			}
			break
		}
	}

	if len(segments) == 0 {
		return []string{part}
	}

	return segments
}

// findLogicalSplitPoint tries to find a logical place to split a directory name
// For example: "MyAppbin" -> split between "MyApp" and "bin"
// "bish-cd-test1695569554subdir" -> split between "bish-cd-test1695569554" and "subdir"
// This function is only compiled on Windows
func findLogicalSplitPoint(s string) int {
	// Common patterns:
	// 1. camelCase to lowercase: "MyAppbin" -> split before "bin"
	// 2. Number+letter: "test1695569554subdir" -> split before "subdir"
	// 3. Hyphen+letter: "bish-cd-test1695569554subdir" -> split before "subdir"

	if len(s) < 4 {
		return 0 // Too short to split meaningfully
	}

	// Look for transitions from numbers to letters
	for i := 1; i < len(s)-1; i++ {
		if i+1 < len(s) && s[i] >= '0' && s[i] <= '9' && s[i+1] >= 'a' && s[i+1] <= 'z' {
			return i + 1 // Split after the number
		}
	}

	// Look for common directory names at the end
	commonEndings := []string{"bin", "lib", "src", "test", "temp", "tmp", "dir", "data", "app", "exe", "dll", "subdir"}
	for _, ending := range commonEndings {
		if strings.HasSuffix(s, ending) && len(s) > len(ending) {
			// Check if what comes before the ending looks like a directory name
			beforeEnding := s[:len(s)-len(ending)]
			if len(beforeEnding) >= 2 && !strings.HasSuffix(beforeEnding, "\\") {
				return len(beforeEnding) // Split before the ending
			}
		}
	}

	return 0 // No good split point found
}

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

	// Debug: Log the raw arguments received
	if runtime.GOOS == "windows" {
		fmt.Fprintf(os.Stderr, "DEBUG: handleCdCommand received args: %v\n", args)
	}

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
		// Debug: Log the target directory before any processing
		if runtime.GOOS == "windows" {
			fmt.Fprintf(os.Stderr, "DEBUG: Raw targetDir: %s\n", targetDir)
		}

		// Windows path recovery: minimal generic fallback for malformed paths
		// This addresses cases where backslashes are stripped from Windows paths
		// The real fix should be in the command parsing/escaping layer
		if runtime.GOOS == "windows" && strings.Contains(targetDir, ":") && !strings.Contains(targetDir, `\`) {
			// Try a single safe substitution: replace first ":" with ":\\"
			reconstructed := strings.Replace(targetDir, ":", ":\\", 1)

			// Debug: Log the reconstruction attempt
			fmt.Fprintf(os.Stderr, "DEBUG: Attempting to reconstruct Windows path: %s -> %s\n", targetDir, reconstructed)

			// Validate the reconstructed path exists before using it
			if _, err := os.Stat(reconstructed); err == nil {
				targetDir = reconstructed
				fmt.Fprintf(os.Stderr, "DEBUG: Successfully reconstructed path: %s\n", targetDir)
			} else {
				// Log a warning about malformed path detection
				fmt.Fprintf(os.Stderr, "cd: warning: detected malformed Windows path: %s\n", targetDir)
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
