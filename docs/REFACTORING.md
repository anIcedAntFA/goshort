# GoShort v0.2.0 → v0.2.1 Refactor Prompt for Claude Code

> **Context:** This prompt addresses issues found during a full code review of GoShort v0.2.0.
> Run each section as a separate CC session to keep context focused.
> After ALL sections are done, run `make lint && make test` to verify.

---

## Session 1: P0 — Fix env var config bug (`_` → `.` breaks multi-word field names)

### Problem

`internal/config/config.go` `envKeyTransform` converts ALL underscores to dots:

```
GOSHORT_CACHE_REDIS_URL → cache.redis.url   ← WRONG
                          cache.redis_url    ← CORRECT (matches koanf tag)
```

Affected fields: `redis_url`, `sqlite_path`, `code_length`, `default_expiry`, `requests_per_minute`, `api_key`, `base_url` — ALL config keys with underscores in the field name (not section separators) are broken when set via env vars.

### Root cause

```go
func envKeyTransform(k, v string) (string, any) {
    return strings.ReplaceAll(
        strings.ToLower(strings.TrimPrefix(k, "GOSHORT_")),
        "_", ".",
    ), v
}
```

The function can't distinguish between `_` as a section separator (`SERVER_PORT` → `server.port`) and `_` as part of a field name (`REDIS_URL` → `redis_url`).

### Fix approach

Use koanf's env provider with a **manual key mapping** or **double-underscore convention**:

**Option: Double-underscore as section separator.**
- `GOSHORT__SERVER__PORT` → `server.port`
- `GOSHORT__CACHE__REDIS_URL` → `cache.redis_url`
- Single underscore within a section stays as underscore in the key.

**BUT** this is a breaking change for any existing env var users. Since we're pre-v1.0, this is acceptable. However, the simpler approach is to use a **known key map** since GoShort has a small, fixed config schema:

```go
// Build the flat key list from defaults map
// "server.port", "cache.redis_url", etc.
// Match env var GOSHORT_SERVER_PORT against these keys
// by normalizing: lowercase the env var suffix, then try all possible
// dot-insertion points to find a match.
```

The cleanest fix for a small config: **use koanf's callback-based env provider** with explicit mapping.

### Implementation

1. In `internal/config/config.go`, replace the `envKeyTransform` function. The new transform must:
   - Strip `GOSHORT_` prefix
   - Lowercase the remainder
   - Replace `_` with `.` ONLY at section boundaries (not within field names)
   - Use the `defaults` map keys as the source of truth for valid key paths

   One clean approach: iterate `defaults` keys, generate the expected env var name from each key (reverse: `server.port` → `server_port`), and build a lookup map `envVarSuffix → koanfKey`. Then the transform just does a map lookup.

   ```go
   // buildEnvKeyMap generates a map from lowercased env var suffixes to koanf keys.
   // Example: "server_port" → "server.port", "cache_redis_url" → "cache.redis_url"
   func buildEnvKeyMap(defaults map[string]any) map[string]string {
       m := make(map[string]string, len(defaults))
       for key := range defaults {
           envSuffix := strings.ReplaceAll(key, ".", "_")
           m[envSuffix] = key
       }
       return m
   }
   ```

   Then the transform becomes:
   ```go
   envMap := buildEnvKeyMap(defaults)

   func envKeyTransform(k, v string) (string, any) {
       suffix := strings.ToLower(strings.TrimPrefix(k, "GOSHORT_"))
       if mapped, ok := envMap[suffix]; ok {
           return mapped, v
       }
       // Fallback: unknown env var, skip it
       return "", nil
   }
   ```

   **Note:** koanf's env provider with `TransformFunc` — if the function returns empty string `""` as key, check koanf docs for how it handles that (it may skip the entry or store under empty key). If koanf doesn't skip empty keys, filter them out differently. Read the koanf env provider source to verify.

