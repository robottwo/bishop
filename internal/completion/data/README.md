# Completion Data Files

This directory contains YAML configuration files that define static command completions for common CLI tools. These files are embedded into the bishop binary at compile time and provide tab-completion suggestions for tool subcommands and options.

## Format

Completion files use the same format as user-defined completion configuration. Each file contains a `commands` map where:

- **Key**: The command name (e.g., `docker`, `kubectl`, `npm`)
- **Value**: Array of completion entries, each containing:
  - `value` (required): The completion text (subcommand, option, etc.)
  - `description` (optional): Human-readable description shown in completion menu

### YAML Example

```yaml
commands:
  docker:
    - value: build
      description: Build an image from a Dockerfile
    - value: run
      description: Run a command in a new container
    - value: ps
      description: List containers
    - value: pull
      description: Pull an image from a registry
    - value: push
      description: Push an image to a registry
    - value: stop
      description: Stop one or more running containers

  kubectl:
    - value: get
      description: Display one or many resources
    - value: apply
      description: Apply a configuration to a resource by file or stdin
    - value: delete
      description: Delete resources by filenames, stdin, or selectors
```

### JSON Example

JSON format is also supported (though YAML is preferred for readability):

```json
{
  "commands": {
    "docker": [
      {
        "value": "build",
        "description": "Build an image from a Dockerfile"
      },
      {
        "value": "run",
        "description": "Run a command in a new container"
      }
    ]
  }
}
```

## File Organization

Completion data is organized into category-based files:

- `containers.yaml` - Container tools (docker, docker-compose, podman, kubectl, helm)
- `package_managers.yaml` - Package managers (npm, yarn, pnpm, pip, cargo, go, apt, brew)
- `cloud.yaml` - Cloud CLIs (aws, gcloud, az, terraform)
- `databases.yaml` - Database clients (psql, mysql, redis-cli, mongosh)
- `developer_tools.yaml` - Dev tools (gh, code, vim, nvim, tmux, curl, jq, systemctl, python)

## User Overrides

Users can define their own completions or override embedded defaults by creating a configuration file at:

- `$XDG_CONFIG_HOME/bish/completions.yaml` (or `.json`)
- `~/.config/bish/completions.yaml` (or `.json`)
- `~/.bish_completions.yaml` (or `.json`)

User-defined completions take precedence over embedded defaults and use the same format as shown above.

## Contributing

When adding or updating completions:

1. Follow the existing format exactly
2. Group related commands in the appropriate category file
3. Provide clear, concise descriptions
4. Sort completions alphabetically by value within each command
5. Test that completions work correctly after embedding

## Technical Details

- Files are embedded using Go's `embed.FS` directive at compile time
- Parsing happens at startup using `gopkg.in/yaml.v3`
- All files matching `*.yaml` in this directory are automatically included
- User completions can extend or override these embedded defaults
