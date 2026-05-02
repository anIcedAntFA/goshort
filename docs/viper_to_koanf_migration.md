# 🔄 Viper → Koanf Migration — All File Changes

> Apply these changes to replace Viper with Koanf across all project documentation.
> Use this as a CC prompt or apply manually.

---

## Summary of Changes

| File | What changes |
|------|-------------|
| `CLAUDE.md` | Tech stack table: Viper → Koanf |
| `docs/DESIGN.md` | Decisions Log D6 rationale, Section 8 libraries table, Section 16 config example |
| `docs/ROADMAP.md` | Milestone 2.6 tasks: install koanf, config.go using koanf API |
| `docs/LEARNING.md` | Part 2.6 Functional Options (Viper mention), Part 6 Knowledge Map |
| `README.md` | Tech stack mention, configuration section |

---

## 1. CLAUDE.md

### Replace in Technology Stack section:

OLD:
```
| CLI | `spf13/cobra` + `spf13/viper` |
```

NEW:
```
| CLI | `spf13/cobra` |
| Config | `knadh/koanf/v2` + TOML parser + env provider |
```

### Replace in Architecture section:

OLD:
```
└── config/            # Viper config loading (goshort.toml)
```

NEW:
```
└── config/            # Koanf config loading (goshort.toml + env vars)
```

### Replace in Code Conventions section:

OLD:
```
- All SQL lives in `db/queries.sql`; never write raw SQL strings in Go code
```

NEW:
```
- All SQL lives in `db/queries.sql`; never write raw SQL strings in Go code
- Config struct tags use `koanf:"field_name"` (not `mapstructure`)
```

---

## 2. docs/DESIGN.md

### Decisions Log — Replace D6 row:

Note: original design doc doesn't have a Viper decision row explicitly,
but the tech stack table in Section 8 references Viper. Add new decision:

ADD to Decisions Log table (after D22):
```
| D23 | Config library | Viper, Koanf, stdlib | **Koanf v2** | Modular deps (313% smaller binary than Viper); case-sensitive keys; no dependency bloat |
```

### Section 8 — Key Go Libraries table:

OLD:
```
| `github.com/spf13/viper` | Configuration | 3 |
```

NEW:
```
| `github.com/knadh/koanf/v2` | Configuration (core) | 2 |
| `github.com/knadh/koanf/providers/file` | Config from TOML file | 2 |
| `github.com/knadh/koanf/providers/env/v2` | Config from env vars | 2 |
| `github.com/knadh/koanf/parsers/toml/v2` | TOML parser | 2 |
```

### Section 16 — Configuration:

OLD (if Viper-specific code exists):
```toml
# goshort.toml
```

No change needed — the TOML file format stays the same. Koanf reads the same
goshort.toml. Only the Go code that loads it changes.

---

## 3. docs/ROADMAP.md

### Milestone 2.6 — Replace entirely:

OLD:
```
### Milestone 2.6: Configuration

- [ ] **T2.28** Install viper: `go get github.com/spf13/viper`
- [ ] **T2.29** Implement `internal/config/config.go` — TOML + env vars
- [ ] **T2.30** Create default `goshort.toml`
- [ ] **T2.31** Verify all config options work
```

NEW:
```
### Milestone 2.6: Configuration (Koanf)

- [ ] **T2.28** Install koanf:
  ```bash
  go get -u github.com/knadh/koanf/v2
  go get -u github.com/knadh/koanf/providers/file
  go get -u github.com/knadh/koanf/providers/env/v2
  go get -u github.com/knadh/koanf/parsers/toml/v2
  ```
- [ ] **T2.29** Implement `internal/config/config.go`:
  - Config struct with `koanf:"field"` tags
  - Load() function: TOML file → env vars (GOSHORT_ prefix) → defaults
  - Env var transform: GOSHORT_SERVER_PORT → server.port
- [ ] **T2.30** Create default `goshort.toml` with all options documented
- [ ] **T2.31** Write `internal/config/config_test.go`:
  - Test defaults when no file/env
  - Test TOML file parsing
  - Test env var override
- [ ] **T2.32** Update `cmd/server/main.go` to use config.Load()
- [ ] **T2.33** Verify: all config options work via file and env vars
```

---

## 4. docs/LEARNING.md

### Part 2.5 — Update DI example:

OLD (if Viper mentioned in main.go wiring):
```go
// cmd/server/main.go — THE ONLY FILE that knows all concrete types
func main() {
    db, _ := storage.NewSQLite("./data/goshort.db")
    cache := cache.NewNoop()  // or cache.NewRedis(redisURL)
```

No change needed — the DI example doesn't reference Viper directly.

### Part 6 — GoShort Knowledge Map:

OLD:
```
│ internal/config/               │                                 │
│   config.go                    │ TOML, env vars, viper, defaults │
```

NEW:
```
│ internal/config/               │                                 │
│   config.go                    │ TOML, env vars, koanf, defaults │
```

### Appendix C — Glossary (if Viper mentioned):

No Viper entry exists in glossary. No change needed.

---

## 5. README.md

### Technology mention — if exists in Architecture section:

The README doesn't explicitly mention Viper by name. The configuration section
shows the TOML format which is library-agnostic. No change needed.

### Development section — if Viper mentioned:

No Viper mention in Development section. No change needed.

---

## 6. CC Prompt for Milestone 2.6 (updated for Koanf)

Use this prompt when implementing Milestone 2.6:

