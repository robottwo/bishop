# Configuration

bishop is configurable via simple dotfiles and environment variables. This guide explains where configuration lives, default values, and common customization tips.

Upstream project: https://github.com/atinylittleshell/gsh  
Fork repository: https://github.com/robottwo/bishop

## Files and Load Order

The shell loads configuration in this order:

1. If launched as a login shell `gsh -l`, sources:
   - `/etc/profile`
   - `~/.bish_profile`
2. Always loads:
   - `~/.bishrc`
   - `~/.bishenv`

Reference implementation for file discovery is in [cmd/bish/main.go](../cmd/bish/main.go).

Default templates you can copy and customize:
- [.bishrc.default](../cmd/bish/.bishrc.default)
- [.bishrc.starship](../cmd/bish/.bishrc.starship)

## ~/.bishrc

Primary runtime configuration file. Recommended setup:

```bash
# Example: configure models and behavior
export BISH_FAST_MODEL_ID="qwen2.5:3b"
export BISH_AGENT_CONTEXT_WINDOW_TOKENS=6000
export BISH_MINIMUM_HEIGHT=10

# Optional: pre-approve safe patterns for agent-executed commands
# Regex, one-per-line in ~/.config/gsh/authorized_commands is managed automatically
# You can still provide defaults via env if desired:
# export BISH_AGENT_APPROVED_BASH_COMMAND_REGEX='^ls.*|^cat.*|^git status.*'

# Enable chat macros (JSON string)
export BISH_AGENT_MACROS='{
  "gitdiff": "summarize the changes in the current git diff",
  "gitpush": "create a concise commit message and push",
  "gitreview": "review my recent changes and suggest improvements"
}'
```

Tip: Keep sensitive values (e.g., API keys) in `~/.bishenv` rather than `~/.bishrc`.

## ~/.bishenv

Environment-only overrides that load after `~/.bishrc`. Useful for secrets or per-machine toggles:

```bash
# Example: OpenAI-compatible endpoint via OpenRouter or your own gateway
export OPENAI_API_KEY="sk-..."
export OPENAI_BASE_URL="https://openrouter.ai/api/v1"

# Ollama for local models
export OLLAMA_HOST="http://127.0.0.1:11434"
```

## Interactive Configuration Menu

gsh provides an interactive configuration menu accessible via the `#!config` command:

```bash
bish> #!config
```

The configuration menu allows you to:
- Configure slow model settings (API key, model ID, base URL) for chat and agent operations
- Configure fast model settings for auto-completion and suggestions
- Set the assistant box height
- Toggle safety checks for command approval

Changes made through the configuration menu are persisted to `~/.bish_config_ui` and automatically sourced in your shell.

## Autocd

Autocd allows you to change directories by typing just the path, without needing to prefix it with `cd`. This is a popular feature in zsh, fish, and bash 4.0+.

### Enable Autocd

In your `~/.bishrc`:

```bash
# Enable autocd
export BISH_AUTOCD=1

# Optional: show the effective cd command when autocd triggers
export BISH_AUTOCD_VERBOSE=1
```

### Usage Examples

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

### Behavior

Autocd triggers when ALL of the following conditions are met:

1. `BISH_AUTOCD` is set to `1`, `true`, `yes`, or `on`
2. The input is a single "word" (no pipes, redirects, semicolons, or other shell operators)
3. The input is NOT a valid command name (not found in PATH, not a builtin, not a function)
4. The input resolves to an existing directory

Commands always take precedence over directories with the same name. For example, if you have a directory named `ls`, typing `ls` will execute the `ls` command, not change to that directory.

## Common Environment Variables

- `BISH_AUTOCD`: Enable autocd feature (default: enabled). Set to `0` or `false` to disable.
- `BISH_AUTOCD_VERBOSE`: Show the effective cd command when autocd triggers (default: enabled).
- `BISH_FAST_MODEL_ID`: Model ID for the fast LLM (default: qwen2.5).
- `BISH_FAST_MODEL_PROVIDER`: LLM provider for fast model (ollama, openai, openrouter).
- `BISH_MINIMUM_HEIGHT`: Minimum number of lines reserved for prompt and UI rendering.
- `BISH_AGENT_CONTEXT_WINDOW_TOKENS`: Context window size for agent chats and tools; messages are pruned beyond this.
- `BISH_AGENT_APPROVED_BASH_COMMAND_REGEX`: Optional regex to pre-approve read-only or safe command families.
- `HTTP(S)_PROXY`, `NO_PROXY`: Standard proxy variables respected by network calls.

See defaults and comments in [.bishrc.default](../cmd/bish/.bishrc.default).

## Prompt Customization with Starship

