# Autocd Feature Specification

## Overview

Autocd allows users to change directories by typing just the directory path, without needing to prefix it with `cd`. This is a popular feature in zsh, fish, and bash 4.0+.

**Example:**
```bash
# Without autocd
bish> cd /home/user/projects

# With autocd enabled
bish> /home/user/projects
# → automatically executes: cd /home/user/projects

bish> ..
# → automatically executes: cd ..

bish> ~/Documents
# → automatically executes: cd ~/Documents
```

## User Stories

1. **As a power user**, I want to navigate directories faster by typing just the path, so I can reduce keystrokes and work more efficiently.

2. **As a new user**, I want autocd to be opt-in, so my shell behaves predictably like bash by default.

3. **As a user**, I want clear feedback when autocd triggers, so I understand what happened.

## Configuration

### Environment Variable

```bash
# Enable autocd (in ~/.bishrc)
export BISH_AUTOCD=1

# Disable autocd (default)
export BISH_AUTOCD=0
# or simply don't set it
```

### Shell Option (Future Enhancement)

```bash
# Enable
shopt -s autocd

# Disable
shopt -u autocd

# Check status
shopt autocd
```

## Behavior Specification

### When Autocd Triggers

Autocd should trigger when ALL of the following conditions are met:

1. `BISH_AUTOCD` is set to `1` (or `true`, `yes`, `on`)
2. The input is a single "word" (no pipes, redirects, semicolons, or other shell operators)
3. The input is NOT a valid command name (not found in PATH, not a builtin, not a function, not an alias)
4. The input resolves to an existing directory

### Resolution Order

When the user types `foo`:

1. **Check if it's a command** - Look up `foo` in:
   - Shell builtins (`cd`, `exit`, `history`, etc.)
   - Functions defined in the session
   - Aliases
   - Commands in `$PATH`

2. **If not a command, check if it's a directory**:
   - Literal path: `/absolute/path`, `./relative`, `../parent`
   - Tilde expansion: `~/Documents`, `~user/files`
   - Relative to current directory: `subdir`
   - Environment variable expansion: `$HOME/projects`

3. **If it's a directory, execute `cd <path>`**

4. **If not a directory, show "command not found"** (normal behavior)

### Edge Cases

| Input | Behavior |
|-------|----------|
| `ls` | Execute `ls` (command exists) |
| `/etc` | `cd /etc` (directory, not a command) |
| `..` | `cd ..` (always a directory) |
| `.` | `cd .` (current directory) |
| `~` | `cd ~` (home directory) |
| `~/bin` | Check if `~/bin` is a dir, then `cd ~/bin` |
| `foo` (dir exists, no cmd) | `cd foo` |
| `foo` (cmd exists) | Execute `foo` command |
| `foo` (neither exists) | "command not found: foo" |
| `foo bar` | Normal execution (multiple words) |
| `./script.sh` | Execute script (file, not directory) |
| `/etc/passwd` | "command not found" or permission error (file, not dir) |
| `foo; bar` | Normal execution (compound command) |
| `foo \| bar` | Normal execution (pipeline) |

### Feedback

When autocd triggers, optionally print the effective command:

```bash
bish> /etc
cd /etc
bish>
```

This can be controlled by:
```bash
export BISH_AUTOCD_VERBOSE=1  # Show "cd <path>" when autocd triggers
```

## Implementation Guide

### Location

The autocd check should be added in `internal/core/shell.go`, in the main loop **before** calling `executeCommand()`.

### Pseudocode

```go
// In RunInteractiveShell(), after handling empty input and @ commands:

// Handle autocd if enabled
if environment.IsAutocdEnabled(runner) && !isCompoundCommand(line) {
    // Check if line could be a directory
    resolvedPath := expandPath(line, runner)  // Handle ~, $HOME, etc.

    if !isCommandOrBuiltin(line, runner) && isDirectory(resolvedPath) {
        // Optionally print feedback
        if environment.IsAutocdVerbose(runner) {
            fmt.Printf("cd %s\n", line)
        }
        // Rewrite command as cd
        line = "cd " + shellQuote(line)
    }
}

// Execute the command (possibly rewritten)
shouldExit, err := executeCommand(ctx, line, ...)
```

### Helper Functions Needed