2. Update `internal/config/config_test.go` — add tests that verify multi-word field names work via env vars:
   ```go
   func TestLoad_EnvOverride_MultiWordField(t *testing.T) {
       t.Setenv("GOSHORT_CACHE_REDIS_URL", "redis://custom:6380")
       t.Setenv("GOSHORT_STORAGE_SQLITE_PATH", "/tmp/custom.db")
       t.Setenv("GOSHORT_SHORTENER_CODE_LENGTH", "8")
       t.Setenv("GOSHORT_SHORTENER_DEFAULT_EXPIRY", "30d")
       t.Setenv("GOSHORT_RATE_LIMIT_REQUESTS_PER_MINUTE", "120")
       t.Setenv("GOSHORT_AUTH_API_KEY", "sk_test_123")
       t.Setenv("GOSHORT_SERVER_BASE_URL", "https://short.example.com")

       cfg, err := config.Load("")
       if err != nil {
           t.Fatalf("Load error: %v", err)
       }

       if cfg.Cache.RedisURL != "redis://custom:6380" {
           t.Errorf("cache.redis_url = %q, want redis://custom:6380", cfg.Cache.RedisURL)
       }
       if cfg.Storage.SQLitePath != "/tmp/custom.db" {
           t.Errorf("storage.sqlite_path = %q, want /tmp/custom.db", cfg.Storage.SQLitePath)
       }
       if cfg.Shortener.CodeLength != 8 {
           t.Errorf("shortener.code_length = %d, want 8", cfg.Shortener.CodeLength)
       }
       if cfg.Shortener.DefaultExpiry != "30d" {
           t.Errorf("shortener.default_expiry = %q, want 30d", cfg.Shortener.DefaultExpiry)
       }
       if cfg.RateLimit.RequestsPerMinute != 120 {
           t.Errorf("rate_limit.requests_per_minute = %d, want 120", cfg.RateLimit.RequestsPerMinute)
       }
       if cfg.Auth.APIKey != "sk_test_123" {
           t.Errorf("auth.api_key = %q, want sk_test_123", cfg.Auth.APIKey)
       }
       if cfg.Server.BaseURL != "https://short.example.com" {
           t.Errorf("server.base_url = %q, want https://short.example.com", cfg.Server.BaseURL)
       }
   }
   ```

3. Verify existing tests still pass — `TestLoad_EnvOverride` (`GOSHORT_SERVER_PORT`) and `TestLoad_EnvPrefix` (`GOSHORT_CACHE_DRIVER`) must not break.

4. Run `make test` and `make lint`.

---

## Session 2: P1 — Fix cache key prefix inconsistency in Delete

### Problem

The handler uses `"short:" + code` as cache key everywhere. But the service's `Delete` method also calls `cache.Delete(ctx, code)` — **without the prefix**. This is dead code that deletes a non-existent key.

### Files to change

**`internal/shortener/service_impl.go`** — Remove the cache delete from the service. Cache is a delivery-layer concern managed by the handler:

```go
// BEFORE:
func (s *ServiceImpl) Delete(ctx context.Context, code string) error {
    if err := s.store.DeleteByCode(ctx, code); err != nil {
        return fmt.Errorf("delete: %w", err)
    }
    _ = s.cache.Delete(ctx, code)  // ← REMOVE THIS LINE
    return nil
}

// AFTER:
func (s *ServiceImpl) Delete(ctx context.Context, code string) error {
    if err := s.store.DeleteByCode(ctx, code); err != nil {
        return fmt.Errorf("delete: %w", err)
    }
    return nil
}
```

**Verify the handler still does the cache delete with correct prefix** (`internal/api/handler.go`):
```go
func (h *Handler) DeleteURL(w http.ResponseWriter, r *http.Request) {
    // ...
    _ = h.cache.Delete(r.Context(), "short:"+code)  // ← this stays, correct prefix
    w.WriteHeader(http.StatusNoContent)
}
```

### Side effects to check

- `internal/shortener/service_test.go` — `TestService_Delete` should still pass (it uses `mockStorage` which doesn't check cache key prefixes). No changes needed in tests.
- The `ServiceImpl` struct still holds a `cache` field — it's used in `Delete`. After removing, check if `cache` is still used elsewhere in the service. If not, consider whether it should remain. Currently the service doesn't use cache for anything else (cache-aside is in the handler). **If `cache` is only used for the delete call being removed, remove the `cache` field from `ServiceImpl` and the `NewService` constructor parameter.**

Wait — check if `cache` is used elsewhere in `ServiceImpl`:
- `Create` — no
- `GetByCode` — no (cache-aside is in handler)
- `Delete` — yes (the line being removed)
- `List` — no
- `IncrementClicks` — no

**If removing the cache field from ServiceImpl:**
1. Update `NewService(store Storage, c Cache, enc Encoder) *ServiceImpl` → `NewService(store Storage, enc Encoder) *ServiceImpl`
2. Update ALL callers of `NewService`:
   - `cmd/server/main.go`
   - `internal/api/handler_test.go` (in `setupTestDeps`)
   - `internal/shortener/service_test.go` (in `newTestService`)
3. The `Cache` interface remains in `shortener/cache.go` because the handler still uses it.

**This is a judgment call.** Removing cache from the service is architecturally cleaner (cache-aside belongs in the delivery layer), but it means the service can never do internal caching in the future without re-adding it. My recommendation: **remove it now, re-add if needed later.** YAGNI.

Run `make test && make lint` after changes.

---

## Session 3: P1 — Fix Prometheus label cardinality for unmatched routes

### Problem

In `internal/api/middleware.go`, the MetricsMiddleware falls back to raw URL path when no route pattern exists:

```go
pattern := chi.RouteContext(r.Context()).RoutePattern()
if pattern == "" {
    pattern = r.URL.Path  // ← unbounded cardinality!
}
```

Bot probes like `/wp-login.php`, `/admin`, etc. each create a unique label value → Prometheus OOM.

### Fix

```go
if pattern == "" {
    pattern = "unmatched"
}
```

### Tests

No test changes needed — the metrics middleware doesn't have direct unit tests (it's tested implicitly through handler tests). But verify `make test` still passes.