```
Implement Milestone 2.6: Configuration (Koanf) from @docs/ROADMAP.md.

Context: @CLAUDE.md, @docs/DESIGN.md Section 16 (Configuration).

## Install koanf (modular — only what we need):
go get -u github.com/knadh/koanf/v2
go get -u github.com/knadh/koanf/providers/file
go get -u github.com/knadh/koanf/providers/env/v2
go get -u github.com/knadh/koanf/parsers/toml/v2

## Create: internal/config/config.go

### Config structs (use `koanf` struct tags, NOT `mapstructure`):

type Config struct {
    Server    ServerConfig    `koanf:"server"`
    Storage   StorageConfig   `koanf:"storage"`
    Cache     CacheConfig     `koanf:"cache"`
    Auth      AuthConfig      `koanf:"auth"`
    RateLimit RateLimitConfig `koanf:"rate_limit"`
    Shortener ShortenerConfig `koanf:"shortener"`
    Logging   LoggingConfig   `koanf:"logging"`
}

type ServerConfig struct {
    Port    int    `koanf:"port"`
    BaseURL string `koanf:"base_url"`
}

type StorageConfig struct {
    Driver     string `koanf:"driver"`
    SQLitePath string `koanf:"sqlite_path"`
}

type CacheConfig struct {
    Driver   string `koanf:"driver"`
    RedisURL string `koanf:"redis_url"`
}

type AuthConfig struct {
    APIKey string `koanf:"api_key"`
}

type RateLimitConfig struct {
    Enabled           bool `koanf:"enabled"`
    RequestsPerMinute int  `koanf:"requests_per_minute"`
}

type ShortenerConfig struct {
    CodeLength    int    `koanf:"code_length"`
    DefaultExpiry string `koanf:"default_expiry"`
}

type LoggingConfig struct {
    Level  string `koanf:"level"`
    Format string `koanf:"format"`
}

### Load function:

func Load(path string) (*Config, error) {
    k := koanf.New(".")

    // 1. Load defaults using confmap provider
    k.Load(confmap.Provider(map[string]any{
        "server.port":                8080,
        "server.base_url":           "http://localhost:8080",
        "storage.driver":            "sqlite",
        "storage.sqlite_path":       "./data/goshort.db",
        "cache.driver":              "none",
        "cache.redis_url":           "redis://localhost:6379",
        "shortener.code_length":     6,
        "shortener.default_expiry":  "0",
        "logging.level":             "info",
        "logging.format":            "json",
        "rate_limit.requests_per_minute": 60,
    }, "."), nil)

    // 2. Load TOML file (optional — don't fail if missing)
    if path != "" {
        if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
            return nil, fmt.Errorf("load config file %s: %w", path, err)
        }
    }

    // 3. Load env vars with GOSHORT_ prefix
    // GOSHORT_SERVER_PORT=9090 → server.port = 9090
    k.Load(env.Provider(".", env.Opt{
        Prefix: "GOSHORT_",
        TransformFunc: func(k, v string) (string, any) {
            key := strings.ReplaceAll(
                strings.ToLower(strings.TrimPrefix(k, "GOSHORT_")),
                "_", ".",
            )
            return key, v
        },
    }), nil)

    // 4. Unmarshal into Config struct
    var cfg Config
    if err := k.Unmarshal("", &cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }

    return &cfg, nil
}

### Required imports:
import (
    "fmt"
    "strings"

    "github.com/knadh/koanf/v2"
    "github.com/knadh/koanf/providers/confmap"
    "github.com/knadh/koanf/providers/env/v2"
    "github.com/knadh/koanf/providers/file"
    toml "github.com/knadh/koanf/parsers/toml/v2"
)

### Create: goshort.toml (project root, example config)
Same TOML content as design doc Section 16 — format unchanged.
Add comments explaining each option.

### Create: internal/config/config_test.go
- TestLoad_Defaults: Load("") → verify all defaults correct
- TestLoad_TOMLFile: create temp TOML file → Load(path) → verify values
- TestLoad_EnvOverride: set GOSHORT_SERVER_PORT=9090 → verify port=9090
- TestLoad_EnvPrefix: set GOSHORT_CACHE_DRIVER=memory → verify cache.driver="memory"
- TestLoad_FileNotFound: Load("nonexistent.toml") → error

### Update: cmd/server/main.go
- Add --config flag (default: "" meaning use defaults + env only)
- Replace all os.Getenv/hardcoded values with cfg := config.Load(configPath)
- Initialize everything from cfg struct

### After implementation:
1. go build ./... — compiles
2. go test ./... — all pass
3. make run — starts with defaults
4. GOSHORT_SERVER_PORT=9090 make run — starts on 9090
5. Create goshort.toml with port=3000 → make run --config=goshort.toml → port 3000
6. golangci-lint run ./... — no issues
```

---

## Key Differences: Koanf vs Viper in Code

| Aspect | Viper | Koanf |
|--------|-------|-------|
| Struct tags | `mapstructure:"field"` | `koanf:"field"` |
| Key case | Forces lowercase | Case-sensitive (respects spec) |
| Env binding | `viper.SetEnvPrefix` + `AutomaticEnv` | `env.Provider` with transform func |
| Defaults | `viper.SetDefault("key", val)` | `confmap.Provider(map)` |
| Load file | `viper.SetConfigFile(path)` | `k.Load(file.Provider(path), parser)` |
| Unmarshal | `viper.Unmarshal(&cfg)` | `k.Unmarshal("", &cfg)` |
| Dependencies | 20+ transitive deps | Core + 3 modules (file, env, toml) |
| Binary bloat | +313% vs baseline | Near baseline |