```go
// internal/environment/autocd.go

// IsAutocdEnabled checks if BISH_AUTOCD is enabled
func IsAutocdEnabled(runner *interp.Runner) bool {
    val := GetEnv(runner, "BISH_AUTOCD")
    return val == "1" || val == "true" || val == "yes" || val == "on"
}

// IsAutocdVerbose checks if BISH_AUTOCD_VERBOSE is enabled
func IsAutocdVerbose(runner *interp.Runner) bool {
    val := GetEnv(runner, "BISH_AUTOCD_VERBOSE")
    return val == "1" || val == "true" || val == "yes" || val == "on"
}
```

```go
// internal/core/autocd.go

// isCompoundCommand checks if input contains shell operators
func isCompoundCommand(input string) bool {
    // Check for pipes, semicolons, &&, ||, redirects, etc.
    // This should use proper parsing, not just string matching
    // to avoid false positives in quoted strings
}

// isCommandOrBuiltin checks if the word is a known command
func isCommandOrBuiltin(word string, runner *interp.Runner) bool {
    // Check builtins
    // Check functions
    // Check aliases
    // Check PATH
}

// expandPath expands ~ and environment variables in a path
func expandPath(path string, runner *interp.Runner) string {
    // Handle ~/path → /home/user/path
    // Handle ~user/path → /home/user/path
    // Handle $VAR/path → expanded/path
}

// isDirectory checks if path exists and is a directory
func isDirectory(path string) bool {
    info, err := os.Stat(path)
    return err == nil && info.IsDir()
}

// shellQuote properly quotes a path for shell execution
func shellQuote(s string) string {
    // Handle paths with spaces, special chars, etc.
}
```

### Files to Modify

1. **`internal/core/shell.go`** - Add autocd check in main loop
2. **`internal/core/autocd.go`** (new) - Autocd logic and helpers
3. **`internal/environment/autocd.go`** (new) - Environment variable helpers
4. **`cmd/bish/.bishrc.default`** - Add commented example config
5. **`docs/CONFIGURATION.md`** - Document the feature

### Testing

```go
// internal/core/autocd_test.go

func TestIsCompoundCommand(t *testing.T) {
    tests := []struct {
        input    string
        expected bool
    }{
        {"ls", false},
        {"/etc", false},
        {"ls | grep foo", true},
        {"ls; pwd", true},
        {"ls && pwd", true},
        {"ls > file", true},
        {"echo 'hello | world'", false},  // pipe in quotes
        {"path/to/dir", false},
        {"ls -la", false},  // args don't make it compound
    }
    // ...
}

func TestAutocdBehavior(t *testing.T) {
    // Create temp directories
    // Test various inputs
    // Verify cd is executed
}
```

## Security Considerations

1. **Path Traversal**: Autocd should respect the same security constraints as `cd`. No special handling needed since we delegate to `cd`.

2. **Symlink Following**: Follow symlinks as `cd` normally does.

3. **Permission Errors**: If the directory exists but isn't accessible, let `cd` handle the error naturally.

4. **Injection Prevention**: The path must be properly quoted when constructing the `cd` command to prevent injection if a directory name contains special characters.

## Compatibility Notes

- **bash**: `shopt -s autocd` (bash 4.0+)
- **zsh**: `setopt AUTO_CD`
- **fish**: Enabled by default

Bishop's implementation aligns with bash's behavior where autocd is opt-in and disabled by default.

## Success Criteria

1. [ ] Typing a directory path changes to that directory when autocd is enabled
2. [ ] Autocd is disabled by default
3. [ ] Commands take precedence over directories with the same name
4. [ ] Compound commands (pipes, semicolons, etc.) are not affected
5. [ ] Paths with spaces and special characters work correctly
6. [ ] Tilde and environment variable expansion works
7. [ ] Clear error messages when directory doesn't exist
8. [ ] Optional verbose mode shows the effective cd command
9. [ ] Configuration documented in `.bishrc.default`
10. [ ] Feature documented in `docs/CONFIGURATION.md`

## Future Enhancements

1. **`shopt` integration**: Support `shopt -s autocd` as alternative to env var
2. **CDPATH integration**: When combined with CDPATH, autocd could search CDPATH directories
3. **Fuzzy matching**: Optionally suggest corrections for typos (e.g., "Did you mean `Documents`?")
