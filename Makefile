.PHONY: all
all: ci

# === VHS Tape to GIF compilation ===
# Find all tape files and derive GIF names
TAPE_FILES := $(wildcard assets/tapes/*.tape)
GIF_FILES := $(patsubst assets/tapes/%.tape,assets/%.gif,$(TAPE_FILES))
BISH_DEMO_HOME := /tmp/bish-demo

# Pattern rule: compile .tape to .gif
assets/%.gif: assets/tapes/%.tape
	@echo "Recording $< -> $@"
	@vhs $<

.PHONY: tapes
tapes: tapes-setup
	# @$(MAKE) -j$(words $(GIF_FILES)) $(GIF_FILES) || ($(MAKE) tapes-cleanup && exit 1)
	@$(MAKE) $(GIF_FILES) || ($(MAKE) tapes-cleanup && exit 1)
	@$(MAKE) tapes-cleanup
	@$(MAKE) publish-docs
	@echo "All tapes compiled"

.PHONY: publish-docs
publish-docs:
	@echo "Publishing GIFs to docs/..."
	@cp assets/*.gif docs/
	@echo "✓ GIFs published to docs/"

.PHONY: tapes-setup
tapes-setup:
	@echo "Setting up isolated bish demo environment..."
	@rm -rf $(BISH_DEMO_HOME)
	@mkdir -p $(BISH_DEMO_HOME)
	@if [ -f ~/.bishrc ]; then cp ~/.bishrc $(BISH_DEMO_HOME)/.bishrc; fi

.PHONY: tapes-cleanup
tapes-cleanup:
	@rm -rf $(BISH_DEMO_HOME)

.PHONY: build
build:
	@echo "=== Building bishop ==="
	@VERSION=$$(cat VERSION) && echo "Building version v$$VERSION..." && \
	echo "Compiling..." && \
	go build -ldflags="-X main.BUILD_VERSION=v$$VERSION" -o ./bin/bish ./cmd/bish/main.go && \
	echo "✓ Compilation completed successfully!" && \
	echo "Binary created: ./bin/bish"

.PHONY: go-mod-verify
go-mod-verify:
	@echo "Verifying go mod tidy..."
	@go mod tidy

.PHONY: test
test:
	@go test -coverprofile=coverage.txt ./...

.PHONY: lint
lint:
	@echo "Running golangci-lint..."
	@golangci-lint run --timeout=5m

.PHONY: vulncheck
vulncheck:
	@echo "Running govulncheck..."
	@govulncheck ./...

.PHONY: ci
ci: lint go-mod-verify vulncheck test build

.PHONY: tools
tools:
	@echo "Installing tools..."
	@go install golang.org/x/vuln/cmd/govulncheck@latest
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.5.0
	@echo ""
	@echo "Checking for vhs (GIF recorder)..."
ifeq ($(OS),Windows_NT)
	@where vhs >nul 2>&1 && \
		echo "✓ vhs is already installed" || \
		(echo "⚠ vhs is not installed." && \
		echo "  Install it from: https://github.com/charmbracelet/vhs")
else
	@command -v vhs >/dev/null 2>&1 && \
		echo "✓ vhs is already installed" || \
		(echo "⚠ vhs is not installed." && \
		echo "  - macOS: brew install vhs" && \
		echo "  - Linux: See https://github.com/charmbracelet/vhs")
endif
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
		(echo "Installing GitHub CLI (gh)..." && \
		$(MAKE) install-gh)
endif

.PHONY: install-gh
install-gh:
	@GH_VERSION=$$(curl -s https://api.github.com/repos/cli/cli/releases/latest | grep '"tag_name"' | sed -E 's/.*"v([^"]+)".*/\1/') && \
	echo "Downloading gh v$$GH_VERSION..." && \
	curl -sLO "https://github.com/cli/cli/releases/download/v$${GH_VERSION}/gh_$${GH_VERSION}_linux_amd64.tar.gz" && \
	tar -xzf "gh_$${GH_VERSION}_linux_amd64.tar.gz" && \
	sudo mv "gh_$${GH_VERSION}_linux_amd64/bin/gh" /usr/local/bin/ && \
	rm -rf "gh_$${GH_VERSION}_linux_amd64" "gh_$${GH_VERSION}_linux_amd64.tar.gz" && \
	echo "✓ gh v$$GH_VERSION installed successfully"

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
