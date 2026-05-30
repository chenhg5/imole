# Contributing to iMole

## Setup

```bash
# Install development tools
brew install golangci-lint

# Install pre-commit hook (runs format/lint checks on every commit)
git config core.hooksPath .githooks
```

## Development

Build and test:

```bash
make build    # build for current platform
make release  # build for all platforms (macOS, Linux, Windows)
go test ./... # run tests
go vet ./...  # vet
```

Format code:

```bash
gofmt -w cmd internal
```

## Code Style

### Go Rules

- Run `gofmt -w cmd internal` before committing
- Use `go vet ./...` to check for issues
- Keep files focused on single responsibility
- Extract constants instead of magic numbers
- Use context for timeout control on external commands
- Add comments explaining **why** something is done, not just **what**

### Safety Rules

- All destructive commands must support `--dry-run`
- Delete only after verification — only delete files with `verified: true`
- Path validation must reject `..` traversal, relative paths, empty paths
- Operation logging through `internal/history/` must be preserved

## Requirements

- Go 1.21+
- macOS, Linux, or Windows (for cross-compilation)
- For USB device testing: macOS with iPhone connected

## Pull Requests

1. Fork and create branch from `main`
2. Make changes
3. Run checks: `make build && go test ./...`
4. Commit and push
5. Open PR targeting `main`

CI will verify formatting, build, and tests.