Run `make lint` to verify.

---

## Session 4: P2 — Fix IncrementClicks error swallowing

### Problem

`internal/api/handler.go` Redirect handler:
```go
go func() { _ = h.svc.IncrementClicks(clickCtx, code) }()
```

Error is silently discarded. If DB is down, click data is lost with zero visibility.

### Fix

In `internal/api/handler.go`, update BOTH places where `IncrementClicks` is called (cache-hit path and cache-miss path):

```go
// BEFORE:
go func() { _ = h.svc.IncrementClicks(clickCtx, code) }()

// AFTER:
go func() {
    if err := h.svc.IncrementClicks(clickCtx, code); err != nil {
        h.logger.Warn("increment clicks failed", "code", code, "error", err)
    }
}()
```

There are TWO `go func()` blocks in the `Redirect` handler — one for cache hit, one for cache miss. Update BOTH.

### Tests

No test changes needed — the fire-and-forget pattern isn't directly asserted in tests. But verify `make test` passes.

---

## Session 5: P2 — Add config validation

### Problem

Only `shortener.code_length` is validated in `main.go`. Other fields have no validation.

### Implementation

Add a `Validate() error` method to `Config` in `internal/config/config.go`:

```go
func (c *Config) Validate() error {
    if c.Server.Port < 0 || c.Server.Port > 65535 {
        return fmt.Errorf("server.port %d out of range [0, 65535]", c.Server.Port)
    }
    if c.Server.BaseURL == "" {
        return fmt.Errorf("server.base_url must not be empty")
    }
    if c.Shortener.CodeLength < 1 || c.Shortener.CodeLength > 255 {
        return fmt.Errorf("shortener.code_length %d out of range [1, 255]", c.Shortener.CodeLength)
    }
    switch c.Cache.Driver {
    case "none", "memory", "redis":
        // valid
    default:
        return fmt.Errorf("cache.driver %q must be one of: none, memory, redis", c.Cache.Driver)
    }
    switch c.Storage.Driver {
    case "sqlite":
        // valid
    default:
        return fmt.Errorf("storage.driver %q must be one of: sqlite", c.Storage.Driver)
    }
    switch c.Logging.Level {
    case "debug", "info", "warn", "error":
        // valid
    default:
        return fmt.Errorf("logging.level %q must be one of: debug, info, warn, error", c.Logging.Level)
    }
    switch c.Logging.Format {
    case "json", "text":
        // valid
    default:
        return fmt.Errorf("logging.format %q must be one of: json, text", c.Logging.Format)
    }
    return nil
}
```

Update `cmd/server/main.go`:
```go
cfg, err := config.Load(*configPath)
if err != nil {
    fmt.Fprintf(os.Stderr, "load config: %v\n", err)
    os.Exit(1)
}

if err := cfg.Validate(); err != nil {
    fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
    os.Exit(1)
}

// REMOVE the inline code_length validation that's currently in main.go:
// if cfg.Shortener.CodeLength < 1 || cfg.Shortener.CodeLength > 255 { ... }
```

Add tests in `internal/config/config_test.go`:

```go
func TestConfig_Validate(t *testing.T) {
    t.Parallel()

    validConfig := func() *config.Config {
        cfg, _ := config.Load("")
        return cfg
    }

    t.Run("defaults are valid", func(t *testing.T) {
        t.Parallel()
        cfg := validConfig()
        if err := cfg.Validate(); err != nil {
            t.Errorf("default config invalid: %v", err)
        }
    })

    t.Run("invalid cache driver", func(t *testing.T) {
        t.Parallel()
        cfg := validConfig()
        cfg.Cache.Driver = "memcached"
        if err := cfg.Validate(); err == nil {
            t.Error("expected error for invalid cache driver")
        }
    })

    t.Run("empty base_url", func(t *testing.T) {
        t.Parallel()
        cfg := validConfig()
        cfg.Server.BaseURL = ""
        if err := cfg.Validate(); err == nil {
            t.Error("expected error for empty base_url")
        }
    })

    t.Run("code_length zero", func(t *testing.T) {
        t.Parallel()
        cfg := validConfig()
        cfg.Shortener.CodeLength = 0
        if err := cfg.Validate(); err == nil {
            t.Error("expected error for code_length 0")
        }
    })

    t.Run("negative port", func(t *testing.T) {
        t.Parallel()
        cfg := validConfig()
        cfg.Server.Port = -1
        if err := cfg.Validate(); err == nil {
            t.Error("expected error for negative port")
        }
    })
}
```

