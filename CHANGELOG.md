# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-05-03

### Added
- API key authentication for write endpoints (`X-API-Key` header)
- Per-IP token bucket rate limiting (`golang.org/x/time/rate`)
- CLI client (`goshort-cli`) with shorten, list, delete, stats commands
- Dockerfile (multi-stage, ~49MB image) and Docker Compose with Caddy reverse proxy
- Automated releases via GoReleaser (cross-platform binaries + Docker image)
- Community files: CONTRIBUTING.md, SECURITY.md, issue templates, PR template

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
- HTTP API with Chi router (POST /api/v1/urls, GET, DELETE, LIST endpoints)
- URL redirect with 302 + click tracking
- Cache-aside pattern with memory and Redis backends
- Configuration via TOML file + environment variables (Koanf)
- Prometheus metrics and structured logging (slog)
- Interactive API docs at `/docs` (Scalar UI + OpenAPI 3.1 spec)
- Background cleanup goroutine for expired URLs
- Bruno API test collection for all endpoints

## [0.1.0] - 2026-05-02

### Added
- Core URL shortening library with zero-collision Sqids encoding
- SQLite storage via sqlc (pure Go, no CGO)
- URL validation (scheme, length, private IP blocking)
- Custom alias support with charset differentiation
- URL expiration with configurable TTL
- In-memory cache with lazy TTL expiry
- Comprehensive test suite (unit, integration, fuzz)
