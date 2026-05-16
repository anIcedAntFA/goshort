# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.6.1] - 2026-05-16

### Added

- **QR multi-format** — `GET /api/v1/urls/{code}/qr` now accepts `?format=png|jpeg|svg`
  (v0.6.0 was PNG-only); response `Content-Type` changes accordingly;
  `JPEG` uses quality 90; `SVG` returns vector markup as `image/svg+xml`
- **Landing page — QR panel** — shorten widget result card now includes a collapsible QR panel:
  lazy-loads the QR image on first reveal, canvas-copy to clipboard, and three outlined
  download buttons (PNG / JPEG / SVG) whose combined height matches the QR image at every breakpoint
- **Landing page — FAQ** — animated accordion using `<details>` + JS-driven `max-height` transition;
  open and close both animate smoothly with inline-style + forced-reflow technique
- **Landing page — Sponsor banner** — support CTA with GitHub Sponsors link placed above FAQ

### Changed

- **Landing page — shorten widget** — CSS and TypeScript logic extracted to `shorten-widget.css`
  and `shorten-widget.ts`; CSS imported via frontmatter (not `<style>`) so
  `[data-theme="dark"]` selectors are not Astro-scoped and work correctly
- **Landing page — result card** — redesigned: Original URL shown first; Shortened URL with
  inline copy icon and `data-copied` feedback; Visit / QR / Copy action row with Lucide icons
- **Landing page — install section** — tabbed quick-start (Docker Compose, Binary, go install)
  + Configure & Run shown side-by-side in a two-column grid; "Full setup guide" link added
- **Landing page — nav highlight** — replaced `IntersectionObserver` with a scroll-based
  algorithm (`offsetTop ≤ scrollY + 120`); reliably highlights Features (first) and FAQ (last)
  across all scroll positions without tuning `rootMargin`
- **Landing page — mobile menu** — slide-in/out animation using `max-height + opacity +
  translateY(-6px → 0)` transition; replaces `display: none/block` snap
- **Landing page — section spacing** — all sections unified to `py-16`
  (hero was `py-20`); consistent vertical rhythm throughout
- **Landing page — accessibility** — Shiki `github-dark` comment color overridden
  (`#8b949e` → `#a0b6c8`, ratio 4.47:1 → ~6.2:1, WCAG AA); dark-mode active tab color
  changed from `#4f46e5` to `#a5b4fc` (ratio 2.4:1 → ~6.0:1, WCAG AA)

### Fixed

- **QR endpoint auth** — `GET /api/v1/urls/{code}/qr` was inside the authenticated
  router group; moved to the public group so it no longer returns 401 for unauthenticated
  callers (widget, browser direct access, sharing)
- **Shorten widget — validation** — client-side URL validation added; empty input and
  non-http(s) URLs show an inline error; error reserves a fixed `min-h-5` line so no
  layout shift occurs; typing clears the error immediately
- **Shorten widget — error dark mode** — CSS imported via frontmatter prevents Astro's
  scoped-style mechanism from attaching `data-astro-cid-*` to `[data-theme="dark"]` rules,
  which was breaking all dark-mode widget styles
- **FAQ — open animation flash** — `details.setAttribute('open')` previously made the CSS
  open-state fallback (`max-height: none`) flash visible before JS could set `max-height: 0`;
  fixed by immediately overriding with inline styles + forced reflow before starting transition
- **Navbar — mobile layout** — mobile grid columns and overlay z-index corrected

## [0.6.0] - 2026-05-15

### Added

- **Public shorten endpoint** — `POST /api/v1/urls/public` requires no authentication
  - 5 req/min per IP via an isolated Chi group (never shares the bucket with authenticated routes)
  - Honeypot field `website`: non-empty value returns a fake 201 with `short_code:"decoy"` to confuse bots silently
  - Forced 30-day expiry; custom alias not allowed on the public endpoint
  - HTTP 422 `unsafe_url` if Safe Browsing is enabled and flags the URL