You can use Starship to render a custom prompt.

1. Install Starship: https://starship.rs
2. Copy the example config and adapt it:
   - [.bishrc.starship](../cmd/bish/.bishrc.starship)

In your `~/.bishrc`:

```bash
# Example Starship integration
export STARSHIP_CONFIG="$HOME/.config/starship.toml"
eval "$(starship init bash)"  # or zsh if you prefer
```

Notes:
- The example includes prompt sections for exit code, duration, and gsh build version in dev mode.
- Adjust symbols, colors, and modules per your preference.

## Login Shell Setup

To make gsh your login shell (not recommended yet; experimental):

```bash
which gsh
echo "/path/to/gsh" | sudo tee -a /etc/shells
chsh -s "/path/to/gsh"
```

If you choose to run as a login shell, `gsh -l` will source `/etc/profile` and `~/.bish_profile` before `~/.bishrc`.

## Authorized Commands Store

When you approve commands during agent operations, gsh stores regex patterns in:

- `~/.config/gsh/authorized_commands`

Manage them with standard file operations:

```bash
# View
cat ~/.config/gsh/authorized_commands

# Edit
$EDITOR ~/.config/gsh/authorized_commands

# Reset
rm ~/.config/gsh/authorized_commands
```

These patterns complement any defaults you provide via environment variables.

## Custom Command Completions

bishop provides intelligent tab completion for 30+ popular CLI tools out of the box, including:

- **Container tools**: docker, docker-compose, podman, kubectl, helm
- **Package managers**: npm, yarn, pnpm, pip, cargo, apt, brew
- **Cloud CLIs**: aws, gcloud, az (Azure)
- **Database clients**: psql, mysql, redis-cli, mongosh
- **Dev tools**: go, cargo, gh (GitHub CLI), terraform
- **System tools**: systemctl, tmux, curl, jq
- **Editors**: vim, nvim, code (VS Code)

### User-Defined Completions

You can define custom completions for your own commands or tools not included by default. Create a configuration file in one of these locations (checked in order):

1. `$XDG_CONFIG_HOME/bish/completions.yaml`
2. `~/.config/bish/completions.yaml`
3. `~/.bish_completions.yaml`

JSON format is also supported (use `.json` extension).

### Configuration Format

**YAML example** (`~/.config/bish/completions.yaml`):

```yaml
commands:
  # Custom CLI tool
  myapp:
    - value: start
      description: Start the application
    - value: stop
      description: Stop the application
    - value: status
      description: Check application status
    - value: logs
      description: View application logs

  # Internal deployment tool
  deploy:
    - value: staging
      description: Deploy to staging environment
    - value: production
      description: Deploy to production environment
    - value: rollback
      description: Rollback to previous version

  # Project-specific commands
  proj:
    - value: build
    - value: test
    - value: lint
    - value: format
```

**JSON example** (`~/.config/bish/completions.json`):

```json
{
  "commands": {
    "myapp": [
      {"value": "start", "description": "Start the application"},
      {"value": "stop", "description": "Stop the application"},
      {"value": "status", "description": "Check application status"}
    ],
    "deploy": [
      {"value": "staging", "description": "Deploy to staging"},
      {"value": "production", "description": "Deploy to production"}
    ]
  }
}
```

### Fields

| Field | Required | Description |
|-------|----------|-------------|
| `value` | Yes | The completion text that will be inserted |
| `description` | No | Optional description shown in completion menu |

### Usage

After creating your configuration file, completions are loaded automatically when bishop starts. Type your command followed by a space and press Tab:

```bash
bish> myapp <Tab>
start   - Start the application
stop    - Stop the application
status  - Check application status
logs    - View application logs

bish> myapp st<Tab>
start   - Start the application
status  - Check application status
stop    - Stop the application
```

### Notes

- User-defined completions can override built-in completions for the same command
- Completions are loaded once at shell startup
- The `description` field is optional but recommended for discoverability
- Both YAML and JSON formats are supported

## Troubleshooting

- Unexpected prompt size: verify `BISH_MINIMUM_HEIGHT`.
- Missing macros: ensure `BISH_AGENT_MACROS` is valid JSON.
- API errors: confirm `OPENAI_BASE_URL` and `OPENAI_API_KEY` or Ollama connectivity.
- Login shell confusion: confirm whether you started gsh as a login shell and which profile files are being sourced.

## Related Docs

- Quick start: [GETTING_STARTED.md](GETTING_STARTED.md)
- Features: [FEATURES.md](FEATURES.md)
- Agent: [AGENT.md](AGENT.md)
- Subagents overview: [SUBAGENTS.md](SUBAGENTS.md)