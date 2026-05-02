# Contributing to GoShort

Thank you for your interest in contributing! GoShort welcomes contributions of all kinds: bug reports, feature requests, documentation, and code.

## Quick Start

```bash
# Fork and clone
git clone https://github.com/<your-username>/goshort.git
cd goshort

# Install dependencies
go mod download

# Setup git hooks (requires lefthook)
lefthook install

# Run tests
make test

# Run linter
make lint

# Build
make build
```

## Development Workflow

1. **Check existing issues** — look for `good first issue` or `help wanted` labels.
2. **Open an issue first** for non-trivial changes. Discuss the approach before writing code.
3. **Fork and branch** — create a feature branch from `main`: `git checkout -b feat/my-feature`
4. **Write tests first** — we practice TDD. Write a failing test, then make it pass.
5. **Follow code conventions** — see below.
6. **Submit a PR** — fill out the PR template. Reference the issue.

## Code Conventions

- **Go style:** follow `gofmt`, `golangci-lint`, and `.editorconfig` (tabs for Go, 2-space for YAML/JSON/Markdown)
- **Error handling:** wrap with `fmt.Errorf("context: %w", err)`, handle errors once
- **Testing:** table-driven tests with `t.Run`, `t.Parallel()` where possible, `httptest` for HTTP handlers
- **Interfaces:** defined in the consumer package, not the implementation package
- **Commits:** Conventional Commits with gitmoji: `✨ feat(api): add URL expiration support`

## What Makes a Good PR

- Solves one thing (don't mix features with refactors)
- Includes tests for new behavior
- Passes `make lint && make test`
- Has a clear description of what and why
- References the related issue

## Architecture

See [docs/DESIGN.md](docs/DESIGN.md) for the full system design.

**Key rule:** dependencies point inward.

```
api/ → shortener/ ← storage/
              ↑
           cache/
```

The `shortener/` package defines the `Storage` and `Cache` interfaces. The `api/` package calls the service. Never bypass the interface.

## Running CI Locally

```bash
make lint        # golangci-lint
make test        # go test ./...
make build       # build both binaries
```

## Questions?

Open a [Discussion](https://github.com/anIcedAntFA/goshort/discussions) or file an issue. Happy to help!