- **Landing page** — Astro v6 static site deployed at [goshort.ngockhoi96.dev](https://goshort.ngockhoi96.dev)
  - **Shorten widget** — live form calling the public endpoint; honeypot included; copy-to-clipboard on result
  - **Tabbed code demo** — API (curl), CLI, and MCP usage examples with Shiki syntax highlighting
  - **Feature cards** — REST API, CLI Tool, MCP Tools, Secure by Default
  - **Install section** — 3-step guide (Download / Configure / Shorten) with inline code snippets
  - **Theme toggle** — Light / Dark / Auto with `localStorage` persistence; blocking inline script prevents flash-of-wrong-theme
  - **Responsive layout** — mobile-first; hamburger menu with CSS-only X animation; IntersectionObserver active-link highlighting
  - Sitemap (`@astrojs/sitemap`), OpenGraph / Twitter Card meta, canonical URL, `favicon.svg`, `robots.txt`
- **Cloudflare Worker** — pure static-asset handler (`env.ASSETS.fetch(request)`, 8 lines)
  - **Option B subdomain split**: `goshort.ngockhoi96.dev` → Cloudflare Worker (static landing page only);
    `goshort.app` → Fly.io directly (API + redirects + MCP — unchanged, zero latency overhead)
  - Worker has no proxy logic; `goshort.app` DNS zone is untouched; rollback is deleting one CNAME
- **CORS middleware** — `CORSMiddleware()` with `Access-Control-Allow-Origin: *` scoped to the public endpoint group;
  `r.Options("/urls/public", ...)` registered in the same group so Chi routes preflight requests before responding 405
- **govulncheck CI workflow** — `.github/workflows/govulncheck.yml` scans Go dependencies against the
  Go vulnerability database on every push/PR to `main` and on a weekly Monday schedule
- **Website CI job** — `website` job in `ci.yml`: `setup-bun` → `bun install --frozen-lockfile` → `bun run lint` → `bun run build`
- **lefthook website-lint hook** — pre-commit Biome check (`bun run check --write`) runs on staged
  `*.ts`, `*.astro`, `*.css`, `*.json`, `*.mjs` files under `website/`; auto-fixes staged
- **FUNDING.yml** — GitHub Sponsors and Ko-fi support links (`github: anIcedAntFA`, `ko_fi: anIcedAntFA`)

### Changed

- README badges reorganised into two rows:
  - Row 1 (health): CI · govulncheck · Codecov · Go Report Card · Release
  - Row 2 (info): License · Go 1.26 · Website · API Docs · Ko-fi
- `SECURITY.md` supported versions updated: 0.6.x and 0.5.x active; < 0.5 no patches
- `CONTRIBUTING.md` — added **Website Development** section covering `bun run dev`, `bun run check`,
  `PUBLIC_API_BASE` override for local dev, and Tailwind/Biome conventions
- `.mcp.json` removed from repository (machine-specific path; README now shows example to create locally)
- `.gitignore` updated: `.wrangler/` and `.dev.vars*` excluded (Wrangler artifacts not committed)

## [0.5.1] - 2026-05-10

### Security

- **SSRF prevention** — preview fetcher validates resolved DNS IPs against private
  network ranges, not just literal hostnames
- **Open Graph support** — preview metadata parses `og:title` / `og:description`
  with HTML fallback

### Fixed

- GoReleaser Docker config updated from v1 `dockers_v2` to v2 `dockers`
- GitHub Actions goreleaser version pinned to `~> v2` (was `latest`)
- `lefthook.yml` commit-msg hook uses POSIX `grep -E` (fixes macOS)
- Removed dead `UpdateMetadata` sqlc query

### Changed

- `HTTPPreviewFetcher` moved to `internal/preview/`
- `SafeBrowsingChecker` moved to `internal/safebrowsing/`
- `APIClient` and `CLIConfig` extracted to `internal/cli/`
- `setupLogger` moved to `internal/config/`, `buildCache` to `internal/cache/`,
  `startCleanupJob` to `internal/storage/`
- `cmd/server/main.go` refactored to `func run() error` (~100 lines, pure wiring)
- MCP time formatting deduplicated via `formatUTC` / `formatUTCPtr` helpers
- Test files split by concern; integration tests added with real SQLite
- Makefile reorganized; added `make ci` and `make test/unit`

## [0.5.0] - 2026-05-10

### Added

- **Batch URL creation** — `POST /api/v1/urls/batch` accepts up to 100 URLs in one
  request; returns per-item results with individual success/failure status
- **QR code generation** — `GET /api/v1/urls/{code}/qr` returns a PNG QR code for any
  short URL; also exposed as `goshort://urls/{code}/qr` MCP resource
- **Link preview metadata** — `title` and `description` fields populated automatically
  on URL creation via `HTTPPreviewFetcher` (fetches `<title>` / `<meta description>`,
  3 s timeout, 512 KB cap, private-IP blocked, fail-open)
- **Expiry update** — `PATCH /api/v1/urls/{code}` updates the expiry of an existing
  short URL without changing its code or target
- **Spam detection** — optional Google Safe Browsing Lookup API v4 integration;
  unsafe URLs are rejected with HTTP 422 `unsafe_url`; fail-open (API errors never
  block creation); enabled by setting `security.safe_browsing_api_key` in config
  or `GOSHORT_SECURITY_SAFE_BROWSING_API_KEY` env var
- **MCP tools** — `batch_shorten_urls` and `update_url` added (7 tools total)
- **MCP resource** — `goshort://urls/{code}/qr` PNG QR code blob resource (3 resources total)
- **Schema migrations** — goose embedded FS (`db/migrations/*.sql`) applied automatically
  at startup; `make migrate` / `make migrate/status` commands added
- **`[security]` config section** — `safe_browsing_api_key` setting

### Changed

- `shortener.NewService` now accepts a 4th `URLChecker` parameter; pass `NoopChecker{}`
  when no spam detection is needed (all existing callers updated)
- `urls` table extended with `title`, `description`, and `preview_fetched_at` columns
  (migration `002_add_preview_metadata.sql`)
- OpenAPI spec updated to v0.5.0 with new endpoints, QR resource, batch operation,
  and `unsafe_url` error code

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

[unreleased]: https://github.com/anIcedAntFA/goshort/compare/v0.6.0...HEAD
[0.6.0]: https://github.com/anIcedAntFA/goshort/compare/v0.5.1...v0.6.0
[0.5.1]: https://github.com/anIcedAntFA/goshort/compare/v0.5.0...v0.5.1
[0.5.0]: https://github.com/anIcedAntFA/goshort/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/anIcedAntFA/goshort/compare/v0.3.1...v0.4.0
[0.3.1]: https://github.com/anIcedAntFA/goshort/compare/v0.3.0...v0.3.1
[0.3.0]: https://github.com/anIcedAntFA/goshort/compare/v0.2.1...v0.3.0
[0.2.1]: https://github.com/anIcedAntFA/goshort/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/anIcedAntFA/goshort/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/anIcedAntFA/goshort/releases/tag/v0.1.0
