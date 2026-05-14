# Contributing to GoShort

Thank you for your interest in contributing! GoShort welcomes contributions of all kinds: bug reports, feature requests, documentation, and code.

## Quick Start

```bash
# Fork and clone
git clone https://github.com/<your-username>/goshort.git
cd goshort

# Install Go dependencies
go mod download

# Install website dependencies (requires Bun)
cd website && bun install && cd ..

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

## Website Development

The landing page lives in `website/` — Astro v6 static site deployed on Cloudflare Workers at `goshort.ngockhoi96.dev`.

```bash
make website/dev    # start Astro dev server on localhost:4321
make website/check  # Biome lint + format (auto-fixes in place)
make website/build  # build to website/dist/
```

- **Linter:** Biome v2 (`biome.jsonc`). The pre-commit hook runs `bun run check` on staged website files automatically.
- **Tailwind:** v4 via `@tailwindcss/vite` plugin — no `tailwind.config.*`, tokens live in `src/styles/global.css` (`@theme` block).
- **API calls from widget:** default `API_BASE` is `https://goshort.app`. Override with `PUBLIC_API_BASE=http://localhost:8080 bun run dev` to point at a local Go server.
- **Type checking:** `bun run astro check` catches Astro template type errors.

## Code Conventions

- **Go style:** follow `gofmt`, `golangci-lint`, and `.editorconfig` (tabs for Go, 2-space for YAML/JSON/Markdown)
- **Error handling:** wrap with `fmt.Errorf("context: %w", err)`, handle errors once
- **Testing:** table-driven tests with `t.Run`, `t.Parallel()` where possible, `httptest` for HTTP handlers, `mcp.NewInMemoryTransports()` for MCP tool tests
- **Interfaces:** defined in the consumer package (`shortener/`), not the implementation package
- **Commits:** Conventional Commits with gitmoji: `✨ feat(api): add URL expiration support`

## Architecture

See [docs/DESIGN.md](docs/DESIGN.md) for the full system design.

**Key rule:** dependencies point inward.

```
         ┌──────────────┐
         │  Delivery    │
         │  api/ + mcp/ │
         └──────┬───────┘
                │ calls Service interface
         ┌──────▼───────┐
         │  Service     │
         │  shortener/  │  ← defines Storage, Cache, Encoder interfaces
         └──┬───────┬───┘
            │       │ satisfies interfaces
     ┌──────▼──┐ ┌──▼────────┐
     │ storage/│ │ cache/    │
     │ encoder/│ │ noop/mem/ │
     └─────────┘ │ redis     │
                 └───────────┘
```

Cache-aside logic lives in the delivery layer (`api/handler.go`), not in the service. The `cmd/server/main.go` is the only file that knows all concrete types — it wires everything together via constructor injection.

## What Makes a Good PR

- Solves one thing (don't mix features with refactors)
- Includes tests for new behavior
- Passes `make lint && make test`
- Has a clear description of what and why
- References the related issue

## Running CI Locally

```bash
# Go
make lint        # golangci-lint
make test        # go test ./...
make build       # build both binaries

# Website
make website/check   # Biome lint + format
make website/build   # Astro static build
```

## Questions?

Open a [Discussion](https://github.com/anIcedAntFA/goshort/discussions) or file an issue. Happy to help!
