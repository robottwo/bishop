package bash

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/robottwo/bishop/internal/environment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/interp"
)

func TestCdCommandHandler(t *testing.T) {
	// Setup temporary directory structure
	tmpDir, err := os.MkdirTemp("", "bish-cd-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Create a file
	file := filepath.Join(tmpDir, "file")
	err = os.WriteFile(file, []byte("content"), 0644)
	require.NoError(t, err)

	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	tests := []struct {
		name          string
		args          []string
		env           map[string]string
		expectedError string
		checkDir      string
	}{
		{
			name:     "cd to valid directory",
			args:     []string{"bish_cd", subDir},
			checkDir: subDir,
		},
		{
			name:          "cd to non-existent directory",
			args:          []string{"bish_cd", filepath.Join(tmpDir, "nonexistent")},
			expectedError: "directory not found", // This is what our cd handler returns
		},
		{
			name:          "cd to file",
			args:          []string{"bish_cd", file},
			expectedError: "directory not found", // On Windows, os.Stat might return "directory not found" instead of "not a directory"
		},
		{
			name: "cd home",
			args: []string{"bish_cd"},
			env: map[string]string{
				"HOME": subDir,
			},
			checkDir: subDir,
		},
		{
			name:          "cd home unset",
			args:          []string{"bish_cd"},
			expectedError: "no home directory",
		},
		{
			name: "cd previous",
			args: []string{"bish_cd", "-"},
			env: map[string]string{
				"OLDPWD": subDir,
			},
			checkDir: subDir,
		},
		{
			name:          "cd previous unset",
			args:          []string{"bish_cd", "-"},
			expectedError: "no previous directory",
		},
	}

	// On Windows, we should use our custom bish_cd handler instead of relying on native cd
	// The native Windows cd command has different behavior and error messages
	// Our custom handler provides consistent cross-platform behavior
	// No changes needed - our handler already provides consistent behavior across platforms

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add platform info for debugging
			t.Logf("Running on: %s/%s", runtime.GOOS, runtime.GOARCH)

			// Check if paths exist before running test
			if _, err := os.Stat(subDir); err != nil {
				t.Logf("WARNING: subDir does not exist: %v", err)
			}
			if _, err := os.Stat(file); err != nil {
				t.Logf("WARNING: file does not exist: %v", err)
			}

			// Reset working directory
			require.NoError(t, os.Chdir(tmpDir))
			// Setup environment map
			envMap := make(map[string]string)
			// Copy from OS environ first?
			// No, let's start clean to be deterministic, but we might need PATH etc.
			// Actually cd handler only needs HOME, OLDPWD, PWD.
			// And standard env vars are good.

			// Let's use os.Environ() but filter out HOME/OLDPWD if we want to override/unset them
			for _, e := range os.Environ() {
				// parse key=value
				// simplistic parsing
				if i := len(e); i > 0 {
					// find =
					for j := 0; j < i; j++ {
						if e[j] == '=' {
							key := e[:j]
							val := e[j+1:]
							envMap[key] = val
							break
						}
					}
				}
			}

			// Apply overrides
			for k, v := range tt.env {
				envMap[k] = v
			}

			// Handle unsetting for specific tests
			if tt.name == "cd home unset" {
				// Explicitly set to empty string to ensure interp doesn't fallback to OS env
				envMap["HOME"] = ""
			}
			if tt.name == "cd previous unset" {
				envMap["OLDPWD"] = ""
			}

			// Use DynamicEnviron to test the fix properly
			dynamicEnv := environment.NewDynamicEnviron()
			// Set initial system environment variables
			dynamicEnv.UpdateSystemEnv()
			// Add our test variables
			for k, v := range envMap {
				dynamicEnv.UpdateBishVar(k, v)
			}

			r, err := interp.New(interp.Env(dynamicEnv), interp.ExecHandlers(NewCdCommandHandler()))
			require.NoError(t, err)

			ctx := context.Background()

			// We need to run "bish_cd arg1 arg2"
			// But bish_cd is not a valid shell command unless we define it or if ExecHandler catches it.
			// ExecHandler catches command execution.

			// Construct command string
			cmdStr := tt.args[0]
			for _, arg := range tt.args[1:] {
				cmdStr += " " + arg
			}

			// Run command
			// Note: We use RunBashCommand helper if available, or just runner.Run
			// But we are in 'bash' package, so we can use interp directly.

			// Parsing manually to avoid dependency loops if using external helpers
			// But wait, we are in 'internal/bash', we can just use runner.

			// Using sh/v3/syntax parser
			// p := syntax.NewParser()
			// file, err := p.Parse(strings.NewReader(cmdStr), "")
			// require.NoError(t, err)
			// err = r.Run(ctx, file)

			// Simplification: just call the handler directly with a mocked context?
			// Mocking interp.HandlerCtx is hard because it uses internal keys.
			// So running via interp is best.

			// However, `bish_cd` is just a string. `interp` will try to look it up in PATH if handler calls next().
			// Our handler intercepts it.

			// Debug: Log the command we're about to run

			// Let's use a small helper to run the command string.
			err = func() error {
				// We can't easily parse here without importing syntax, which is fine.
				// But wait, the test file is in 'bash' package.
				return RunScript(ctx, r, cmdStr)
			}()

			if tt.expectedError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			} else {
				require.NoError(t, err)
				if tt.checkDir != "" {
					wd, _ := os.Getwd()
					// Evaluate symlinks just in case
					wd, _ = filepath.EvalSymlinks(wd)
					checkDir, _ := filepath.EvalSymlinks(tt.checkDir)
					assert.Equal(t, checkDir, wd)
				}
			}
		})
	}
}

