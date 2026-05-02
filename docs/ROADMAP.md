# 🗺️ ROADMAP.md — GoShort Project Roadmap

> **Role:** Tech Lead + Senior PM  
> **Workflow per task:** Discuss → Plan → Execute → Test → Verify → Commit  
> **Tracking:** Each task has a checkbox. Check it off when done.  
> **With Claude Code:** Copy a task block, paste as prompt, CC executes.

---

## Table of Contents

1. [Project Infrastructure & Tooling](#1-project-infrastructure--tooling)
2. [Phase 0: Project Skeleton](#phase-0-project-skeleton--1-day)
3. [Phase 1: Core Library + Storage](#phase-1-core-library--storage--2-weeks)
4. [Phase 2: HTTP API + Scaling Reads](#phase-2-http-api--scaling-reads--3-weeks)
5. [Phase 3: Auth + Rate Limiting + CLI](#phase-3-auth--rate-limiting--cli--2-weeks)
6. [Phase 3.5: Deploy + Infrastructure](#phase-35-deploy--infrastructure--1-week)
7. [Phase 4: MCP Server](#phase-4-mcp-server--1-week)
8. [Phase 5+: Future](#phase-5-future)

---

## 1. Project Infrastructure & Tooling

### Tooling Decisions

| Tool | Purpose | When to Add | Config File |
|------|---------|-------------|-------------|
| **golangci-lint** | Meta-linter (50+ linters, parallel, cached) | Phase 0 | `.golangci.yml` |
| **lefthook** | Git hooks manager (pre-commit: lint+test) | Phase 0 | `lefthook.yml` |
| **gitleaks** | Detect secrets (API keys, passwords) in commits | Phase 0 | `.gitleaks.toml` |
| **Makefile** | Task runner (build, test, lint, generate) | Phase 0 | `Makefile` |
| **.editorconfig** | Consistent editor settings across IDEs | Phase 0 | `.editorconfig` |
| **Codecov** | Test coverage tracking + PR comments | Phase 1 | `.github/workflows/ci.yml` |
| **GitHub Actions** | CI: lint, test, build on every push/PR | Phase 1 | `.github/workflows/` |
| **goreleaser** | Cross-platform binary releases + Docker images | Phase 3 | `.goreleaser.yml` |
| **LICENSE** | MIT license (opensource-friendly) | Phase 0 | `LICENSE` |
| **SECURITY.md** | Security vulnerability reporting policy | Phase 3 | `SECURITY.md` |
| **CHANGELOG.md** | Release notes (manual or auto from commits) | Phase 3 | `CHANGELOG.md` |
| **CODEOWNERS** | Auto-assign PR reviewers | Phase 3 (opensource) | `.github/CODEOWNERS` |
| **Issue templates** | Bug report + feature request templates | Phase 3 (opensource) | `.github/ISSUE_TEMPLATE/` |

### Versioning Strategy

```
Semantic Versioning: MAJOR.MINOR.PATCH

v0.1.0  — Phase 1 done (core library, tests pass)
v0.2.0  — Phase 2 done (HTTP API works)
v0.3.0  — Phase 3 done (auth, CLI, Docker)
v1.0.0  — Phase 3 complete, opensource release
v1.1.0  — Phase 4 (MCP)
v2.0.0  — Phase 5+ (breaking changes if any)

Tag format: v0.1.0
Branch: main (trunk-based development)
```

### Config File Specifications

**`.golangci.yml`** — Linter configuration:
```yaml
version: "2"
linters:
  default: standard
  enable:
    - errcheck        # check for unchecked errors
    - govet           # go vet on steroids
    - staticcheck     # comprehensive static analysis
    - unused          # detect unused code
    - gosimple        # simplify code
    - ineffassign     # detect useless assignments
    - gocritic        # opinionated code checks
    - gofumpt         # strict gofmt
    - errname         # error naming conventions
    - errorlint       # error wrapping checks
    - gosec           # security checks
    - bodyclose       # detect unclosed HTTP response bodies
    - noctx           # detect HTTP requests without context
linters-settings:
  govet:
    enable-all: true
  gofumpt:
    extra-rules: true
  errcheck:
    check-blank: true
issues:
  max-issues-per-linter: 50
  max-same-issues: 5
formatters:
  enable:
    - gofumpt
```

**`lefthook.yml`** — Git hooks:
```yaml
pre-commit:
  parallel: true
  commands:
    lint:
      glob: "*.go"
      run: golangci-lint run --fix
      stage_fixed: true
    test:
      glob: "*.go"
      run: go test -race -count=1 -timeout=30s ./...
    secrets:
      run: gitleaks git --no-banner --staged

pre-push:
  commands:
    full-test:
      run: go test -race -count=1 -timeout=60s ./...
    vet:
      run: go vet ./...

commit-msg:
  commands:
    conventional:
      run: |
        msg=$(cat {1})
        if ! echo "$msg" | grep -qE '^(feat|fix|refactor|test|docs|chore|ci|style|perf|build)(\(.+\))?: .+'; then
          echo "❌ Commit message must follow Conventional Commits format"
          echo "   Example: feat: add URL validation"
          echo "   Got: $msg"
          exit 1
        fi
```

**`Makefile`** — Task runner:
```makefile
.PHONY: build test lint generate clean run

# Build
build:
	go build -o bin/goshort ./cmd/server

build-cli:
	go build -o bin/goshort-cli ./cmd/cli

# Test
test:
	go test -race -count=1 -timeout=60s ./...

test-cover:
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

test-fuzz:
	go test -fuzz=Fuzz -fuzztime=30s ./internal/shortener/

# Lint
lint:
	golangci-lint run

lint-fix:
	golangci-lint run --fix

# Code generation
generate:
	sqlc generate

# Run
run:
	go run ./cmd/server

# Clean
clean:
	rm -rf bin/ coverage.out coverage.html

# Docker
docker-build:
	docker build -t goshort .

docker-up:
	docker compose up -d

docker-down:
	docker compose down

# All checks (CI equivalent)
ci: lint test build
```

**`.editorconfig`**:
```ini
root = true

[*]
charset = utf-8
end_of_line = lf
indent_style = tab
indent_size = 4
insert_final_newline = true
trim_trailing_whitespace = true

[*.{yml,yaml,json,toml}]
indent_style = space
indent_size = 2

[*.md]
trim_trailing_whitespace = false

[Makefile]
indent_style = tab
```

**`.gitleaks.toml`**:
```toml
title = "GoShort Gitleaks Config"

[allowlist]
  paths = [
    '''go\.sum''',
    '''vendor/''',
    '''docs/''',
  ]
```

### GitHub Actions Workflows

**`.github/workflows/ci.yml`** — Run on every push/PR:
```yaml
name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: golangci/golangci-lint-action@v6
        with:
          version: latest

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test -race -coverprofile=coverage.out -covermode=atomic ./...
      - uses: codecov/codecov-action@v4
        with:
          file: coverage.out
          token: ${{ secrets.CODECOV_TOKEN }}

  build:
    runs-on: ubuntu-latest
    needs: [lint, test]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go build -o goshort ./cmd/server

  security:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: zricethezav/gitleaks-action@v2
```

**`.github/workflows/release.yml`** — Run on version tag:
```yaml
name: Release
on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
      packages: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**`.goreleaser.yml`** — Release configuration:
```yaml
version: 2
builds:
  - main: ./cmd/server
    binary: goshort
    env:
      - CGO_ENABLED=0
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ldflags:
      - -s -w -X main.version={{.Version}} -X main.commit={{.Commit}}

archives:
  - formats: [tar.gz]
    format_overrides:
      - goos: windows
        formats: [zip]

dockers:
  - image_templates:
      - "ghcr.io/yourname/goshort:{{ .Tag }}"
      - "ghcr.io/yourname/goshort:latest"
    dockerfile: Dockerfile

changelog:
  sort: asc
  filters:
    exclude:
      - '^docs:'
      - '^chore:'
      - '^ci:'
```

---

## Phase 0: Project Skeleton (~1 day)

> **Goal:** Empty but runnable project with all tooling configured.  
> **Deliverable:** `make ci` passes (lint + test + build on zero code).

### Milestone 0.1: Repository Setup

- [x] **T0.1** Create GitHub repository `goshort`
- [x] **T0.2** Clone locally, `cd goshort`
- [x] **T0.3** `go mod init github.com/yourname/goshort`
- [x] **T0.4** Create directory structure:
  ```
  mkdir -p cmd/server cmd/cli internal/{shortener,storage,cache,api,mcp,config} db docs api-tests data
  ```
- [x] **T0.5** Create placeholder `cmd/server/main.go`:
  ```go
  package main
  func main() { println("goshort") }
  ```
- [x] **T0.6** Verify: `go build ./cmd/server` compiles

### Milestone 0.2: Tooling Configuration

- [x] **T0.7** Create `.gitignore` (from design doc)
- [x] **T0.8** Create `.editorconfig`
- [x] **T0.9** Create `Makefile` with targets: build, test, lint, clean, ci
- [x] **T0.10** Install & configure golangci-lint:
  ```bash
  curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b ./bin
  ```
- [x] **T0.11** Create `.golangci.yml`
- [x] **T0.12** Install & configure lefthook:
  ```bash
  go install github.com/evilmartians/lefthook@latest
  lefthook install
  ```
- [x] **T0.13** Create `lefthook.yml`
- [x] **T0.14** Install gitleaks: `brew install gitleaks` or download binary
- [x] **T0.15** Create `.gitleaks.toml`
- [x] **T0.16** Verify: `make lint` passes, `make test` passes

### Milestone 0.3: GitHub Setup

- [x] **T0.17** Create `.github/workflows/ci.yml`
- [x] **T0.18** Create `LICENSE` (MIT)
- [x] **T0.19** Create initial `README.md` (project description, badges)
- [x] **T0.20** Copy project docs: `CLAUDE.md`, `docs/design.md`, `docs/LEARNING.md`, `docs/cc-workflow-guide.md`
- [x] **T0.21** Install samber skills (8 selected — see `docs/install-samber-skills.md`)
- [x] **T0.22** `git add . && git commit -m "chore: initial project skeleton with tooling"`
- [x] **T0.23** Push to GitHub, verify CI passes
- [x] **T0.24** Tag: `git tag v0.0.1 && git push --tags`

---

## Phase 1: Core Library + Storage (~2 weeks)

> **Goal:** Business logic works, fully tested via TDD. No HTTP yet.  
> **Deliverable:** `make test` passes with 90%+ coverage on core.  
> **Tag:** `v0.1.0`

### Milestone 1.1: Domain Types & Interfaces

- [x] **T1.1** Create `internal/shortener/model.go` — URL struct, CreateRequest, ListOptions
- [x] **T1.2** Create `internal/shortener/errors.go` — Sentinel errors: ErrNotFound, ErrExpired, ErrAliasTaken, ErrReservedPath, ErrInvalidURL
- [x] **T1.3** Create `internal/shortener/service.go` — Service interface definition:
  ```go
  type Service interface {
      Create(ctx, CreateRequest) (*URL, error)
      GetByCode(ctx, code) (*URL, error)
      Delete(ctx, code) error
      List(ctx, ListOptions) ([]URL, int, error)
      IncrementClicks(ctx, code) error
  }
  ```
- [x] **T1.4** Create `internal/storage/storage.go` — Storage interface
- [x] **T1.5** Create `internal/cache/cache.go` — Cache interface (Get, Set, Delete)
- [x] **T1.6** Create `internal/shortener/encoder.go` — Encoder interface (Encode, Decode)
- [x] **T1.7** Verify: `go build ./...` compiles

### Milestone 1.2: Sqids Encoder (TDD)

- [x] **T1.8** Write failing tests: `internal/shortener/encoder_test.go`
  - Encode returns non-empty string
  - Encode → Decode roundtrip returns same ID
  - Different IDs produce different codes
  - Codes contain only `[a-zA-Z0-9]`
  - Consecutive IDs produce non-sequential codes
- [x] **T1.9** Install sqids: `go get github.com/sqids/sqids-go`
- [x] **T1.10** Implement `SqidsEncoder` — make tests pass
- [x] **T1.11** Write fuzz test: `FuzzEncode` — never panics on any int64 input
- [x] **T1.12** `make test` — all pass

### Milestone 1.3: URL Validator (TDD)

- [x] **T1.13** Write failing tests: `internal/shortener/validator_test.go`
  - ValidateURL: valid HTTP/HTTPS, reject FTP/empty/no-scheme/too-long/private-IPs
  - ValidateAlias: valid chars, length 3-30, reject hyphens at start/end, reject reserved paths
  - ValidateExpiresIn: valid durations (1h, 7d, 30d), reject invalid formats, min/max bounds
- [x] **T1.14** Implement validators — make tests pass
- [x] **T1.15** Write fuzz test: `FuzzValidateURL` — never panics
- [x] **T1.16** `make test` — all pass

### Milestone 1.4: SQLite Storage (TDD)

- [x] **T1.17** Configure sqlc: create `db/sqlc.yaml`, `db/schema.sql`, `db/queries.sql`
- [x] **T1.18** Run `sqlc generate` — verify generated Go code
- [x] **T1.19** Install SQLite driver: `go get modernc.org/sqlite`
- [x] **T1.20** Implement `internal/storage/sqlite.go` — wraps sqlc-generated queries
- [x] **T1.21** Write integration tests: `internal/storage/sqlite_test.go`
  - Create URL → GetByCode returns it
  - Create with custom alias → GetByCode returns it
  - GetByCode for non-existent → ErrNotFound
  - Delete → GetByCode returns ErrNotFound
  - List with pagination
  - IncrementClicks
  - DeleteExpired removes only expired rows
  - Counter increments atomically
- [x] **T1.22** Compile-time interface check: `var _ storage.Storage = (*SQLiteStorage)(nil)`
- [x] **T1.23** `make test` — all pass

### Milestone 1.5: Cache Layer

- [x] **T1.24** Implement `internal/cache/noop.go` — no-op (Get always misses)
- [x] **T1.25** Implement `internal/cache/memory.go` — `sync.Map` with TTL
- [x] **T1.26** Write tests for memory cache: set/get, TTL expiry, delete
- [x] **T1.27** Compile-time interface checks for both

### Milestone 1.6: Service Implementation (TDD)

- [ ] **T1.28** Write failing tests: `internal/shortener/service_test.go`
  - Create URL: returns valid short code
  - Create with custom alias: returns alias as short code
  - Create with duplicate alias: returns ErrAliasTaken
  - Create with reserved path: returns ErrReservedPath
  - Create with expiration: sets expires_at correctly
  - GetByCode: returns URL
  - GetByCode expired: returns ErrExpired
  - Delete: removes URL
  - List: returns paginated results
- [ ] **T1.29** Implement `internal/shortener/service_impl.go` — make tests pass
- [ ] **T1.30** `make test` — all pass
- [ ] **T1.31** `make test-cover` — verify 90%+ on core packages
- [ ] **T1.32** `make lint` — no issues
- [ ] **T1.33** Commit: `feat: implement core library with TDD`
- [ ] **T1.34** Tag: `git tag v0.1.0`

---

## Phase 2: HTTP API + Scaling Reads (~3 weeks)

> **Goal:** Working API, all endpoints from design doc, caching layers, observability.  
> **Deliverable:** `curl localhost:8080/api/v1/urls` works.  
> **Tag:** `v0.2.0`

### Milestone 2.1: Chi Router + Handlers

- [ ] **T2.1** Install chi: `go get github.com/go-chi/chi/v5`
- [ ] **T2.2** Create `internal/api/router.go` — route definitions with groups
- [ ] **T2.3** Create `internal/api/errors.go` — error response helpers, error→status mapping
- [ ] **T2.4** Create `internal/api/handler.go` — Handler struct with service dependency
- [ ] **T2.5** Implement `POST /api/v1/urls` handler
- [ ] **T2.6** Implement `GET /{code}` redirect handler (302 + click counting)
- [ ] **T2.7** Implement `GET /api/v1/urls` list handler (pagination)
- [ ] **T2.8** Implement `GET /api/v1/urls/{code}` detail handler
- [ ] **T2.9** Implement `DELETE /api/v1/urls/{code}` handler
- [ ] **T2.10** Implement `GET /health` endpoint

### Milestone 2.2: HTTP Tests

- [ ] **T2.11** Write httptest tests for every endpoint + error case
  - POST: 201, 400 (bad URL), 409 (alias taken), 422 (invalid alias, reserved path)
  - GET redirect: 302, 404, 410 (expired)
  - GET list: 200 with pagination
  - DELETE: 204, 404
  - Health: 200
- [ ] **T2.12** `make test` — all pass

### Milestone 2.3: Server Wiring

- [ ] **T2.13** Update `cmd/server/main.go`:
  - Parse config (TOML + env vars)
  - Initialize storage, cache, encoder, service, handler
  - Setup router with middleware placeholder
  - Graceful shutdown with signal handling
  - Start background cleanup goroutine
- [ ] **T2.14** Verify: `make run` starts server, `curl` works

### Milestone 2.4: Logging + Metrics

- [ ] **T2.15** Create `internal/api/middleware.go` — logging middleware (slog)
- [ ] **T2.16** Add request duration, method, path, status to logs
- [ ] **T2.17** Install Prometheus: `go get github.com/prometheus/client_golang`
- [ ] **T2.18** Add metrics middleware: request count, duration histogram
- [ ] **T2.19** Implement `GET /metrics` endpoint
- [ ] **T2.20** Verify: `curl localhost:8080/metrics` shows Prometheus output

### Milestone 2.5: Caching Layers

- [ ] **T2.21** Integrate cache into redirect handler (cache-aside pattern)
- [ ] **T2.22** Implement cache TTL alignment (match URL expiry)
- [ ] **T2.23** Implement cache invalidation on DELETE
- [ ] **T2.24** Install Redis client: `go get github.com/redis/go-redis/v9`
- [ ] **T2.25** Implement `internal/cache/redis.go`
- [ ] **T2.26** Write tests for Redis cache (with real Redis via Docker)
- [ ] **T2.27** Config flag: `cache.driver = "none" | "memory" | "redis"`

### Milestone 2.6: Configuration

- [ ] **T2.28** Install viper: `go get github.com/spf13/viper`
- [ ] **T2.29** Implement `internal/config/config.go` — TOML + env vars
- [ ] **T2.30** Create default `goshort.toml`
- [ ] **T2.31** Verify all config options work

### Milestone 2.7: API Documentation

- [ ] **T2.32** Write `docs/openapi.yaml` (all 5 endpoints)
- [ ] **T2.33** Serve Scalar UI at `/docs` endpoint
- [ ] **T2.34** Verify: browser at `localhost:8080/docs` shows interactive API docs

### Milestone 2.8: Bruno Test Collection

- [ ] **T2.35** Create Bruno collection in `api-tests/`
- [ ] **T2.36** Create `.bru` files for each endpoint
- [ ] **T2.37** Create environments: local + production
- [ ] **T2.38** Verify all Bruno tests pass against running server

### Milestone 2.9: CI + Coverage

- [ ] **T2.39** Setup Codecov: connect GitHub repo, add token to secrets
- [ ] **T2.40** Verify CI workflow: lint → test (with coverage upload) → build
- [ ] **T2.41** Add Codecov badge to README
- [ ] **T2.42** `make lint && make test` — all pass
- [ ] **T2.43** Commit: `feat: HTTP API with caching, monitoring, and docs`
- [ ] **T2.44** Tag: `git tag v0.2.0`

---

## Phase 3: Auth + Rate Limiting + CLI (~2 weeks)

> **Goal:** Secure, deployable, usable product. v1.0 opensource release.  
> **Deliverable:** CLI works, Docker runs, GitHub release with binaries.  
> **Tag:** `v1.0.0`

### Milestone 3.1: Authentication

- [ ] **T3.1** Implement API key auth middleware
- [ ] **T3.2** Config: `auth.api_key` from TOML/env
- [ ] **T3.3** Skip auth for public routes: `GET /{code}`, `/health`, `/docs`
- [ ] **T3.4** Write tests: 401 without key, 403 wrong key, 200 correct key
- [ ] **T3.5** Update Bruno tests with API key in environment

### Milestone 3.2: Rate Limiting

- [ ] **T3.6** Implement token bucket rate limiter middleware
- [ ] **T3.7** Per-IP rate limiting with `golang.org/x/time/rate`
- [ ] **T3.8** Config: `rate_limit.enabled`, `rate_limit.requests_per_minute`
- [ ] **T3.9** Return 429 with `Retry-After` header
- [ ] **T3.10** Write tests: under limit passes, over limit returns 429

### Milestone 3.3: CLI Client

- [ ] **T3.11** Install cobra: `go get github.com/spf13/cobra`
- [ ] **T3.12** Implement `cmd/cli/main.go` with root command
- [ ] **T3.13** `goshort-cli shorten <url> [--alias] [--expires]`
- [ ] **T3.14** `goshort-cli list [--page] [--per-page]`
- [ ] **T3.15** `goshort-cli delete <code>`
- [ ] **T3.16** `goshort-cli stats <code>`
- [ ] **T3.17** Config file: `~/.goshort.toml` (server URL, API key)
- [ ] **T3.18** Pipe support: `echo "https://example.com" | goshort-cli shorten`
- [ ] **T3.19** Write CLI tests

### Milestone 3.4: Docker

- [ ] **T3.20** Create `Dockerfile` (multi-stage build)
- [ ] **T3.21** Create `docker-compose.yml` (app + Caddy + optional Redis)
- [ ] **T3.22** Create `Caddyfile`
- [ ] **T3.23** Verify: `docker compose up -d` → app accessible via HTTPS locally
- [ ] **T3.24** Test: `docker build -t goshort . && docker run -p 8080:8080 goshort`

### Milestone 3.5: Release Infrastructure

- [ ] **T3.25** Create `.goreleaser.yml`
- [ ] **T3.26** Create `.github/workflows/release.yml`
- [ ] **T3.27** Verify: `goreleaser release --snapshot --clean` builds all platforms
- [ ] **T3.28** Create `CHANGELOG.md` (initial entry)
- [ ] **T3.29** Create `SECURITY.md`
- [ ] **T3.30** Create `.github/CODEOWNERS`
- [ ] **T3.31** Create `.github/ISSUE_TEMPLATE/bug_report.yml`
- [ ] **T3.32** Create `.github/ISSUE_TEMPLATE/feature_request.yml`

### Milestone 3.6: README + Opensource

- [ ] **T3.33** Write comprehensive `README.md`:
  - Project description + badges (CI, coverage, release)
  - Quick start (3 commands)
  - Installation: binary download, Docker, go install
  - Configuration reference
  - API documentation link
  - Contributing guide
- [ ] **T3.34** Final review: `make ci` passes, all tests green
- [ ] **T3.35** Commit: `feat: v1.0.0 — auth, CLI, Docker, opensource ready`
- [ ] **T3.36** Tag + push: `git tag v1.0.0 && git push --tags`
- [ ] **T3.37** Verify: GitHub release created with binaries + Docker image

---

## Phase 3.5: Deploy + Infrastructure (~1 week)

> **Goal:** App live on internet. Learn infrastructure by doing.  
> **Deliverable:** `https://goshort.yourdomain.com` works.

### Milestone 3.5.1: Fly.io Deploy (Ship Fast)

- [ ] **T3.38** Install flyctl: `brew install flyctl`
- [ ] **T3.39** `fly launch` — configure app
- [ ] **T3.40** `fly volumes create data --size 1` — persistent storage
- [ ] **T3.41** `fly deploy` — verify app is live
- [ ] **T3.42** Configure custom domain in Fly.io dashboard
- [ ] **T3.43** Test: short URL creation + redirect works on live URL

### Milestone 3.5.2: Cloudflare DNS + CDN

- [ ] **T3.44** Point domain DNS to Fly.io (or VPS) via Cloudflare
- [ ] **T3.45** Enable Cloudflare proxy (orange cloud)
- [ ] **T3.46** Configure Cache Rules for redirect responses
- [ ] **T3.47** Verify: CDN caching works (check `cf-cache-status` header)

### Milestone 3.5.3: Bare VPS Deploy (Learn)

- [ ] **T3.48** Buy VPS (Hetzner CX22 ~$4/mo)
- [ ] **T3.49** SSH in, create non-root user, setup SSH keys
- [ ] **T3.50** Configure firewall: `ufw allow 22,80,443`
- [ ] **T3.51** Copy GoShort binary to server
- [ ] **T3.52** Create systemd service file, enable + start
- [ ] **T3.53** Install Nginx, configure reverse proxy
- [ ] **T3.54** Install Certbot, setup HTTPS (Let's Encrypt)
- [ ] **T3.55** Verify: `https://goshort.yourdomain.com` works via VPS
- [ ] **T3.56** Compare: try Caddy as alternative to Nginx
- [ ] **T3.57** Try Docker Compose deploy on VPS

---

## Phase 4: MCP Server (~1 week)

> **Goal:** AI agents can shorten URLs via MCP tools.  
> **Deliverable:** Claude Code calls `shorten_url` successfully.  
> **Tag:** `v1.1.0`

### Milestone 4.1: MCP Implementation

- [ ] **T4.1** Install mcp-go: `go get github.com/mark3labs/mcp-go`
- [ ] **T4.2** Create `internal/mcp/server.go` — MCP server setup (stdio transport)
- [ ] **T4.3** Create `internal/mcp/tools.go` — tool definitions:
  - `shorten_url(url, alias?, expires_in?)`
  - `list_urls(page?, per_page?)`
  - `get_stats(code)`
  - `delete_url(code)`
- [ ] **T4.4** Add `mcp` subcommand to `cmd/server/main.go`
- [ ] **T4.5** Create `.mcp.json` config for Claude Code

### Milestone 4.2: Test + Verify

- [ ] **T4.6** Write tests for MCP tool handlers
- [ ] **T4.7** Manual test: connect Claude Code, run `shorten_url`
- [ ] **T4.8** Update README with MCP setup instructions
- [ ] **T4.9** Commit: `feat: MCP server for AI agent integration`
- [ ] **T4.10** Tag: `git tag v1.1.0`

---

## Phase 5+: Future

> Prioritized backlog. Tackle after Phase 4.

### Analytics & Dashboard
- [ ] Click analytics (referrer, user-agent, timestamp)
- [ ] Basic stats API endpoint
- [ ] Simple HTML dashboard (or Grafana)

### Security
- [ ] Spam/malicious URL detection (blocklist)
- [ ] URL content scanning
- [ ] API key rotation support

### AI Agent
- [ ] Smart slug generation (fetch page title → suggest alias)
- [ ] Bulk URL categorization
- [ ] Natural language CLI interface

### Scale (Learning)
- [ ] PostgreSQL storage backend
- [ ] Redis distributed counter + counter batching
- [ ] DB replication (primary-replica) via Docker Compose
- [ ] singleflight for cache stampede prevention

---

## Appendix: Task Workflow

### For each task, follow this flow:

```
1. DISCUSS — Do I understand the task? Check design.md if unclear.
2. PLAN    — What files to create/modify? What tests to write first?
3. EXECUTE — Write failing test (Red) → implement (Green) → refactor.
4. TEST    — make test (all pass?) → make lint (no issues?)
5. VERIFY  — Manual check: does it actually work? (curl, run app)
6. COMMIT  — Conventional commit: feat:/fix:/test:/refactor:
```

### CC Prompt Template

```
Implement task T1.8 from ROADMAP.md.
Read @docs/design.md Section 7 for context.

Write failing tests first (TDD), then implement.
Run `make test` to verify. Run `make lint` to check style.
```

### Session Boundaries

```
Good session: 1 milestone (3-6 tasks), clear goal, clear deliverable.
Bad session:  "Build everything in Phase 1" (too broad, context overflows).

Start fresh sessions at milestone boundaries.
```

---

*Last updated: 2025-06-01*
*Companion to: [design.md](./design.md) · [LEARNING.md](./LEARNING.md) · [cc-workflow-guide.md](./cc-workflow-guide.md)*
