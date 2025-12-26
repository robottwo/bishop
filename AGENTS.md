# Agent Instructions

## Setup Requirements

Before starting any work, ensure the development environment is properly configured:

### Install Git Hooks

**Always run this first** to set up the pre-commit hook that runs linters, tests, and vulnerability checks:

```bash
make install-hooks
```

This installs a pre-commit hook that runs:
1. `golangci-lint` - Code linting
2. `go test ./...` - Unit tests
3. `govulncheck` - Security vulnerability checks

If you don't have the required tools installed, run:

```bash
make tools
```

This installs:
- `govulncheck` - Security vulnerability scanner
- `golangci-lint` - Go linter
- Checks for `gh` (GitHub CLI) - needed for PR and issue operations

### Verify Your Changes

Before committing, you can manually run the full CI suite:

```bash
make ci
```

This runs: `lint` → `vulncheck` → `test` → `build`

### Important

- **Never skip the pre-commit hook** (`--no-verify`) unless absolutely necessary
- If the hook fails, fix the issues before committing
- Run `make ci` to verify everything passes before pushing

## Git Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/#specification) types for **branch names**, **commit messages**, and **PR titles**.

### Format

#### Branch Names
Use forward slashes (`/`):
- **Basic:** `<type>/<description>`
- **With scope:** `<type>(<scope>)/<description>`

#### Commit Messages & PR Titles
Use colons (`:`) followed by a space:
- **Basic:** `<type>: <description>`
- **With scope:** `<type>(<scope>): <description>`
- **Breaking change:** `<type>!: <description>`

### Rules

- **kebab-case** descriptions (50 chars max)
- Present tense ("add", not "added")
- Be concise but clear

### Types

- `feat`: New feature
- `fix`: Bug fix  
- `docs`: Documentation
- `style`: Formatting only
- `refactor`: Code improvement
- `perf`: Performance boost
- `test`: Tests
- `build`: Build system
- `ci`: CI configuration
- `chore`: Other changes
- `revert`: Previous commit

### Scopes (Optional)

**Common scopes:**
- `auth` - Authentication
- `ui` - User interface
- `api` - API endpoints
- `config` - Configuration
- `deps` - Dependencies
- `test` - Testing

### Examples

**Basic:**
- `feat/user-login`
- `fix/memory-leak`
- `docs/update-readme`

**Scoped:**
- `feat(auth)/oauth-support`
- `fix(ui)/mobile-layout`
- `docs(api)/endpoints`

**Breaking:**
- `feat!: remove-deprecated-api`

### Workflow

1. Create branch per logical work unit
2. Make focused commits
3. Follow commit conventions
4. Push and PR
5. Delete after merge

### Mistakes to Avoid

❌ `feat/UserLogin` (caps)  
❌ `fix/memory_leak` (underscores)  
❌ `feature/login` (wrong type)  
❌ `feat/very-long-description-exceeding-limit`  
❌ `fix/` (no description)  

✅ `feat/user-login`  
✅ `fix/memory-leak`  
✅ `feat(auth)/oauth`