Run `make test && make lint`.

---

## Session 6: P3 — Optimize slow pagination test

### Problem

`internal/storage/sqlite_test.go` `TestSQLiteStorage_ListURLs_Pagination` sleeps 1.1s between each of 5 inserts (total 5.5s) because `datetime('now')` has 1-second resolution.

### Fix approach

Add a `created_at` parameter to `CreateURL` sqlc query so tests can supply explicit timestamps. This requires:

1. Update `db/queries.sql` — add a new query or modify `CreateURL` to accept `created_at`:

   ```sql
   -- name: CreateURL :one
   INSERT INTO urls (short_code, original_url, is_custom, expires_at, created_at)
   VALUES (?, ?, ?, ?, COALESCE(?, datetime('now')))
   RETURNING *;
   ```

   This makes `created_at` optional — pass NULL to use the default `datetime('now')`, or pass a specific value for tests.

2. Run `sqlc generate` to regenerate `internal/db/`.

3. Update `internal/storage/sqlite.go` `CreateURL` to pass the new parameter. For production use, pass `sql.NullString{}` (NULL → uses default). For the `CreateParams` struct in `shortener/storage.go`, add an optional `CreatedAt *time.Time` field.

4. Update the test to pass explicit timestamps instead of sleeping:

   ```go
   for i, code := range codes {
       params := sampleParams(code, "https://example.com/"+code)
       // Offset each by 1 second so ordering is deterministic
       ts := time.Now().Add(time.Duration(i) * time.Second)
       params.CreatedAt = &ts
       s.CreateURL(ctx, params)
   }
   // No more time.Sleep!
   ```

5. Update `internal/shortener/service_impl.go` — in `Create`, don't set `CreatedAt` in `CreateParams` (let DB default handle it). Only tests use explicit timestamps.

6. Update `mockStorage` in `service_test.go` if it reads `CreateParams.CreatedAt`.

7. Run `sqlc generate`, then `make test && make lint`.

**Alternative (simpler but less clean):** Change `ListURLs` ORDER BY from `created_at DESC` to `id DESC`. Since `AUTOINCREMENT` guarantees sequential IDs, this gives the same ordering without needing distinct timestamps. This requires:
- Update `db/queries.sql`: `ORDER BY id DESC` instead of `ORDER BY created_at DESC`
- Run `sqlc generate`
- Remove `time.Sleep` from the test
- No schema changes needed

**Recommendation:** Use the `ORDER BY id DESC` approach — it's simpler, equally correct for single-instance SQLite, and doesn't touch the schema or CreateParams. The `created_at` approach is better if you want to support "sort by date" as a user-facing feature later.

---

## Session 7: Architectural — Remove Decode from Encoder interface

### Rationale

`Decode` is defined on the `Encoder` interface but never called through the interface by any consumer. `GetByCode` looks up by short code string directly in the DB — no decoding needed. `Decode` only exists for roundtrip testing of the encoder itself.

Per Go proverb #4: "The bigger the interface, the weaker the abstraction." If no consumer calls `Decode`, it shouldn't be in the interface.

### Changes

1. **`internal/shortener/encoder.go`** — Remove `Decode` from interface:

   ```go
   // BEFORE:
   type Encoder interface {
       Encode(id int64) (string, error)
       Decode(code string) (int64, error)
   }

   // AFTER:
   type Encoder interface {
       Encode(id int64) (string, error)
   }
   ```

