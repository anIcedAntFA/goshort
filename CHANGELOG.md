# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.4.0] - 2026-05-03

### Added

- **MCP server** — AI agent integration via [Model Context Protocol](https://modelcontextprotocol.io)
  using the official [`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk) v1.6.0
  - **Dual transport** — stdio (`--mcp` flag for local Claude Code / Cursor) and
    Streamable HTTP (`--mcp-http :9090` for remote agents)
  - **5 tools** — `shorten_url`, `list_urls`, `get_url_stats`, `delete_url`, `lookup_url`
  - **2 resources** — `goshort://stats/summary` (total count + top URLs),
    `goshort://urls/{code}` (full details for a short code)
  - **2 prompts** — `shorten_and_share` (platform-aware sharing copy),
    `batch_shorten` (multi-URL workflow)
  - **API key auth** — `APIKeyMiddleware` wraps the HTTP transport with constant-time
    `X-API-Key` comparison; auth disabled when key is empty
  - **MCP on main port** — `/mcp` now served on the main HTTP server (port 8080)
    alongside the REST API via `mcp.Server.HTTPHandler(apiKey)`; auth is handled
    inside the handler — `/mcp` intentionally bypasses Chi's rate-limit middleware
    (MCP sessions are long-lived; per-request rate limiting would break them)
- **`.mcp.json`** — ready-to-use MCP config for Claude Code (stdio only; remote HTTP
  now uses the main server at `https://goshort.app/mcp`)
- **`[mcp]` config section** — `mcp.base_url` override (falls back to `server.base_url`)
  configurable via `GOSHORT_MCP_BASE_URL` env var

### Changed

- `cmd/server/main.go` — extracted `runMCPMode` and `runHTTPServer` helpers to keep
  cyclomatic complexity within lint limits (gocognit ≤ 15)
- Cloudflare cache Rule 2 updated to bypass `/mcp` — long-lived MCP SSE sessions
  must not be served from CDN cache
- `.mcp.json` remote entry removed — `url`+`headers` format is Claude Desktop only
  (not supported in Claude Code project `.mcp.json`)

## [0.3.1] - 2026-05-03

### Fixed

- **Redis URL format** — `NewRedisCache` now accepts both `redis://localhost:6379` (URL scheme)
  and plain `host:port` — fixes a bug where the config default (`cache.redis_url = "redis://localhost:6379"`)
  was rejected by the Redis client which expected bare `host:port` format

### Added

- **Local Redis dev setup** — `docker-compose.dev.yml` with a throwaway Redis container
  (no persistence, pure in-memory) for running integration tests locally
  - `make dev/redis` / `make dev/redis/stop` — start/stop the container
  - `make test/redis` — run the full suite with `-tags redis`
  - `make test/all` — auto-detects a running Redis instance and adjusts test tags accordingly
- **Compile-time interface checks** — `var _ Cache = (*T)(nil)` assertions added to
  `MemoryCache` and `NoopCache` (already present in `RedisCache`)
- **Binaries output to `bin/`** — `make build` now places `bin/goshort` and `bin/goshort-cli`
  in a dedicated subdirectory instead of the project root

### Changed

- Redis integration tests append `t.Name()` to all cache keys to prevent collisions under `-race`
- CLI documentation corrected: `shorten` takes a URL argument directly — stdin piping is not supported
- Demo tape fixed: binary paths updated to `bin/`, build step hidden with `Hide`/`Show`, server logs
  redirected to `/tmp/goshort.log` to avoid interleaving with recorded output

## [0.3.0] - 2026-05-03

### Added

- **Authentication** — API key middleware with constant-time comparison
  (`X-API-Key` header) for all `/api/v1/*` write endpoints
  - Empty key = auth disabled (backward compatible with unauthenticated deployments)
  - Returns 401 with structured JSON error body for missing or invalid keys
- **Rate Limiting** — Per-IP token bucket using `golang.org/x/time/rate`
  - Configurable requests per minute via `rate_limit.requests_per_minute`
  - Background goroutine cleans stale IP entries (5 min sweep, 10 min idle TTL)
  - Returns 429 with `Retry-After: 60` header on limit exceeded
- **CLI Client** (`goshort-cli`) — standalone HTTP client for GoShort server
  - Commands: `shorten`, `list`, `stats`, `delete`, `version`
  - Pipe support: `echo "https://..." | goshort-cli shorten`
  - Config file: `~/.goshort.toml` with `server_url` and `api_key`
  - Flag → env var → config file → default resolution order
- **Docker** — Production-ready containerization
  - Multi-stage Dockerfile (~49MB Alpine image, non-root `goshort` user)
  - Docker Compose with Caddy reverse proxy (auto-TLS via `local_certs`)
  - Health check via `wget --spider /health`
  - `Dockerfile.goreleaser` for GoReleaser pre-built binary injection
- **Release Infrastructure** — Automated cross-platform releases
  - GoReleaser v2 builds server + CLI for linux/darwin/windows × amd64/arm64
  - Multi-arch Docker image published to `ghcr.io/anicedantfa/goshort` via buildx
  - GitHub Actions release workflow triggered on `v*` tag push
- **Community Files** — Opensource readiness
  - `CONTRIBUTING.md` with development workflow and commit conventions
  - `SECURITY.md` with vulnerability reporting policy and scope table
  - GitHub issue templates for bug reports and feature requests (YAML form)
  - PR template with what/why/how checklist
  - `CODEOWNERS` for automatic reviewer assignment on security-sensitive paths

## [0.2.1] - 2026-05-02

### Changed

- Fixed env var mapping for multi-word config fields (`GOSHORT_CACHE_REDIS_URL`)
- Replaced `ORDER BY created_at` with `ORDER BY id DESC` in ListURLs (faster, deterministic)
- Removed cache dependency from service layer; cache is now delivery-layer only
- Reduced Prometheus label cardinality (unmatched routes use `unmatched` label)

### Fixed

- Suppressed gosec G710 false positives on validated redirect handler

## [0.2.0] - 2026-05-02

### Added

- **HTTP API** — Chi v5 router with full CRUD for short URLs
  - `POST /api/v1/urls` — create with optional alias and expiry
  - `GET /api/v1/urls` — paginated list with click counts
  - `GET /api/v1/urls/:code` — details for a single URL
  - `DELETE /api/v1/urls/:code` — permanent deletion with cache invalidation
  - `GET /:code` — 302 redirect with click tracking; 410 for expired URLs
- **Caching** — cache-aside pattern with pluggable backends
  - `none` (default), `memory` (sync.Map, lazy TTL), `redis` (go-redis/v9)
  - Cache TTL capped to `min(24h, remaining_expiry)` — expired URLs never served from cache
- **Configuration** — Koanf v2 with layered resolution
  - TOML file auto-discovered at `./goshort.toml` or `$XDG_CONFIG_HOME/goshort/goshort.toml`
  - Environment variable override: `GOSHORT_<SECTION>_<KEY>`
- **Observability** — Prometheus metrics at `/metrics` + `slog` structured logging
  - Business counters: `goshort_urls_created_total`, `goshort_redirects_total`
  - Logging middleware skips `/health` to reduce noise
- **Interactive API docs** — OpenAPI 3.1 spec served as Scalar UI at `/docs`
- **Bruno API tests** — 10 `.bru` files covering all endpoints with environment support

## [0.1.0] - 2026-05-02

### Added

- **Core library** — URL shortening with zero-collision code generation
  - Atomic SQLite counter feeds Sqids for non-sequential, bijective short codes
  - Generated codes use `[a-zA-Z0-9]`; custom aliases allow hyphens — charset
    differentiation makes collisions impossible by construction
- **SQLite storage** — sqlc-generated type-safe queries, pure Go driver (no CGO)
  - `urls` table with index on `short_code`; `counter` table for Sqids seed
  - Lazy expiry check on read + periodic background cleanup
- **URL validation** — scheme allowlist, 2048-char limit, private/loopback IP blocking
- **In-memory cache** — `sync.Map` with lazy TTL; `NoopCache` for disabled state
- **Test suite** — table-driven unit tests, `httptest` integration tests, fuzz tests
  - 91%+ coverage on `shortener/` and `config/` packages

---

[unreleased]: https://github.com/anIcedAntFA/goshort/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/anIcedAntFA/goshort/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/anIcedAntFA/goshort/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/anIcedAntFA/goshort/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/anIcedAntFA/goshort/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/anIcedAntFA/goshort/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/anIcedAntFA/goshort/releases/tag/v0.1.0
