# Contributing to bishop

Thanks for your interest in contributing. bishop is an actively maintained fork of the original gsh project, with a faster development cadence and a commitment to contributing improvements back upstream.

- Fork repository: https://github.com/robottwo/bishop
- Upstream attribution: see the About this fork section in README.md

Our goals:
- Move quickly while keeping the codebase maintainable and well-tested
- Keep changes upstream-friendly to ease publishing MRs back to the original project
- Maintain user trust with a clear review and release process

## Contents

- How we work
- Upstream contribution flow
- Development setup
- Branching, commits, and PRs
- Testing
- Documentation changes
- Releasing and versioning
- Contact and support

## How we work

- Faster iteration: We prioritize smaller, incremental PRs that can be reviewed and merged quickly.
- Compatibility by default: Prefer changes that are compatible with upstream gsh design and APIs.
- Testing is mandatory: PRs should include tests when adding or modifying behavior.

## Branch workflow

We use a two-tier branching model:

- **`uat`** (default branch): User Acceptance Testing branch
  - All feature branches merge here first via PR
  - This is where integration testing and validation happens
  - The default branch for new PRs
  
- **`main`**: Production-ready stable branch
  - Only accepts merges from `uat` via PR
  - Protected: requires PR, no direct pushes
  - Represents tested, release-ready code

**Workflow:**
1. Create feature branches from `uat`: `feature/your-feature` or `fix/your-fix`
2. Submit PR targeting `uat` (default)
3. After PR is merged to `uat` and validated, create PR from `uat` → `main`
4. Once merged to `main`, the feature is considered stable and release-ready

This ensures all code goes through integration testing in `uat` before reaching the stable `main` branch.

## Upstream contribution flow

We aim to keep bishop close to upstream and upstream-friendly.

1. Implement changes in bishop
   - Keep PRs focused and small.
   - Include tests and docs updates where applicable.
2. Evaluate upstreamability
   - If the change is generic and beneficial to upstream users, mark the PR description with Upstream-candidate.
3. Publishing upstream MRs
   - Maintainers will open upstream PRs (or invite authors to do so) with minimal rebasing needed.
   - Keep implementation neutral: avoid bishop-specific flags or branding in shared logic.
4. Syncing with upstream
   - We regularly rebase or merge upstream main into bishop main to minimize drift.
   - Conflicts are resolved in favor of keeping a clean, maintainable surface that can be upstreamed later.

Tips for upstreamable changes:
- Keep feature flags and environment variables generic.
- Avoid forking interfaces unless necessary; prefer extension points.
- Include rationale and examples in commit messages and PRs.

## Development setup

Requirements:
- Go 1.24.13+ on macOS or Linux
- Make
- pre-commit (optional, but recommended)

Clone and build:

```bash
git clone https://github.com/robottwo/bishop.git
cd bishop
make build
# binary at ./bin/bish
```

### Git Hooks

We provide a git hook to run linters and tests before commit. To set up the git hooks:

```bash
make install-hooks
```

This will install a `pre-commit` hook that runs:
- `golangci-lint`
- `go test`
- `govulncheck`

Ensure you have the required tools installed via `make tools`.

Useful docs:
- Getting started: docs/GETTING_STARTED.md
- Configuration: docs/CONFIGURATION.md
- Features: docs/FEATURES.md
- Roadmap: ROADMAP.md

## Branching, commits, and PRs

Branching:
- Create branches from `uat`: `feature/short-description` or `fix/short-description`.
- PRs should target `uat` by default (it's the default branch).

Commit messages:
- Prefer Conventional Commits style when possible (feat:, fix:, chore:, docs:, refactor:, test:, perf:).
- Keep subject concise; add detail in body if needed.

Pull Requests:
- Keep PRs focused: one logical change per PR.
- Include a brief summary, motivation, and any risks.
- Add tests and documentation updates when appropriate.
- Link related issues.

Review:
- Expect actionable feedback; we optimize for quick iteration.
- Address comments or explain tradeoffs; force-push is fine for your branch.

## Testing

Run tests:

```bash
go test ./...
```

Run specific package tests:

```bash
go test ./internal/agent/...
```

Add tests for:
- New features
- Bug fixes (including regression coverage)
- Edge cases around permissions, file operations, and environment handling

CI:
- We use GitHub Actions (or will enable it shortly). Keep PRs green and reproducible locally.

## Documentation changes

We keep README.md concise and link to focused docs:

- docs/GETTING_STARTED.md
- docs/CONFIGURATION.md
- docs/FEATURES.md
- AGENTS.md
- SUBAGENTS.md

Guidelines:
- Prefer adding details to the appropriate doc file rather than the README.
- Keep examples short and runnable.
- Update links when reorganizing docs; verify relative paths.

## Releasing and versioning

- Releases follow semantic versioning where possible.
- Changelog entries are generated from PR titles and commit messages; keep them descriptive.
- Release artifacts and distribution channels may evolve as the project grows.

## Contact and support

- Issues: https://github.com/robottwo/bishop/issues
- For upstream matters, include a note in the issue if you believe the change is an upstream candidate.

Thank you for helping improve bishop while keeping it aligned with the broader gsh ecosystem.