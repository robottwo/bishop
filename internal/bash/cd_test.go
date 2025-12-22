package bash

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"mvdan.cc/sh/v3/expand"
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
			expectedError: "directory not found",
		},
		{
			name:          "cd to file",
			args:          []string{"bish_cd", file},
			expectedError: "not a directory",
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			// Convert map to slice for ListEnviron
			var envSlice []string
			for k, v := range envMap {
				envSlice = append(envSlice, k+"="+v)
			}

			env := expand.ListEnviron(envSlice...)

			r, err := interp.New(interp.Env(env), interp.ExecHandlers(NewCdCommandHandler()))
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
