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

## Creating Your Own Completions

Users can define their own completions or override embedded defaults by creating a configuration file. This is useful for:

- Adding completions for custom scripts or tools
- Extending completions for existing tools
- Overriding or customizing embedded completions
- Adding company-specific commands or workflows

### Configuration File Location

Create your completion file at one of these locations (checked in order):

1. `$XDG_CONFIG_HOME/bish/completions.yaml` (or `.json`)
2. `~/.config/bish/completions.yaml` (or `.json`)
3. `~/.bish_completions.yaml` (or `.json`)

User-defined completions automatically override embedded defaults for the same command.

### Step-by-Step Guide

#### 1. Create the Configuration Directory

```bash
mkdir -p ~/.config/bish
```

#### 2. Create Your Completions File

```bash
touch ~/.config/bish/completions.yaml
```

#### 3. Add Your Completions

Open the file in your editor and add completions following the format below.

### Basic Example

Here's a simple example for a custom deployment script:

```yaml
commands:
  deploy:
    - value: staging
      description: Deploy to staging environment
    - value: production
      description: Deploy to production environment
    - value: rollback
      description: Rollback to previous version
    - value: status
      description: Check deployment status
```

Now when you type `deploy` followed by TAB, you'll see these completions with their descriptions.

### Advanced Examples

#### Multiple Commands

Define completions for multiple commands in one file:

```yaml
commands:
  # Custom backup script
  backup:
    - value: database
      description: Backup database to S3
    - value: files
      description: Backup files to S3
    - value: restore
      description: Restore from backup
    - value: list
      description: List available backups

  # Custom build script
  build:
    - value: frontend
      description: Build frontend assets
    - value: backend
      description: Build backend binaries
    - value: all
      description: Build all components
    - value: clean
      description: Clean build artifacts
```

#### Extending Existing Tools

Add additional completions for tools that already have embedded completions:

```yaml
commands:
  # Extend docker completions with custom aliases or less common commands
  docker:
    - value: prune-all
      description: Prune containers, images, volumes, and networks
    - value: system
      description: Manage Docker system

  # Add kubectl shortcuts
  kubectl:
    - value: ctx
      description: Switch context
    - value: ns
      description: Switch namespace
```

#### Options and Flags

Include common flags and options:

```yaml
commands:
  myapp:
    - value: start
      description: Start the application
    - value: stop
      description: Stop the application
    - value: --config
      description: Specify config file path
    - value: --verbose
      description: Enable verbose logging
    - value: --port
      description: Set listening port
    - value: -h
      description: Show help
    - value: -v
      description: Show version
```

### Best Practices

1. **Clear Descriptions**: Write concise, helpful descriptions (aim for 50 characters or less)
2. **Alphabetical Order**: Sort completions alphabetically by value for easier maintenance
3. **Common First**: If you have logical grouping, put most-used commands first
4. **Include Flags**: Add both short (`-h`) and long (`--help`) flag variants
5. **Consistent Style**: Match the style of embedded completions for familiar tools
6. **Avoid Duplicates**: If overriding embedded completions, only include what's different

### Testing Your Completions

After creating your completion file:

1. **Reload bishop**: Start a new bishop session to load the completions
   ```bash
   bish
   ```

2. **Test Tab Completion**: Type your command and press TAB
   ```bash
   deploy <TAB>
   ```

3. **Check for Errors**: If completions don't appear, check:
   - File syntax (use `yamllint` or an online YAML validator)
   - File location (must be in one of the supported paths)
   - File permissions (must be readable)

### Troubleshooting

#### Completions Not Appearing

**Check file syntax:**
```bash
# Install yamllint if not already installed
brew install yamllint  # macOS
# or
apt install yamllint   # Ubuntu/Debian

# Validate your file
yamllint ~/.config/bish/completions.yaml
```

**Check file location:**
```bash
# Verify the file exists
ls -la ~/.config/bish/completions.yaml

# Check permissions
chmod 644 ~/.config/bish/completions.yaml
```

**Enable debug mode** (if supported by bishop):
Check bishop logs for completion loading errors.

#### YAML Syntax Errors

Common mistakes:

```yaml
# ❌ Wrong - missing description key
commands:
  docker:
    - value: build
      Build an image  # Missing 'description:' key

# ✅ Correct
commands:
  docker:
    - value: build
      description: Build an image

# ❌ Wrong - incorrect indentation
commands:
docker:
  - value: build
    description: Build an image

# ✅ Correct - consistent 2-space indentation
commands:
  docker:
    - value: build
      description: Build an image
```

#### Overrides Not Working

If your custom completions aren't overriding embedded ones:

1. Ensure you're using the exact same command name (case-sensitive)
2. Restart bishop to reload configuration
3. Check that your file is in the correct location
4. Verify YAML syntax is valid

### JSON Format Alternative

If you prefer JSON over YAML:

```json
{
  "commands": {
    "deploy": [
      {
        "value": "staging",
        "description": "Deploy to staging environment"
      },
      {
        "value": "production",
        "description": "Deploy to production environment"
      }
    ]
  }
}
```

Save as `~/.config/bish/completions.json` instead.

### Real-World Example

Here's a complete example for a typical development workflow:

```yaml
commands:
  # Project management CLI
  proj:
    - value: create
      description: Create new project from template
    - value: list
      description: List all projects
    - value: switch
      description: Switch to different project
    - value: delete
      description: Delete a project

  # Database migration tool
  migrate:
    - value: up
      description: Run pending migrations
    - value: down
      description: Rollback last migration
    - value: create
      description: Create new migration file
    - value: status
      description: Show migration status
    - value: reset
      description: Reset database and rerun all migrations

  # Development server
  dev:
    - value: start
      description: Start development server
    - value: stop
      description: Stop development server
    - value: restart
      description: Restart development server
    - value: logs
      description: View server logs
    - value: --port
      description: Specify port number
    - value: --debug
      description: Enable debug mode
```

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
