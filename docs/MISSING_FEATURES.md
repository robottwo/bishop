# Missing User-Friendly Shell Features

This document identifies common user-friendly features found in other modern shells (bash, zsh, fish, nushell) that Bishop is currently missing.

## High Priority - Core UX Features

### 1. Syntax Highlighting in Input
**Found in:** fish, zsh (with plugins)
**Status:** On ROADMAP
**Description:** Colorize commands as they're typed - valid commands in green, errors in red, strings in quotes highlighted, etc. This provides instant visual feedback before execution.

### 2. History Expansion
**Found in:** bash, zsh
**Status:** On ROADMAP
**Description:** Standard bash history expansion patterns:
- `!!` - repeat last command
- `!$` - last argument of previous command
- `!^` - first argument of previous command
- `!n` - execute command number n
- `!-n` - execute nth previous command
- `!string` - most recent command starting with string
- `^old^new` - quick substitution

### 3. Keybind Configuration (inputrc)
**Found in:** bash, zsh
**Status:** On ROADMAP
**Description:** Allow users to customize key bindings via config file (similar to `.inputrc`). Currently keybindings are hardcoded in `DefaultKeyMap`.

### 4. Vi Editing Mode
**Found in:** bash, zsh, fish
**Status:** Missing
**Description:** Alternative to Emacs mode - modal editing with vim keybindings (`set -o vi` in bash). Many power users prefer this.

---

## Medium Priority - Navigation & Convenience

### 5. Autocd
**Found in:** zsh, fish, bash (4.0+)
**Status:** Missing
**Description:** Automatically `cd` into a directory by just typing its name, without the `cd` command. Enable with `shopt -s autocd` in bash.

### 6. Directory Stack (pushd/popd/dirs)
**Found in:** bash, zsh, fish
**Status:** Partial (delegated to bash)
**Description:** Built-in directory stack management with visual feedback and completions. Commands like `pushd`, `popd`, `dirs`, and `cd -` for previous directory.

### 7. CDPATH Support
**Found in:** bash, zsh
**Status:** Missing
**Description:** List of directories to search when `cd`ing to a relative path. Allows `cd myproject` to work from anywhere if `myproject` is in a CDPATH directory.

### 8. Directory Bookmarks / Named Directories
**Found in:** zsh, fish
**Status:** Missing
**Description:** Save frequently used directories with short names. In zsh: `hash -d docs=~/Documents` then `cd ~docs`. Fish has `set -U fish_user_paths`.

### 9. Spelling Correction / "Did You Mean"
**Found in:** zsh, fish
**Status:** Missing
**Description:** Automatic correction of typos in commands and paths:
```
$ gti status
zsh: correct 'gti' to 'git' [nyae]?
```

### 10. Command-Not-Found Handler
**Found in:** bash, zsh, fish (via packages)
**Status:** Missing
**Description:** When a command isn't found, suggest packages that provide it:
```
$ htop
Command 'htop' not found, but can be installed with:
  sudo apt install htop
```

---

## Medium Priority - Display & Feedback

### 11. Right Prompt (RPROMPT)
**Found in:** zsh, fish
**Status:** Missing
**Description:** Display information on the right side of the terminal (git status, time, etc.). Disappears when typing approaches it.

### 12. Transient Prompt
**Found in:** zsh (p10k), fish
**Status:** Missing
**Description:** Simplify the prompt after command execution to reduce scrollback clutter. Only the current prompt is fully decorated.

### 13. Command Execution Time Display
**Found in:** fish, zsh (plugins)
**Status:** Missing
**Description:** Automatically show how long a command took to execute when it exceeds a threshold (e.g., >5 seconds).

### 14. Themes / Colorschemes
**Found in:** fish, zsh (oh-my-zsh)
**Status:** Minimal
**Description:** Currently only has basic styles in `internal/styles/`. Could offer user-configurable color themes for the entire shell experience.

---

## Lower Priority - Advanced Features

### 15. Abbreviations
**Found in:** fish
**Status:** Missing
**Description:** Short text that expands to full command when you press space:
```
abbr -a gco 'git checkout'
# typing "gco " expands to "git checkout "
```
Different from aliases because the expansion is visible before execution.

### 16. Edit Command in $EDITOR (Ctrl+X Ctrl+E)
**Found in:** bash, zsh
**Status:** Missing
**Description:** Open the current command line in `$EDITOR` for complex multi-line editing, then execute when editor closes.

### 17. Glob Qualifiers
**Found in:** zsh
**Status:** Missing
**Description:** Advanced glob patterns like `**/*(.)` (all files), `**/*(/)` (all directories), `*(mh-1)` (modified in last hour).

### 18. Extended Globbing / Brace Expansion Improvements
**Found in:** zsh, bash
**Status:** Delegated to bash
**Description:** Patterns like `**/*.go`, `{1..10}`, `!(pattern)` for exclusion.

### 19. Global Aliases
**Found in:** zsh
**Status:** Missing
**Description:** Aliases that expand anywhere in the command, not just at the beginning:
```
alias -g L='| less'
alias -g G='| grep'
# Usage: cat file.txt G pattern L
```

### 20. Job Control UI
**Found in:** bash, zsh
**Status:** Basic (delegated to bash)
**Description:** Better UI for background jobs with `jobs`, `fg`, `bg`. Visual indicators for running background processes.

### 21. Process Substitution Enhancements
**Found in:** zsh, bash
**Status:** Basic
**Description:** Easier syntax for comparing outputs: `diff <(cmd1) <(cmd2)`

---

## Quick Wins (Low Effort, High Value)

1. **`cd -`** for previous directory (may already work via bash)
2. **Show execution time** for long-running commands
3. **Autocd** toggle in config
4. **`!!` and `!$`** history expansion
5. **Spelling correction** with confirmation prompt

---

## Already Implemented Well

For context, Bishop already handles these features well:
- Tab completion with multiple sources
- Ctrl+R history search with fuzzy matching
- Directory-aware history filtering
- Emacs-style key bindings
- Kill ring (Ctrl+Y, Alt+Y)
- Clear screen (Ctrl+L)
- AI-powered command suggestions
- Git status in prompt/border

---

## Recommendation

Based on impact and effort, recommended implementation order:

1. **History expansion** (`!!`, `!$`, `!-1`) - Users expect this
2. **Syntax highlighting** - Visual feedback is major UX win
3. **Autocd** - Simple to implement, popular feature
4. **Command execution time** - Easy to add, helpful feedback
5. **Vi mode** - Many power users need this
6. **Spelling correction** - Reduces friction from typos
7. **Right prompt** - Popular for displaying info
8. **Edit in $EDITOR** - Power user feature
