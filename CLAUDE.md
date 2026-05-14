# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GoShort is a self-hosted URL shortener built in Go — a dual-purpose project: a practical tool and a learning vehicle for Go, system design, caching, and MCP/AI integration.

**Current state:** Phase 6 M6.6 done — landing page deployed at [goshort.ngockhoi96.dev](https://goshort.ngockhoi96.dev) (Option B subdomain split; `goshort.app` stays on Fly.io). Phase 5 shipped batch, QR codes, link previews, expiry update, spam detection (7 tools, 3 resources, 2 prompts). API + redirects at [goshort.app](https://goshort.app).

## Commands

```bash
# Build
make build                           # Build bin/goshort + bin/goshort-cli
go build ./cmd/server/...            # Server only
go build ./cmd/cli/...               # CLI only

# Test
make test                            # go test ./...
make test/race                       # With race detector
make test/cover                      # HTML coverage report
make test/redis                      # Include Redis integration tests (-tags redis)
make test/all                        # Auto-detect Redis, run accordingly
go test -run TestServiceCreate ./... # Single test
go test -fuzz FuzzValidator ./...    # Fuzz testing

# Lint
make lint                            # golangci-lint run
make lint/fix                        # With auto-fix

# Code generation
sqlc generate                        # Regenerate type-safe Go from SQL

# Migrations
make migrate                         # Run goose migrations against ./data/goshort.db
make migrate/status                  # Show applied/pending migrations

# Docker
docker compose up -d                 # Production stack (GoShort + Caddy)
make dev/redis                       # Throwaway Redis for integration tests
make dev/redis/stop                  # Stop Redis

# Deploy
fly deploy                           # Deploy to Fly.io

# Website (Astro)
make website/dev                     # Start Astro dev server (bun run dev)
make website/build                   # Build static site to website/dist/
make website/check                   # Lint + format with Biome (bun run check)
```

## Architecture

Three-layer pragmatic structure — avoid over-engineering:

```
cmd/
├── server/main.go     # HTTP server + MCP wiring entry point
└── cli/main.go        # CLI entry point (Cobra)

internal/
├── shortener/         # Core business logic (service, validator, model, errors)
├── encoder/           # Sqids-based short code generation
├── storage/           # Storage interface + SQLite (sqlc) implementation
├── cache/             # Cache interface + noop/memory/redis implementations
├── api/               # Chi HTTP handlers, router, middleware, error types
├── mcp/               # MCP server: tools, resources, prompts, auth
├── config/            # Koanf config loading (goshort.toml + env vars)
└── db/                # sqlc-generated code (DO NOT EDIT)

db/
├── schema.sql         # Table definitions
├── migrations/        # goose versioned SQL migrations (embedded via go:embed)
├── queries.sql        # All SQL queries (sqlc input)
└── sqlc.yaml          # sqlc config

website/
├── src/
│   ├── worker.ts      # Cloudflare Worker entry point (pure static-asset handler)
│   ├── layouts/       # base-layout.astro (HTML shell, SEO, theme script)
│   ├── components/    # navbar, footer, theme-toggle, shorten-widget
│   ├── pages/         # index.astro (output: 'static')
│   ├── styles/        # global.css (@theme tokens, component classes)
│   └── lib/           # dom.ts (typed data-* query helpers)
├── public/            # robots.txt, favicon.svg
└── wrangler.jsonc     # Cloudflare Workers config (route: goshort.ngockhoi96.dev)
```

**Layer boundaries:** `api/` and `mcp/` call `shortener.Service` interface; `shortener/` calls `Storage` and `Encoder` interfaces only — never concrete types. Cache-aside is a delivery-layer concern (in `api/handler.go`), not in the service. `cmd/server/main.go` is the only file that knows all concrete types (DI wiring point).

## Key Design Decisions

Consult `docs/DESIGN.md` for full rationale. Critical decisions:

- **Short code generation:** Counter + Sqids library (zero collision, non-sequential). Generated codes use `[a-zA-Z0-9]` only; custom aliases allow hyphens (`my-link`) — charset differentiation prevents collision.
- **Database:** SQLite via sqlc (Phases 1–4), PostgreSQL (Phase 5+). Two tables: `urls` and `counter`.
- **Caching:** Cache-aside pattern in delivery layer. `cache.driver` config: `none | memory | redis`. Interface must be satisfied by all three.
- **Redirect strategy:** 302 (temporary) not 301 — 301 is browser-cached permanently and kills click analytics.
- **HTTP router:** Chi v5 (stdlib-compatible, no lock-in).
- **Logging:** `slog` stdlib only — zero dependencies.
- **Auth:** API key with constant-time comparison (Phase 3). MCP `/mcp` endpoint uses its own `APIKeyMiddleware`, bypasses Chi rate limit middleware (MCP sessions are long-lived).
- **MCP:** Official `modelcontextprotocol/go-sdk` v1.6. Dual transport: stdio (`--mcp`) for local Claude Code/Cursor, Streamable HTTP on `/mcp` (same port as REST API) for remote agents.
- **Link previews:** `PreviewFetcher` interface (consumer-package pattern); `HTTPPreviewFetcher` fetches `<title>`/`<meta description>` on create (3 s timeout, 512 KB cap, fail-open, blocks private IPs). Use `NoopPreviewFetcher` in tests.
- **Spam detection:** `URLChecker` interface; `SafeBrowsingChecker` calls Google Safe Browsing v4 — **fail-open**: any API error returns nil so URL creation is never blocked. `NoopChecker` used when no key is configured.
- **Schema migrations:** goose embedded FS — `db/migrations/*.sql` versioned files applied automatically at server startup via `goose.Up`.
- **Test-only constructors:** `export_test.go` (package `shortener`, not `shortener_test`) exposes internal fields to external test packages without polluting the public API.
- **Tailwind v4 `@apply` limitation:** Cannot `@apply` custom component classes inside other custom classes — only utility classes. Use the two-class pattern in HTML: `class="btn btn-primary"`, where `.btn` provides structure and `.btn-primary` adds colour only.
- **Biome + Astro false positives:** `noUnusedVariables` and `noUnusedImports` fire on frontmatter vars/imports used in Astro templates. Suppressed via `overrides` in `website/biome.jsonc`. Do not remove that override.
- **Public endpoint (`POST /api/v1/urls/public`):** No auth required, 5 req/min via isolated `r.Group()`, honeypot field `website` (non-empty → fake 201 with `short_code:"decoy"`), forced 30-day expiry, no custom alias allowed.
- **CORS for public endpoint:** `CORSMiddleware()` with `Access-Control-Allow-Origin: *` — appropriate since the endpoint has no auth and is rate-limited. Must also register `r.Options("/urls/public", ...)` in the same group or Chi returns 405 before the middleware runs.
- **Website deployment (Option B subdomain split):** Landing page at `goshort.ngockhoi96.dev` (Cloudflare Worker, pure static). `goshort.app` stays on Fly.io for API + redirects + MCP. Worker serves only `env.ASSETS.fetch(request)` — no proxy logic. Shorten widget calls `https://goshort.app/api/v1/urls/public` explicitly; override with `PUBLIC_API_BASE` env var for local dev.

## Phased Roadmap

| Phase | Focus | Status |
|-------|-------|--------|
| 1 | Core library: SQLite, sqlc, Sqids, TDD | ✅ v0.1.0 |
| 2 | HTTP API (Chi), caching, slog, Prometheus | ✅ v0.2.0 |
| 3 | Auth, CLI (Cobra), rate limiting, Docker, Fly.io | ✅ v0.3.0 |
| 3.5 | Fly.io deploy + Cloudflare DNS/CDN | ✅ goshort.app |
| 4 | MCP server (official Go SDK, stdio + HTTP) | ✅ v0.4.0 |
| 5 | Batch, QR codes, link previews, expiry update, spam detection | ✅ v0.5.0 |
| 6  | Landing page (Astro + Cloudflare Worker), public endpoint | ✅ v0.6.0 |
| 7+ | Analytics, PostgreSQL, Redis counter | 🔲 |

## Technology Stack

| Concern | Library |
|---------|---------|
| HTTP | `go-chi/chi/v5` |
| DB access | `sqlc-dev/sqlc` (generates type-safe Go from SQL) |
| Short codes | `sqids-org/sqids-go` |
| CLI | `spf13/cobra` |
| Config | `knadh/koanf/v2` + TOML parser + env provider |
| Cache (Redis) | `redis/go-redis/v9` |
| Metrics | `prometheus/client_golang` |
| Rate limiting | `golang.org/x/time/rate` |
| MCP server | `modelcontextprotocol/go-sdk` (official, v1.6.0) |
| Migrations | `pressly/goose/v3` (embedded FS, versioned SQL) |
| QR codes | `skip2/go-qrcode` |
| Spam detection | Google Safe Browsing Lookup API v4 (optional, fail-open) |
| API testing | Bruno (`.bru` files in `api-tests/`) |
| Website | Astro v6, `@tailwindcss/vite` v4, `@biomejs/biome` v2, Bun |
| Cloudflare Worker | Pure static asset handler at `goshort.ngockhoi96.dev` (Option B subdomain split) |

## Configuration

App reads `goshort.toml` at runtime (loaded via Koanf):

```toml
[server]
port = 8080
base_url = "https://short.yourdomain.com"

[storage]
driver = "sqlite"
sqlite_path = "./data/goshort.db"

[cache]
driver = "none"   # none | memory | redis

[auth]
api_key = ""

[rate_limit]
enabled = false
requests_per_minute = 60

[shortener]
code_length = 6
default_expiry = "0"

[logging]
level = "info"
format = "json"

[mcp]
base_url = ""     # override for MCP responses; falls back to server.base_url

[security]
safe_browsing_api_key = ""  # empty = disabled; env: GOSHORT_SECURITY_SAFE_BROWSING_API_KEY
```

## Code Conventions

- Tabs for Go files, 2-space spaces for YAML/JSON/Markdown (enforced by `.editorconfig`)
- Error handling: wrap with `%w` for `errors.Is/As`; handle errors once, don't log-then-return
- Interfaces defined in the consumer package (`shortener.Storage`, `shortener.Cache`, `shortener.Encoder`) — not in the implementation package
- Table-driven tests with `t.Run` subtests; use `httptest` for HTTP handler tests; MCP tests use `mcp.NewInMemoryTransports()` for in-process testing
- All SQL lives in `db/queries.sql`; never write raw SQL strings in Go code
- Config struct tags use `koanf:"field_name"` (not `mapstructure`)

## Reference Documentation

- `docs/DESIGN.md` — Full system design, API spec, data model, scaling rationale, risk table
- `docs/LEARNING.md` — Go philosophy, patterns, and GoShort-specific knowledge map
- `docs/DEPLOYMENT.md` — Fly.io, Cloudflare, Docker Compose deployment guide
- `docs/ROADMAP.md` — Task-level roadmap with checkboxes