2. **`internal/encoder/sqids.go`** — Keep `Decode` method on `SqidsEncoder` (it's still useful for testing), but it no longer satisfies an interface method. No code change needed in this file.

3. **`internal/encoder/sqids_test.go`** — Tests for `Decode` remain valid. They test the concrete `SqidsEncoder` type, not the interface. No changes needed.

4. **Verify** no code calls `Decode` through the `Encoder` interface. Search the codebase:
   ```bash
   grep -rn "\.Decode(" internal/ --include="*.go" | grep -v "_test.go" | grep -v "encoder/"
   ```
   Should return nothing. If any file calls `.Decode()` on an `Encoder` interface variable, that call needs updating.

5. Run `make test && make lint`.

---

## Session 8: Architectural — Remove cache from ServiceImpl

This follows from Session 2. If the cache delete was the only use of cache in the service, clean up completely.

### Changes

1. **`internal/shortener/service_impl.go`**:
   ```go
   // BEFORE:
   type ServiceImpl struct {
       store   Storage
       cache   Cache
       encoder Encoder
   }

   func NewService(store Storage, c Cache, enc Encoder) *ServiceImpl {
       return &ServiceImpl{store: store, cache: c, encoder: enc}
   }

   // AFTER:
   type ServiceImpl struct {
       store   Storage
       encoder Encoder
   }

   func NewService(store Storage, enc Encoder) *ServiceImpl {
       return &ServiceImpl{store: store, encoder: enc}
   }
   ```

   Remove any remaining `s.cache` references in the file.

2. **`cmd/server/main.go`** — Update the `NewService` call:
   ```go
   // BEFORE:
   svc := shortener.NewService(store, c, enc)

   // AFTER:
   svc := shortener.NewService(store, enc)
   ```

   The `c` (cache) variable is still needed — it's passed to `api.NewHandler`.

3. **`internal/api/handler_test.go`** — Update `setupTestDeps`:
   ```go
   // BEFORE:
   svc := shortener.NewService(store, cache.NewNoopCache(), enc)

   // AFTER:
   svc := shortener.NewService(store, enc)
   ```

4. **`internal/shortener/service_test.go`** — Update `newTestService`:
   ```go
   // BEFORE:
   return shortener.NewService(store, cache.NewNoopCache(), enc), store

   // AFTER:
   return shortener.NewService(store, enc), store
   ```

5. **`internal/shortener/cache.go`** — Keep the `Cache` interface! The handler still uses it. Only the service stops depending on it.

6. Run `make test && make lint`.

---

## Session 9: Documentation — Interface evolution strategy in DESIGN.md

### Add to `docs/DESIGN.md`

Find section **8. Architecture & Project Structure** and add a new subsection after the existing content:

```markdown
### Interface Evolution Strategy

The `Service` interface currently has 5 methods (Create, GetByCode, Delete, List, IncrementClicks).
This is appropriate for Phase 1–3 where a single consumer (HTTP handler) uses all methods.

**Split point: Phase 4 (MCP server).** When the MCP server arrives as a second consumer
with different needs, split `Service` into composable interfaces:

​```go
type URLReader interface {
    GetByCode(ctx context.Context, code string) (*URL, error)
    List(ctx context.Context, opts ListOptions) ([]URL, int, error)
}

type URLWriter interface {
    Create(ctx context.Context, req CreateRequest) (*URL, error)
    Delete(ctx context.Context, code string) error
}

type ClickTracker interface {
    IncrementClicks(ctx context.Context, code string) error
}

// Full service composes all three.
type Service interface {
    URLReader
    URLWriter
    ClickTracker
}
​```

Because Go interfaces are satisfied implicitly, `ServiceImpl` will satisfy both the old
`Service` and the new split interfaces with zero code changes. The split only affects
consumers: each declares the minimal interface it needs.

**Cache placement:** Cache-aside logic lives in the delivery layer (HTTP handler, MCP handler),
not in the service. The service operates on storage directly. This keeps the service testable
without cache concerns and lets each delivery layer manage its own caching strategy.

**`Encoder` interface:** Contains only `Encode`. The `Decode` method exists on the concrete
`SqidsEncoder` for testing and potential future use, but is not part of the interface because
no consumer calls it.
```

Run `make lint` (docs don't affect lint, but good habit).

---

## Session 10: Final verification

Run the full CI checks:

```bash
make lint
make test
make test/race
make build
```

All must pass. If any tests fail, fix them before committing.

### Commit strategy

Split into atomic commits following the git-workflow skill:

1. `🐛 fix(config): resolve env var mapping for multi-word field names`
2. `🐛 fix(api): prevent Prometheus label cardinality explosion on unmatched routes`
3. `♻️ refactor(shortener): remove cache dependency from service layer`
4. `🐛 fix(api): log IncrementClicks errors instead of swallowing`
5. `✨ feat(config): add Validate method for config validation`
6. `♻️ refactor(shortener): remove Decode from Encoder interface`
7. `⚡️ perf(storage): optimize pagination test with ORDER BY id DESC`
8. `📝 docs(design): add interface evolution strategy`

Each commit should leave tests green. Don't batch unrelated changes.3
