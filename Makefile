.PHONY: all
all: ci

.PHONY: build
build:
	@echo "=== Building bishop ==="
	@VERSION=$$(cat VERSION) && echo "Building version v$$VERSION..." && \
	echo "Compiling..." && \
	go build -ldflags="-X main.BUILD_VERSION=v$$VERSION" -o ./bin/bish ./cmd/bish/main.go && \
	echo "✓ Compilation completed successfully!" && \
	echo "Binary created: ./bin/bish"

.PHONY: test
test:
	@go test -coverprofile=coverage.txt ./...

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run

.PHONY: vulncheck
vulncheck:
	@echo "Running govulncheck..."
	@govulncheck ./...

.PHONY: ci
ci: lint vulncheck test build

.PHONY: tools
tools:
	@echo "Installing tools..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo ""
	@echo "Checking for GitHub CLI (gh)..."
ifeq ($(OS),Windows_NT)
	@where gh >nul 2>&1 && \
		echo "✓ gh is already installed" || \
		(echo "⚠ gh (GitHub CLI) is not installed." && \
		echo "  Install it from: https://cli.github.com/" && \
		echo "  - Windows: winget install GitHub.cli")
else
	@command -v gh >/dev/null 2>&1 && \
		echo "✓ gh is already installed" || \
		(echo "⚠ gh (GitHub CLI) is not installed." && \
		echo "  Install it from: https://cli.github.com/" && \
		echo "  - macOS: brew install gh" && \
		echo "  - Linux: See https://github.com/cli/cli/blob/trunk/docs/install_linux.md")
endif

.PHONY: install-hooks
install-hooks:
	@echo "Installing git hooks..."
ifeq ($(OS),Windows_NT)
	@if not exist .git\hooks mkdir .git\hooks
	@copy /Y githooks\pre-commit .git\hooks\pre-commit >nul
else
	@mkdir -p .git/hooks
	@cp githooks/pre-commit .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
endif
	@echo "Git hooks installed successfully."

.PHONY: clean
clean:
ifeq ($(OS),Windows_NT)
	@if exist bin rmdir /s /q bin
	@del /f /q coverage.out coverage.txt 2>nul
else
	@rm -rf ./bin
	@rm -f coverage.out coverage.txt
endif