// Helper to run script string
func RunScript(ctx context.Context, r *interp.Runner, code string) error {
	_, _, err := RunBashCommand(ctx, r, code)
	return err
}

// TestCdUpdatesRunnerDir verifies that cd correctly updates runner.Dir
// This is the critical fix - runner.Dir must be updated for the interpreter
// to know the current working directory.
func TestCdUpdatesRunnerDir(t *testing.T) {
	// Setup temporary directory structure
	tmpDir, err := os.MkdirTemp("", "bish-cd-runnerdir-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Start in tmpDir
	require.NoError(t, os.Chdir(tmpDir))

	// Setup environment
	dynamicEnv := environment.NewDynamicEnviron()
	dynamicEnv.UpdateSystemEnv()

	r, err := interp.New(interp.Env(dynamicEnv), interp.ExecHandlers(NewCdCommandHandler()))
	require.NoError(t, err)

	// IMPORTANT: Set the global cdRunner so the handler can update runner.Dir
	SetCdRunner(r)
	defer SetCdRunner(nil) // Clean up

	ctx := context.Background()

	// Verify initial state (use EvalSymlinks to handle macOS /var -> /private/var)
	expectedInitialDir, _ := filepath.EvalSymlinks(tmpDir)
	actualInitialDir, _ := filepath.EvalSymlinks(r.Dir)
	assert.Equal(t, expectedInitialDir, actualInitialDir, "Initial runner.Dir should be tmpDir")

	// Run cd to subdirectory
	err = RunScript(ctx, r, fmt.Sprintf("bish_cd %q", subDir))
	require.NoError(t, err)

	// Verify runner.Dir is updated
	expectedDir, _ := filepath.EvalSymlinks(subDir)
	actualDir, _ := filepath.EvalSymlinks(r.Dir)
	assert.Equal(t, expectedDir, actualDir, "runner.Dir should be updated to subDir after cd")

	// Verify os.Getwd() also matches
	osWd, _ := os.Getwd()
	osWd, _ = filepath.EvalSymlinks(osWd)
	assert.Equal(t, expectedDir, osWd, "os.Getwd() should match the new directory")
}

// TestCdUpdatesGetPwd verifies that GetPwd returns the correct directory after cd
func TestCdUpdatesGetPwd(t *testing.T) {
	// Setup temporary directory structure
	tmpDir, err := os.MkdirTemp("", "bish-cd-getpwd-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Start in tmpDir
	require.NoError(t, os.Chdir(tmpDir))

	// Setup environment
	dynamicEnv := environment.NewDynamicEnviron()
	dynamicEnv.UpdateSystemEnv()

	r, err := interp.New(interp.Env(dynamicEnv), interp.ExecHandlers(NewCdCommandHandler()))
	require.NoError(t, err)

	// Set the global cdRunner
	SetCdRunner(r)
	defer SetCdRunner(nil)

	ctx := context.Background()

	// Verify initial GetPwd
	initialPwd := environment.GetPwd(r)
	initialPwd, _ = filepath.EvalSymlinks(initialPwd)
	expectedInitial, _ := filepath.EvalSymlinks(tmpDir)
	assert.Equal(t, expectedInitial, initialPwd, "Initial GetPwd should return tmpDir")

	// Run cd to subdirectory
	err = RunScript(ctx, r, fmt.Sprintf("bish_cd %q", subDir))
	require.NoError(t, err)

	// Verify GetPwd returns the new directory
	newPwd := environment.GetPwd(r)
	newPwd, _ = filepath.EvalSymlinks(newPwd)
	expectedNew, _ := filepath.EvalSymlinks(subDir)
	assert.Equal(t, expectedNew, newPwd, "GetPwd should return subDir after cd")
}

// TestCdUpdatesOldPwd verifies that OLDPWD is set correctly after cd
func TestCdUpdatesOldPwd(t *testing.T) {
	// Setup temporary directory structure
	tmpDir, err := os.MkdirTemp("", "bish-cd-oldpwd-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Start in tmpDir
	require.NoError(t, os.Chdir(tmpDir))

	// Setup environment
	dynamicEnv := environment.NewDynamicEnviron()
	dynamicEnv.UpdateSystemEnv()

	r, err := interp.New(interp.Env(dynamicEnv), interp.ExecHandlers(NewCdCommandHandler()))
	require.NoError(t, err)

	// Set the global cdRunner
	SetCdRunner(r)
	defer SetCdRunner(nil)

	ctx := context.Background()

	// Run cd to subdirectory
	err = RunScript(ctx, r, fmt.Sprintf("bish_cd %q", subDir))
	require.NoError(t, err)

	// Verify OLDPWD in OS environment
	// Note: The OS environment is the reliable source for OLDPWD since the interpreter's
	// internal Vars map may be managed/reset by the interpreter during command execution.
	osOldPwd := os.Getenv("OLDPWD")
	osOldPwd, _ = filepath.EvalSymlinks(osOldPwd)
	expectedOld, _ := filepath.EvalSymlinks(tmpDir)
	assert.Equal(t, expectedOld, osOldPwd, "OS OLDPWD should be set to the previous directory")
}

func TestCdMinusPrintsPath(t *testing.T) {
	// Setup temporary directory structure
	tmpDir, err := os.MkdirTemp("", "bish-cd-minus-test")
	require.NoError(t, err)
	defer func() {
		_ = os.RemoveAll(tmpDir)
	}()

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// Save original working directory
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	// Start in tmpDir
	require.NoError(t, os.Chdir(tmpDir))

	// Setup environment with OLDPWD pointing to subDir
	dynamicEnv := environment.NewDynamicEnviron()
	dynamicEnv.UpdateSystemEnv()
	dynamicEnv.UpdateBishVar("OLDPWD", subDir)

	r, err := interp.New(interp.Env(dynamicEnv), interp.ExecHandlers(NewCdCommandHandler()))
	require.NoError(t, err)

	ctx := context.Background()

	// Run cd - and capture output
	stdout, _, err := RunBashCommand(ctx, r, "bish_cd -")
	require.NoError(t, err)

	// Verify the path was printed to stdout (with trailing newline)
	expectedPath, _ := filepath.EvalSymlinks(subDir)
	actualPath, _ := filepath.EvalSymlinks(strings.TrimSpace(stdout))
	assert.Equal(t, expectedPath, actualPath, "cd - should print the new directory path")
}
