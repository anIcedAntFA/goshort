# 🧪 TESTING.md — The Complete Guide to Testing in Go

> **By:** A Principal Engineer who has seen too many test suites that either test nothing or test everything wrong.
>
> **For:** Developers who want to understand testing deeply — not just "write a test", but *think* in tests. Go in general, GoShort specifically.
>
> **How to use:** Read linearly on first pass. Then use as a reference. Each section has a `🎯 In GoShort:` annotation showing exactly where each concept appears in the codebase.

---

## 📖 Table of Contents

- [Part 0: Mindset — How to Think About Testing](#part-0-mindset--how-to-think-about-testing)
- [Part 1: Foundations — Vocabulary & Anatomy](#part-1-foundations--vocabulary--anatomy)
- [Part 2: Go Testing Toolkit](#part-2-go-testing-toolkit)
- [Part 3: Table-Driven Tests — The Go Way](#part-3-table-driven-tests--the-go-way)
- [Part 4: Isolation Strategies — Mock, Fake, Stub, Spy](#part-4-isolation-strategies--mock-fake-stub-spy)
- [Part 5: HTTP Testing](#part-5-http-testing)
- [Part 6: Database Testing](#part-6-database-testing)
- [Part 7: Concurrency Testing](#part-7-concurrency-testing)
- [Part 8: Fuzz & Property Testing](#part-8-fuzz--property-testing)
- [Part 9: Benchmarks — Performance Testing](#part-9-benchmarks--performance-testing)
- [Part 10: Advanced Patterns](#part-10-advanced-patterns)
- [Part 11: GoShort Test Suite — Complete Map](#part-11-goshort-test-suite--complete-map)
- [Part 12: Anti-Pattern Field Guide](#part-12-anti-pattern-field-guide)
- [Part 13: CI & Infrastructure](#part-13-ci--infrastructure)
- [Part 14: TDD — Test-Driven Development](#part-14-tdd--test-driven-development)
- [Part 15: Property-Based vs Example-Based Testing](#part-15-property-based-vs-example-based-testing)
- [Part 16: stretchr/testify — The Ecosystem Standard](#part-16-stretchrtestify--the-ecosystem-standard)
- [Part 17: Test Tooling — Generators, Linters, Utilities](#part-17-test-tooling--generators-linters-utilities)
- [Appendix A: Test Naming Convention](#appendix-a-test-naming-convention)
- [Appendix B: Cheat Sheet](#appendix-b-cheat-sheet)
- [Appendix C: Glossary](#appendix-c-glossary)

---

## Part 0: Mindset — How to Think About Testing

### 🧘 The Core Philosophy

```
╭────────────────────────────────────────────────────────────╮
│                                                            │
│   "A test is not proof that code works.                    │
│    A test is proof that code works in this specific way,   │
│    under these specific conditions."                       │
│                                                            │
│   Tests don't eliminate bugs. They make bugs               │
│   reproducible — and reproducible bugs get fixed.          │
│                                                            │
╰────────────────────────────────────────────────────────────╯
```

Most developers think about testing wrong. They think: "I wrote the code. Now I need to write tests to prove it works." That mindset leads to tests that only verify the happy path and break whenever you refactor.

The right mindset: **Tests are the first client of your code.** Writing a test forces you to think about how your code will be used — before you write it.

### 🎯 What Tests Actually Do

Tests serve four distinct purposes. Most people only think about #1:

```
┌──────────────────┬───────────────────────────────────────────────┐
│  Purpose         │  What It Means                                │
├──────────────────┼───────────────────────────────────────────────┤
│  1. Verification │  Code does what it should (correctness)       │
│  2. Regression   │  Code keeps doing it after changes            │
│  3. Design       │  Testable code is well-designed code          │
│  4. Documentation│  Tests explain intended behavior to others    │
└──────────────────┴───────────────────────────────────────────────┘
```

> 💡 **ELI5 — Purpose #3:** If your code is hard to test, it's usually because too many things are tangled together. Testing forces untangling. That's why TDD (Test-Driven Development) tends to produce better architecture — you have to think about seams before you write any code.

### 🏔️ The Testing Pyramid

```
         ┌───────────────────┐
         │    E2E / Smoke    │  ← Few, slow, brittle, expensive
         │    (2-5 tests)    │    Test the whole system end-to-end
         └─────────┬─────────┘
    ┌──────────────┴──────────────┐
    │  Integration / HTTP Tests   │  ← Some, medium speed
    │  (30-50 tests)              │    Test components wired together
    └──────────────┬──────────────┘
  ┌────────────────┴────────────────┐
  │       Unit Tests                │  ← Many, fast, isolated
  │       (100-300 tests)           │    Test one thing at a time
  └─────────────────────────────────┘
```

**The pyramid exists for a reason:**
- Unit tests are fast (microseconds). You run them constantly.
- Integration tests are slower (milliseconds). You run them on save or commit.
- E2E tests are slow (seconds). You run them before deploy.

**Inverting the pyramid = pain:**
```
        ┌────────────────────────────────┐
        │         E2E Tests              │  ← Anti-pattern: "Ice Cream Cone"
        │   (hundreds of Selenium tests) │
        └───────────────┬────────────────┘
         ┌──────────────┴───────────┐
         │    Integration Tests     │
         │    (a few)               │
         └──────────────┬───────────┘
           ┌────────────┴──────────┐
           │  Unit Tests (barely)  │  ← No foundation
           └───────────────────────┘
```

Result: slow CI, flaky tests, nobody trusts the suite, everyone skips tests.

### 🌟 Three Qualities of a Great Test Suite

```
  1. FAST         → < 10s for the full unit suite
                    Slow tests don't get run

  2. RELIABLE     → Never fails intermittently
                    Flaky tests are worse than no tests
                    (team ignores all red, including real failures)

  3. MEANINGFUL   → A failing test tells you what broke and why
                    A passing test gives you confidence to ship
```

**GoShort's suite achieves all three:**
- `go test ./...` ≈ 5-8s (fast)
- No time.Sleep-based tests except with safe margins (reliable)
- Named subtests, clear assertions, sentinel errors (meaningful)

### 🧠 The Two Schools: London vs Chicago

Before writing any test, understand which school you're in:

```
┌───────────────────┬──────────────────────────────────────────────┐
│ London School     │ Chicago School                               │
│ (Mockist)         │ (Classicist / Detroit)                       │
├───────────────────┼──────────────────────────────────────────────┤
│ Mock everything   │ Use real dependencies where practical        │
│ outside the unit  │ Only mock external services (DB, network)    │
│                   │                                              │
│ Tests: fast,      │ Tests: slower, catch integration bugs        │
│ isolated, brittle │ more realistic                               │
│ to refactors      │                                              │
│                   │                                              │
│ Good for: service │ Good for: pure functions, small modules,     │
│ layer with many   │ storage implementations with real DB         │
│ error branches    │                                              │
└───────────────────┴──────────────────────────────────────────────┘
```

> 💡 **ELI5:** London School = test your car by checking each part in isolation (engine on stand, wheels on jig). Chicago School = just drive the car and see if it goes.

**GoShort uses both deliberately:**
- `service_test.go` = London School (mock storage, full error injection)
- `storage/sqlite_test.go` = Chicago School (real SQLite `:memory:`)
- `api/handler_test.go` = Chicago School (real service + real SQLite wired to Chi router)

---

## Part 1: Foundations — Vocabulary & Anatomy

### 1.1 Test Types

```
╭─────────────────────────────────────────────────────────────╮
│  UNIT TEST                                                   │
│  Tests one function or type in complete isolation.           │
│  All dependencies replaced with controlled substitutes.      │
│  Speed: microseconds. Scope: single function.                │
╰─────────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────╮
│  INTEGRATION TEST                                            │
│  Tests multiple components wired together.                   │
│  May use real database, real HTTP server, real file system.  │
│  Speed: milliseconds. Scope: package or subsystem.           │
╰─────────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────╮
│  END-TO-END (E2E) TEST                                       │
│  Tests the entire system from the outside.                   │
│  Real server, real DB, real HTTP client.                     │
│  Speed: seconds. Scope: whole application.                   │
╰─────────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────╮
│  FUZZ TEST                                                   │
│  Generates random inputs to find crashes/panics.             │
│  Tests invariants: "no matter the input, this holds true."   │
│  Speed: run continuously. Scope: specific function.          │
╰─────────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────╮
│  BENCHMARK                                                   │
│  Measures performance of a function (ns/op, B/op, allocs).  │
│  Not a correctness test. Detects performance regressions.    │
│  Speed: seconds. Scope: specific hot path.                   │
╰─────────────────────────────────────────────────────────────╯

╭─────────────────────────────────────────────────────────────╮
│  EXAMPLE TEST                                                │
│  Runnable documentation. Output is verified by go test.      │
│  Appears in pkg.go.dev as executable code.                   │
│  Speed: fast. Scope: public API.                             │
╰─────────────────────────────────────────────────────────────╯
```

### 1.2 Anatomy of a Go Test

```go
// File: internal/shortener/validator_test.go
package shortener_test  // ← black-box: tests public API only
                        //   "package foo_test" not "package foo"

import (
    "testing"           // ← stdlib, always needed
    "github.com/anIcedAntFA/goshort/internal/shortener"
)

// TestFoo_Scenario is the naming convention: TestType_Method_Scenario
func TestValidateURL_EmptyString(t *testing.T) {
    t.Parallel()  // ← run concurrently with other tests

    // Arrange
    input := ""

    // Act
    err := shortener.ValidateURL(input)

    // Assert
    if err == nil {
        t.Error("expected error for empty URL, got nil")
        //       ↑ t.Error logs the failure but continues the test
        //         t.Fatal would stop immediately
    }
}
```

**The Three Parts of Every Test:**

```
┌─────────────────────────────────────────────────────┐
│  ARRANGE (Given)                                     │
│  Set up the world: create objects, seed data,        │
│  configure mocks. This is your test's precondition.  │
├─────────────────────────────────────────────────────┤
│  ACT (When)                                          │
│  Call the function you're testing. One action.       │
│  If you're calling multiple things, split the test.  │
├─────────────────────────────────────────────────────┤
│  ASSERT (Then)                                       │
│  Check the result: return value, side effects,       │
│  state changes. Make failures specific and readable. │
└─────────────────────────────────────────────────────┘
```

### 1.3 Black-box vs White-box Testing

```go
// BLACK-BOX (package foo_test) — preferred for most tests
// Tests the public API only. Refactoring internals won't break tests.
package shortener_test

import "github.com/anIcedAntFA/goshort/internal/shortener"

func TestService_Create(t *testing.T) {
    svc := shortener.NewService(...)
    url, err := svc.Create(ctx, req)  // only public methods
    // ...
}
```

```go
// WHITE-BOX (package foo) — use sparingly, for unexported helpers
// Directly access unexported functions. Tests coupled to internals.
package shortener  // ← same package, no import needed

func TestParseExpiresIn(t *testing.T) {
    d, err := parseExpiresIn("7d")  // unexported function
    // ...
}
```

**When to use white-box:**
- Testing an unexported helper that can't be reached via the public API
- Testing internal error branches that require impossible real-world conditions
- Never use it just to avoid writing a proper public API

**🎯 In GoShort:** `service_internal_test.go` tests `parseExpiresIn` (white-box). This function is private and can only be invoked through `Create()` which validates inputs first — making it impossible to test the "zero value" error path via the public API. White-box is the right call here.

### 1.4 The `testing.T` Object — Your Interface to Go's Test Runner

```go
// Error handling
t.Error("message")      // log failure, continue running
t.Errorf("fmt %v", x)  // log failure with formatting, continue
t.Fatal("message")      // log failure, STOP this test immediately
t.Fatalf("fmt %v", x)  // same with formatting

// Logging (only visible when test fails or -v flag)
t.Log("debug info")
t.Logf("value: %v", x)

// Test control
t.Parallel()            // run this test concurrently with others
t.Skip("reason")        // skip this test (counts as skipped, not failed)
t.Skipf("fmt %v", x)   // skip with formatting

// Cleanup
t.Cleanup(func() {      // run after test completes (even if failed)
    db.Close()
})
t.TempDir()             // create + auto-cleanup temp directory
t.Setenv("K", "V")      // set + auto-restore env variable

// Subtests
t.Run("subtest name", func(t *testing.T) {
    // t is a NEW *testing.T scoped to this subtest
})

// Helper marking
t.Helper()              // marks function as helper; failure line points to CALLER
```

**`t.Fatal` vs `t.Error` — when to use which:**

```go
// Use t.Fatal when remaining assertions would panic or be meaningless
url, err := svc.Create(ctx, req)
if err != nil {
    t.Fatalf("Create: %v", err)  // no point continuing if url is nil
}
if url.ShortCode != "abc" {    // url is guaranteed non-nil here
    t.Errorf("ShortCode = %q, want %q", url.ShortCode, "abc")
}

// Use t.Error when you want to collect all failures at once
if resp.StatusCode != 201 {
    t.Errorf("status = %d, want 201", resp.StatusCode)
}
if resp.Body == nil {
    t.Error("body is nil")  // continue to check other fields too
}
```

### 1.5 Test File Naming & Organization

```
internal/shortener/
├── service.go
├── service_impl.go
├── service_test.go              ← black-box tests (package shortener_test)
├── service_internal_test.go     ← white-box tests (package shortener)
├── validator.go
├── validator_test.go            ← black-box
├── model.go
└── errors.go
    errors_test.go               ← tests for error behavior
```

**Rules:**
- Test files end in `_test.go`
- `_internal_test.go` suffix = white-box (convention, not enforced by Go)
- `export_test.go` = expose internals for black-box tests (no `_test` in name)

```go
// internal/api/export_test.go — special file
// NOT a test file (no _test.go suffix won't work here — this IS compiled)
// Actually: it IS a test file with a trick:
package api

// Expose unexported function for testing
var CacheTTL = cacheTTL
```

Wait — the correct way:
```go
// internal/api/export_test.go
package api  // same package, but in a _test.go file
             // Go compiles this ONLY during testing

var CacheTTL = cacheTTL  // now accessible as api.CacheTTL in api_test package
```

---

## Part 2: Go Testing Toolkit

### 2.1 Running Tests

```bash
# Run all tests in the module
go test ./...

# Run with race detector (ALWAYS use this in CI)
go test -race ./...

# Run specific package
go test ./internal/shortener/...

# Run specific test by name (regex)
go test -run TestValidateURL ./internal/shortener/

# Run specific subtest
go test -run "TestValidateURL/empty_string" ./internal/shortener/

# Verbose output (see t.Log, test names)
go test -v ./...

# Run N times (useful to catch flaky tests)
go test -count=5 ./...

# Short mode (skip slow tests that check t.Short())
go test -short ./...

# With build tags
go test -tags redis ./...

# Timeout (default is 10 minutes!)
go test -timeout 30s ./...

# Show coverage
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out  # open in browser
go tool cover -func=coverage.out  # print per-function %

# Combined — the gold standard CI command
go test -race -count=1 -timeout=60s -coverprofile=coverage.out ./...
```

### 2.2 Build Tags — Conditional Test Compilation

```go
//go:build redis
// ↑ This comment MUST be first line (with blank line below)
// This file is only compiled when -tags redis is passed

package cache_test

import "testing"

func TestRedisCache_SetGet(t *testing.T) {
    // This test only runs when: go test -tags redis ./...
}
```

**Common patterns:**
```go
//go:build integration    // slow tests that require external services
//go:build redis          // Redis-dependent tests
//go:build postgres       // PostgreSQL tests
//go:build !race          // tests that are known to be unsafe under race detector
```

**🎯 In GoShort:** `internal/cache/redis_test.go` uses `//go:build redis`. This means `go test ./...` never fails in a Redis-less dev environment. CI runs with `-tags redis` when Redis is available.

### 2.3 `TestMain` — Package-Level Setup & Teardown

```go
// package_test.go (conventional name)
package mypackage_test

import (
    "os"
    "testing"
)

func TestMain(m *testing.M) {
    // Setup: runs ONCE before all tests in this package
    setupDatabase()
    setupFixtures()

    // Run all tests
    code := m.Run()

    // Teardown: runs ONCE after all tests
    cleanupDatabase()

    // Exit with the test result code
    os.Exit(code)
}
```

**Use cases:**
- Starting a test server once for all tests
- Goroutine leak detection with `goleak`
- Global fixture setup (expensive operations you don't want per-test)

```go
// With goleak (CRITICAL for packages with goroutines)
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
    // ↑ This automatically calls m.Run() AND checks for goroutine leaks
    // If any goroutine started during tests is still running after → FAIL
}
```

### 2.4 `t.Cleanup` — The Right Way to Teardown

```go
// ❌ BAD: defer in test body doesn't work correctly in subtests
func TestSomething(t *testing.T) {
    db, _ := sql.Open("sqlite", ":memory:")
    defer db.Close()  // runs when TestSomething returns, NOT when subtest returns

    t.Run("subtest", func(t *testing.T) {
        // db might already be closed here in complex scenarios
    })
}

// ✅ GOOD: t.Cleanup runs when the test (including subtests) finishes
func TestSomething(t *testing.T) {
    db, _ := sql.Open("sqlite", ":memory:")
    t.Cleanup(func() {
        _ = db.Close()  // runs when THIS test (and all its subtests) finish
    })

    t.Run("subtest", func(t *testing.T) {
        // db is guaranteed alive here
    })
}
```

**🎯 In GoShort:** Every test helper that creates a database connection uses `t.Cleanup`:
```go
func newTestStorage(t *testing.T) *storage.SQLiteStorage {
    t.Helper()
    s, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
    if err != nil {
        t.Fatalf("NewSQLiteStorage: %v", err)
    }
    t.Cleanup(func() {
        if err := s.Close(); err != nil {
            t.Errorf("close storage: %v", err)
        }
    })
    return s
}
```

### 2.5 `t.Setenv` and `t.TempDir` — Safe Environment Isolation

```go
// t.Setenv — sets env var AND automatically restores it after the test
func TestConfig_EnvOverride(t *testing.T) {
    t.Setenv("GOSHORT_SERVER_PORT", "9090")
    // After test: GOSHORT_SERVER_PORT is restored to original value
    // Works correctly even with t.Parallel() — each test gets its own env
    // ⚠️ Actually: t.Setenv calls t.Helper() and t.Cleanup() internally
    //             But env vars are process-global! Use with care in parallel tests
    //             Go docs: "It is not safe to call t.Setenv in parallel tests"
}

// t.TempDir — creates a temp directory that is automatically cleaned up
func TestConfig_AutoDiscover(t *testing.T) {
    dir := t.TempDir()
    // dir is something like /tmp/TestConfig_AutoDiscover123456789
    // After test: entire directory is removed
    configPath := filepath.Join(dir, "goshort.toml")
    os.WriteFile(configPath, []byte("[server]\nport = 7777"), 0o600)
}
```

**⚠️ `t.Setenv` and `t.Parallel()`:** Do NOT use `t.Setenv` in parallel tests. Environment variables are process-global. If two parallel tests set the same env var, they interfere with each other.

```go
// ❌ DANGEROUS — parallel + Setenv on same key
func TestA(t *testing.T) {
    t.Parallel()
    t.Setenv("PORT", "8080")  // interferes with TestB
}
func TestB(t *testing.T) {
    t.Parallel()
    t.Setenv("PORT", "9090")  // interferes with TestA
}

// ✅ CORRECT — Setenv tests are NOT parallel
func TestA(t *testing.T) {
    // No t.Parallel()
    t.Setenv("PORT", "8080")
}
```

**🎯 In GoShort:** `config_test.go` uses `t.Setenv` without `t.Parallel()` for multi-word field tests. Single-field tests that don't overlap CAN be parallel — but GoShort conservatively marks all env tests non-parallel.

### 2.6 `t.Skip` — Conditional Test Skipping

```go
// Skip when an external service is unavailable
func TestRedisCache_SetGet(t *testing.T) {
    c, err := cache.NewRedisCache("localhost:6379")
    if err != nil {
        t.Skip("Redis not available:", err)
        // t.Skip immediately stops the test and marks it SKIPPED (not FAILED)
    }
    // ...
}

// Skip in -short mode (for slow tests)
func TestExpensiveOperation(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping slow test in -short mode")
    }
    // ...
}
```

### 2.7 `t.Helper` — Making Failure Messages Useful

```go
// Without t.Helper() — failure points to line INSIDE the helper
func setupUser(t *testing.T) *User {
    u, err := db.CreateUser(...)
    if err != nil {
        t.Fatalf("setup failed: %v", err)
        // ↑ Error: "setup failed: ..." — points to THIS LINE in helper
        //   Not helpful: which test called setupUser?
    }
    return u
}

// With t.Helper() — failure points to the CALLER
func setupUser(t *testing.T) *User {
    t.Helper()  // ← marks this function as a helper
    u, err := db.CreateUser(...)
    if err != nil {
        t.Fatalf("setup failed: %v", err)
        // ↑ Error points to the TEST that called setupUser(), not this line
        //   Much more useful when debugging
    }
    return u
}
```

**Rule:** Every function that calls `t.Fatal`, `t.Error`, etc. should call `t.Helper()` first.

---

## Part 3: Table-Driven Tests — The Go Way

### 3.1 Why Table-Driven Tests?

```
Problem: You have a function with 15 different input cases.

Bad approach: write 15 separate Test functions.
  - 300+ lines of nearly identical code
  - Adding a new case requires a new function
  - Hard to see "what cases are covered" at a glance

Go approach: one test with a table of cases.
  - All cases in one place
  - Adding a case = adding one struct literal
  - Coverage at a glance
```

### 3.2 Basic Pattern

```go
func TestValidateURL(t *testing.T) {
    t.Parallel()

    cases := []struct {
        name    string    // ← name for t.Run — makes failures readable
        input   string
        wantErr bool
    }{
        {name: "valid https",          input: "https://example.com",  wantErr: false},
        {name: "valid http",           input: "http://example.com",   wantErr: false},
        {name: "empty",                input: "",                      wantErr: true},
        {name: "ftp not allowed",      input: "ftp://example.com",    wantErr: true},
        {name: "private ip 192.168",   input: "http://192.168.1.1",   wantErr: true},
        {name: "localhost blocked",    input: "http://localhost",      wantErr: true},
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()  // ← each subtest also parallel

            err := ValidateURL(tc.input)

            if tc.wantErr && err == nil {
                t.Errorf("ValidateURL(%q) = nil, want error", tc.input)
            }
            if !tc.wantErr && err != nil {
                t.Errorf("ValidateURL(%q) = %v, want nil", tc.input, err)
            }
        })
    }
}
```

**Running a specific subtest:**
```bash
go test -run "TestValidateURL/valid_https" ./internal/shortener/
#                              ↑ spaces in test names become underscores in -run
```

### 3.3 Advanced Table Pattern — Rich Case Types

```go
func TestService_Create(t *testing.T) {
    t.Parallel()

    cases := []struct {
        name        string
        req         shortener.CreateRequest
        setupFn     func(store *mockStorage)  // optional setup
        wantErr     error                     // nil = expect success
        checkResult func(t *testing.T, url *shortener.URL)  // optional assertions
    }{
        {
            name: "valid URL generates short code",
            req:  shortener.CreateRequest{URL: "https://example.com"},
            checkResult: func(t *testing.T, url *shortener.URL) {
                t.Helper()
                if url.ShortCode == "" {
                    t.Error("expected non-empty ShortCode")
                }
                if url.IsCustom {
                    t.Error("IsCustom should be false for generated codes")
                }
            },
        },
        {
            name: "invalid URL returns ErrInvalidURL",
            req:  shortener.CreateRequest{URL: "not-a-url"},
            wantErr: shortener.ErrInvalidURL,
        },
        {
            name: "alias taken returns ErrAliasTaken",
            req:  shortener.CreateRequest{URL: "https://example.com", CustomAlias: "taken"},
            setupFn: func(store *mockStorage) {
                // Pre-populate to simulate "taken" state
                store.insertDirect(&shortener.URL{ShortCode: "taken"})
            },
            wantErr: shortener.ErrAliasTaken,
        },
        {
            name: "store error is propagated",
            req:  shortener.CreateRequest{URL: "https://example.com"},
            setupFn: func(store *mockStorage) {
                store.errIncrementCounter = errors.New("db down")
            },
            wantErr: nil, // just check err != nil
        },
    }

    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()

            store := newMockStorage()
            if tc.setupFn != nil {
                tc.setupFn(store)
            }
            svc := shortener.NewService(store, enc)

            url, err := svc.Create(context.Background(), tc.req)

            if tc.wantErr != nil {
                if !errors.Is(err, tc.wantErr) {
                    t.Errorf("err = %v, want wrapping %v", err, tc.wantErr)
                }
                return
            }
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if tc.checkResult != nil {
                tc.checkResult(t, url)
            }
        })
    }
}
```

### 3.4 Boundary Value Analysis — Always Test the Edges

```
For any function with min/max limits:

  TEST:  min-1 (just below valid) → should fail
  TEST:  min   (exact minimum)    → should pass
  TEST:  min+1 (just above min)   → should pass
  TEST:  max-1 (just below max)   → should pass
  TEST:  max   (exact maximum)    → should pass
  TEST:  max+1 (just above max)   → should fail
```

**🎯 In GoShort:** `TestValidateAlias` covers exactly this:
```go
{name: "min_length_3",  alias: "abc",                      wantErr: nil},   // min
{name: "length_2",      alias: "ab",                       wantErr: shortener.ErrInvalidAlias}, // min-1
{name: "max_length_30", alias: strings.Repeat("a", 30),   wantErr: nil},   // max
{name: "length_31",     alias: strings.Repeat("a", 31),   wantErr: shortener.ErrInvalidAlias}, // max+1
```

### 3.5 Naming Convention for Test Cases

```
Good names read like sentences:
  "valid URL returns short code"          ← happy path, states what happens
  "empty URL returns ErrInvalidURL"       ← sad path, includes error type
  "alias already taken returns conflict"  ← context + result
  "store error is propagated"             ← describes the behavior being tested

Bad names:
  "test1"         ← meaningless
  "happy path"    ← vague
  "error"         ← which error?
  "foo_bar_baz"   ← inconsistent style
```

---

## Part 4: Isolation Strategies — Mock, Fake, Stub, Spy

### 4.1 The Vocabulary (Most People Misuse These Terms)

```
┌──────────────┬─────────────────────────────────────────────────────┐
│  Term        │  What it is                                         │
├──────────────┼─────────────────────────────────────────────────────┤
│  Dummy       │  Passed around but never actually used.              │
│              │  e.g., nil logger, empty context                    │
│              │                                                     │
│  Stub        │  Returns hardcoded responses to calls.              │
│              │  e.g., always returns ErrNotFound for GetByCode     │
│              │                                                     │
│  Fake        │  Working implementation, but unsuitable for prod.   │
│              │  e.g., in-memory database, test HTTP server         │
│              │                                                     │
│  Spy         │  Records calls for later assertion.                 │
│              │  e.g., "was this method called with this arg?"      │
│              │                                                     │
│  Mock        │  Pre-programmed with expectations + verification.   │
│              │  In Go: usually means "hand-written test double"    │
│              │  that implements an interface                       │
└──────────────┴─────────────────────────────────────────────────────┘
```

> 💡 **ELI5:** Stub = answering machine that always says "not available." Fake = a real answering machine that works perfectly but runs on batteries (won't survive a power outage). Mock = a robot actor that plays the role of a person, follows a script, and fails if you don't follow the script.

### 4.2 In Go: Interfaces Are the Seam

The entire isolation strategy in Go relies on interfaces:

```go
// 1. Define the interface in the CONSUMER package (not the implementation package)
// internal/shortener/storage.go
package shortener

type Storage interface {
    GetByCode(ctx context.Context, code string) (*URL, error)
    CreateURL(ctx context.Context, params CreateParams) (URL, error)
    // ...
}

// 2. Real implementation in infrastructure package
// internal/storage/sqlite.go
package storage

type SQLiteStorage struct { db *sql.DB }

func (s *SQLiteStorage) GetByCode(ctx context.Context, code string) (*shortener.URL, error) {
    // real SQLite query
}

// 3. Test double in test file
// internal/shortener/service_test.go
package shortener_test

type mockStorage struct {
    urls        map[string]*shortener.URL
    errGetByCode error  // inject errors per method
}

func (m *mockStorage) GetByCode(_ context.Context, code string) (*shortener.URL, error) {
    if m.errGetByCode != nil {
        return nil, m.errGetByCode
    }
    u, ok := m.urls[code]
    if !ok {
        return nil, fmt.Errorf("%w", shortener.ErrNotFound)
    }
    return u, nil
}
```

**The key insight:** The `Service` never knows whether it's talking to SQLite or a map. The interface is the seam that makes this possible.

### 4.3 Hand-Written Mock vs Generated Mock

```
┌─────────────────────────┬─────────────────────────────────────────┐
│  Hand-Written Mock      │  Generated Mock (mockgen/moq)            │
├─────────────────────────┼─────────────────────────────────────────┤
│  You write it yourself  │  Tool generates from interface           │
│  Full control           │  Less code to maintain                  │
│  Understand every line  │  Can become noisy (gomock matchers)      │
│  Natural Go code        │  Framework dependency                    │
│  Refactor safely        │  Regenerate on interface change          │
│                         │                                         │
│  Best for: small        │  Best for: large interfaces, teams that  │
│  interfaces (1-5 methods│  need strict call verification           │
│  )                      │                                         │
└─────────────────────────┴─────────────────────────────────────────┘
```

**GoShort uses hand-written mocks** — the `Storage` interface has 9 methods which is larger than typical, but the `mockStorage` in `service_test.go` is clean and readable because it's just a map with per-method error injection.

### 4.4 Per-Method Error Injection Pattern

This is GoShort's pattern and it's excellent:

```go
type mockStorage struct {
    mu      sync.Mutex
    urls    map[string]*shortener.URL
    counter int64

    // Each method has its own error field.
    // Set it to inject an error for that specific method.
    errListURLs         error
    errCountURLs        error
    errCreateURL        error
    errIncrementCounter error
    errGetByCode        error
}

func (m *mockStorage) GetByCode(_ context.Context, code string) (*shortener.URL, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.errGetByCode != nil {  // ← error injection point
        return nil, m.errGetByCode
    }
    // normal logic...
}
```

**Usage in tests:**
```go
func TestService_Create_StoreError(t *testing.T) {
    store := newMockStorage()
    store.errCreateURL = errors.New("db write failed")  // inject!

    svc := shortener.NewService(store, enc)
    _, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})

    if err == nil {
        t.Fatal("expected error, got nil")
    }
}
```

### 4.5 Fakes — Real Implementations for Testing

A fake is a real working implementation that's not production-suitable:

```go
// MemoryCache is a FAKE (not a mock)
// It's a real, working cache implementation — just in-process, not Redis.
// Used in handler tests when we need cache behavior but not Redis.
type MemoryCache struct {
    m sync.Map
}

func (c *MemoryCache) Get(ctx context.Context, key string) (string, bool) {
    // real implementation, not hardcoded
}
```

**Where fakes shine:**
```
SQLite :memory:  → real database behavior, no file I/O
httptest.Server  → real HTTP server, no network
MemoryCache      → real cache behavior, no Redis
```

**🎯 In GoShort:**
- `cache.MemoryCache` = fake cache (real behavior, in-process)
- `cache.NoopCache` = stub (always returns "not found")
- `storage.SQLiteStorage` with `:memory:` DSN = fake database (real SQL, in-memory)
- `httptest.NewServer` = fake HTTP server (real HTTP, in-process)

### 4.6 When to Mock vs When to Use Real Dependencies

```
Use MOCKS when:
  ✓ Testing error branches that are hard/impossible to trigger with real deps
  ✓ The real dependency is slow (network calls, disk I/O)
  ✓ You need precise control (inject specific error at specific call #3)
  ✓ The dependency has non-deterministic behavior (time, random)

Use REAL DEPENDENCIES when:
  ✓ The dependency is fast enough (SQLite :memory: is ~1ms)
  ✓ You want to catch real integration bugs (constraint violations, SQL mistakes)
  ✓ The dependency is a fake (httptest.Server, in-memory DB)
  ✓ You're testing the integration between components

Use BOTH (GoShort's approach):
  ✓ Service layer: mock storage (test business logic in isolation)
  ✓ HTTP layer: real service + real SQLite (test the integration)
  ✓ Storage layer: real SQLite :memory: (test SQL queries)
```

---

## Part 5: HTTP Testing

### 5.1 The `httptest` Package — No Network Required

```go
import "net/http/httptest"

// httptest.NewRecorder — captures the HTTP response
rec := httptest.NewRecorder()
// rec.Code       → status code (default 200)
// rec.Header()   → response headers
// rec.Body       → *bytes.Buffer with response body

// httptest.NewRequest — creates an *http.Request without a real network
req := httptest.NewRequest(http.MethodPost, "/api/v1/urls", strings.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("X-API-Key", "test-key")

// Execute: call your handler (or router) directly
router.ServeHTTP(rec, req)

// Assert
if rec.Code != http.StatusCreated {
    t.Errorf("status = %d, want 201", rec.Code)
}
```

**🎯 In GoShort:** `handler_test.go` has a `serve` helper that wraps this:
```go
func serve(t *testing.T, router http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
    t.Helper()
    var req *http.Request
    if body != nil {
        req = httptest.NewRequest(method, path, bytes.NewReader(body))
        req.Header.Set("Content-Type", "application/json")
    } else {
        req = httptest.NewRequest(method, path, http.NoBody)
    }
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)
    return rec
}
```

### 5.2 `httptest.NewServer` — For Client Testing

When testing an HTTP *client* (like GoShort's CLI), you need a fake server:

```go
// Client-side test: CLI calls the API
func TestAPIClient_CreateURL(t *testing.T) {
    t.Parallel()

    // Create a fake server that responds like the real GoShort API
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Assertions about the incoming request
        if r.Method != http.MethodPost {
            t.Errorf("method = %s, want POST", r.Method)
        }
        if r.Header.Get("X-API-Key") != "test-key" {
            t.Errorf("missing API key header")
        }

        // Return a canned response
        w.WriteHeader(http.StatusCreated)
        _, _ = json.NewEncoder(w).Encode(CreateResponse{
            ShortCode: "abc123",
            ShortURL:  "http://localhost:8080/abc123",
        })
    }))
    defer srv.Close()  // cleanup after test

    // Point the client at the fake server
    client := NewAPIClient(srv.URL, "test-key")

    resp, err := client.CreateURL(context.Background(), CreateRequest{URL: "https://example.com"})
    if err != nil {
        t.Fatalf("CreateURL: %v", err)
    }
    if resp.ShortCode != "abc123" {
        t.Errorf("ShortCode = %q, want abc123", resp.ShortCode)
    }
}
```

### 5.3 Full-Stack HTTP Test Pattern

Testing a complete request → response flow:

```go
func setupTestDeps(t *testing.T) http.Handler {
    t.Helper()

    // Real dependencies
    store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
    if err != nil {
        t.Fatalf("storage: %v", err)
    }
    t.Cleanup(func() { _ = store.Close() })

    enc, _ := encoder.NewSqidsEncoder(4)
    svc := shortener.NewService(store, enc)
    handler := api.NewHandler(svc, cache.NewNoopCache(), slog.Default(), "http://localhost:8080")

    return api.NewRouter(handler, api.RouterConfig{})
}

func TestCreateURL_ThenRedirect(t *testing.T) {
    t.Parallel()
    router := setupTestDeps(t)

    // Step 1: Create
    body := `{"url": "https://example.com"}`
    createRec := serve(t, router, "POST", "/api/v1/urls", []byte(body))
    if createRec.Code != http.StatusCreated {
        t.Fatalf("create failed: %d", createRec.Code)
    }
    var created struct{ ShortCode string `json:"short_code"` }
    json.NewDecoder(createRec.Body).Decode(&created)

    // Step 2: Redirect
    redirectRec := serve(t, router, "GET", "/"+created.ShortCode, nil)
    if redirectRec.Code != http.StatusFound {
        t.Fatalf("redirect failed: %d", redirectRec.Code)
    }
    if loc := redirectRec.Header().Get("Location"); loc != "https://example.com" {
        t.Errorf("Location = %q, want https://example.com", loc)
    }
}
```

### 5.4 Testing Middleware

Middleware is tricky to test in isolation. Two approaches:

**Approach A: Test through the router (preferred)**
```go
func TestAuthMiddleware_MissingKey(t *testing.T) {
    t.Parallel()
    router := setupTestDepsWithAuth(t, "secret-key")

    req := httptest.NewRequest("GET", "/api/v1/urls", nil)
    // Note: no X-API-Key header
    rec := httptest.NewRecorder()
    router.ServeHTTP(rec, req)

    if rec.Code != http.StatusUnauthorized {
        t.Errorf("status = %d, want 401", rec.Code)
    }
}
```

**Approach B: Test the middleware function directly**
```go
func TestAuthMiddleware_Unit(t *testing.T) {
    t.Parallel()

    // A simple "next" handler that always returns 200
    next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusOK)
    })

    // Wrap with our middleware
    handler := AuthMiddleware("secret")(next)

    // Test: missing key
    req := httptest.NewRequest("GET", "/", nil)
    rec := httptest.NewRecorder()
    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusUnauthorized {
        t.Errorf("status = %d, want 401", rec.Code)
    }
}
```

---

## Part 6: Database Testing

### 6.1 The Golden Rule: `:memory:` SQLite

```go
// ✅ CORRECT — in-memory, no file I/O, auto-cleanup
store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")

// ❌ BAD — file on disk, need cleanup, can interfere between tests
store, err := storage.NewSQLiteStorage(context.Background(), "/tmp/test.db")

// ❌ WORSE — shared state between tests, parallel tests fight each other
var globalDB *sql.DB
```

**Why `:memory:` is perfect for testing:**
- Each test gets a fresh, empty database
- No disk I/O — fast as in-memory operations
- Auto-cleaned when `db.Close()` is called
- Parallel tests never share state

### 6.2 Test Data Setup Patterns

**Pattern 1: Direct insertion (for storage layer tests)**
```go
func TestGetByCode_NotFound(t *testing.T) {
    t.Parallel()
    s := newTestStorage(t)

    // No setup needed — empty DB
    _, err := s.GetByCode(context.Background(), "nonexistent")
    if !errors.Is(err, shortener.ErrNotFound) {
        t.Errorf("got %v, want ErrNotFound", err)
    }
}

func TestGetByCode_Found(t *testing.T) {
    t.Parallel()
    s := newTestStorage(t)
    ctx := context.Background()

    // Direct insertion for setup
    _, err := s.CreateURL(ctx, shortener.CreateParams{
        ShortCode:   "abc123",
        OriginalURL: "https://example.com",
    })
    if err != nil {
        t.Fatalf("setup: CreateURL: %v", err)
    }

    // Now test
    got, err := s.GetByCode(ctx, "abc123")
    // ...
}
```

**Pattern 2: Service-level setup (for handler tests)**
```go
func TestDeleteURL_ThenGet(t *testing.T) {
    t.Parallel()
    router := setupTestDeps(t)

    // Create via the API (not direct DB insertion)
    createBody := `{"url": "https://example.com"}`
    createRec := serve(t, router, "POST", "/api/v1/urls", []byte(createBody))
    created := decodeJSON[createURLResp](t, createRec)

    // Delete
    deleteRec := serve(t, router, "DELETE", "/api/v1/urls/"+created.ShortCode, nil)
    if deleteRec.Code != http.StatusNoContent {
        t.Fatalf("delete: %d", deleteRec.Code)
    }

    // Verify gone
    getRec := serve(t, router, "GET", "/api/v1/urls/"+created.ShortCode, nil)
    if getRec.Code != http.StatusNotFound {
        t.Errorf("after delete: status = %d, want 404", getRec.Code)
    }
}
```

**Pattern 3: `insertDirect` for impossible states**
```go
// Some states cannot be reached through the normal service API.
// Example: an already-expired URL (Create validates positive duration).
// Solution: bypass the service and insert directly into the storage.

func (m *mockStorage) insertDirect(u *shortener.URL) {
    m.mu.Lock()
    defer m.mu.Unlock()
    cp := *u
    m.urls[u.ShortCode] = &cp
}

// In test:
func TestGetByCode_Expired(t *testing.T) {
    t.Parallel()
    svc, store := newTestService(t)

    past := time.Now().Add(-time.Hour)
    store.insertDirect(&shortener.URL{
        ShortCode:   "expired",
        OriginalURL: "https://example.com",
        ExpiresAt:   &past,
    })

    _, err := svc.GetByCode(context.Background(), "expired")
    if !errors.Is(err, shortener.ErrExpired) {
        t.Errorf("got %v, want ErrExpired", err)
    }
}
```

### 6.3 Testing with Real Redis — Build Tags

```go
//go:build redis

package cache_test

import (
    "os"
    "testing"
)

// Helper: skip if Redis not available
func newTestRedis(t *testing.T) *cache.RedisCache {
    t.Helper()
    addr := os.Getenv("TEST_REDIS_ADDR")
    if addr == "" {
        addr = "localhost:6379"
    }
    c, err := cache.NewRedisCache(addr)
    if err != nil {
        t.Skip("Redis not available:", err)
    }
    t.Cleanup(func() { _ = c.Close() })
    return c
}
```

**Why build tags for external services:**
```
Without build tag:
  go test ./...
  → FAIL (Redis not running on dev machine)
  → Everyone installs Redis just to run tests
  → CI needs Redis on every job

With build tag:
  go test ./...              → passes (Redis tests skipped)
  go test -tags redis ./...  → full suite with Redis
  CI: add Redis service, run with -tags redis
```

### 6.4 Key Isolation: Preventing Test Pollution

When multiple tests share a database (even with parallel tests on the same `:memory:` DB), test pollution can occur:

```go
// Problem: test A creates "abc", test B also creates "abc" → conflict
// Solution: use unique keys per test

func TestRedisCache_SetGet(t *testing.T) {
    t.Parallel()
    c := newTestRedis(t)

    // ❌ BAD: fixed key, conflicts with other parallel tests
    _ = c.Set(ctx, "my-key", "value", time.Minute)

    // ✅ GOOD: key includes test name
    key := "test:redis:" + t.Name()
    _ = c.Set(ctx, key, "value", time.Minute)
}
```

**🎯 In GoShort's Redis tests:**
```go
key := "test:redis:setget:" + t.Name()
// t.Name() = "TestRedisCache_SetGet" → unique per test
```

---

## Part 7: Concurrency Testing

### 7.1 The Race Detector — Your Best Friend

```bash
go test -race ./...
```

The race detector instruments your code to detect data races at runtime. It's not perfect (only catches races that actually occur during the run), but it catches most races.

**What a race looks like:**
```
==================
WARNING: DATA RACE
Write at 0x... by goroutine 7:
  main.incrementCounter()
      /path/to/file.go:42 +0x4c

Previous read at 0x... by goroutine 6:
  main.readCounter()
      /path/to/file.go:38 +0x44

Goroutine 7 (running) created at:
  main.main()
      /path/to/file.go:28 +0x84
==================
```

**Always run with `-race` in CI. Period.**

### 7.2 Testing Concurrent Access

```go
func TestMemoryCache_Concurrent(t *testing.T) {
    t.Parallel()
    c := cache.NewMemoryCache()
    ctx := context.Background()

    const goroutines = 50
    done := make(chan struct{}, goroutines)

    for range goroutines {
        go func() {
            defer func() { done <- struct{}{} }()
            // All goroutines race on the same key
            _ = c.Set(ctx, "shared-key", "value", time.Minute)
            _, _ = c.Get(ctx, "shared-key")
            _ = c.Delete(ctx, "shared-key")
        }()
    }

    // Wait for all goroutines
    for range goroutines {
        <-done
    }
    // If -race doesn't fire, the test passes
    // This test by itself doesn't assert much — it RELIES on the race detector
}
```

### 7.3 Testing Goroutine Cleanup — `goleak`

One of the most common production bugs: goroutines that don't stop when they should.

```go
// main_test.go
package api_test

import (
    "testing"
    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

**What goleak does:**
1. Before tests: records all currently running goroutines
2. After all tests complete: checks if any NEW goroutines are still running
3. If yes: FAIL with a report of which goroutines leaked

**Common goroutine leaks:**
```go
// ❌ LEAK: goroutine never exits
go func() {
    for {
        doWork()  // no exit condition, no context check
    }
}()

// ✅ CORRECT: goroutine respects context cancellation
go func() {
    for {
        select {
        case <-ctx.Done():
            return  // exits when context is cancelled
        default:
            doWork()
        }
    }
}()
```

**Handling expected goroutines:**
```go
func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m,
        // Ignore known background goroutines from dependencies
        goleak.IgnoreTopFunction("github.com/some/lib.backgroundWorker"),
        // Ignore goroutines that finish "soon" after test
        goleak.IgnoreCurrent(),
    )
}
```

### 7.4 Testing Time-Dependent Code

```go
// ❌ BAD: time.Sleep is flaky
func TestCacheExpiry(t *testing.T) {
    c := NewMemoryCache()
    c.Set(ctx, "key", "val", 50*time.Millisecond)
    time.Sleep(100*time.Millisecond)  // 2× margin, still flaky on slow CI
    _, ok := c.Get(ctx, "key")
    if ok {
        t.Error("expected expired")
    }
}

// ✅ BETTER: 3× margin (still not perfect)
time.Sleep(3 * 50*time.Millisecond)  // 150ms for 50ms TTL

// ✅ BEST (Go 1.24+): testing/synctest — deterministic fake time
func TestCacheExpiry(t *testing.T) {
    synctest.Run(func() {
        c := NewMemoryCacheWithClock(synctest.Now)  // inject fake clock
        c.Set(ctx, "key", "val", 50*time.Millisecond)
        synctest.Advance(100*time.Millisecond)  // advance fake time
        _, ok := c.Get(ctx, "key")
        if ok {
            t.Error("expected expired")
        }
    })
}
```

**GoShort's approach:** Uses 3× margin (Option B) for now. Migrating to `testing/synctest` is a future improvement (requires clock injection in `MemoryCache`).

---

## Part 8: Fuzz & Property Testing

### 8.1 What is Fuzzing?

```
Normal testing:                    Fuzzing:
  You provide inputs               Go generates inputs
  You know what to expect          You only know invariants

  TestValidateURL("ftp://x") →     FuzzValidateURL(random) →
  assert err != nil                assert "never panics"
```

**Fuzzing finds:**
- Panics / nil pointer dereferences
- Infinite loops
- Unexpected behavior on weird inputs (null bytes, unicode, very long strings)

**Fuzzing does NOT find:**
- Logic errors (you must define what "correct" means)
- Performance issues

### 8.2 Writing a Fuzz Test

```go
func FuzzValidateURL(f *testing.F) {
    // 1. Seed corpus — starting inputs for the fuzzer
    // These run even without -fuzz flag (like unit tests)
    f.Add("https://example.com")       // valid
    f.Add("")                           // empty
    f.Add("javascript:alert(1)")        // dangerous
    f.Add("http://127.0.0.1")           // private IP
    f.Add(strings.Repeat("a", 3000))   // very long

    // 2. Fuzz function — called with generated inputs
    f.Fuzz(func(t *testing.T, input string) {
        // Invariant: this must NEVER panic
        // The fuzzer will try to make it panic
        err := shortener.ValidateURL(input)
        _ = err  // we don't care about the result, just "no panic"
    })
}
```

**Running fuzz tests:**
```bash
# Run corpus tests only (fast, runs as part of go test)
go test ./internal/shortener/

# Run actual fuzzing for 30 seconds
go test -fuzz=FuzzValidateURL -fuzztime=30s ./internal/shortener/

# Run fuzzing until it finds a failure
go test -fuzz=FuzzValidateURL ./internal/shortener/

# After finding a failure — failure is saved to testdata/fuzz/
# Re-run the specific failure:
go test -run=FuzzValidateURL/testdata/fuzz/FuzzValidateURL/abc123 ./internal/shortener/
```

### 8.3 Stronger Fuzz: Roundtrip Invariants

```go
func FuzzSqidsEncoder_Encode(f *testing.F) {
    enc, _ := encoder.NewSqidsEncoder(6)

    f.Add(int64(0))
    f.Add(int64(1))
    f.Add(int64(math.MaxInt64))
    f.Add(int64(-1))

    f.Fuzz(func(t *testing.T, id int64) {
        code, err := enc.Encode(id)
        if err != nil {
            return  // negative IDs error — expected
        }
        if code == "" {
            t.Errorf("Encode(%d) returned empty code without error", id)
        }

        // ROUNDTRIP INVARIANT: Decode(Encode(n)) must equal n
        got, err := enc.Decode(code)
        if err != nil {
            t.Errorf("Decode(Encode(%d)) failed: %v", id, err)
        }
        if got != id {
            t.Errorf("roundtrip broken: %d → %q → %d", id, code, got)
        }
    })
}
```

**Types of invariants for fuzzing:**
```
ROUNDTRIP:   encode(decode(x)) == x  or  decode(encode(x)) == x
IDEMPOTENT:  f(f(x)) == f(x)
NEVER PANIC: no input causes panic
BOUNDED:     len(f(x)) <= maxLength
PURE:        f(x) == f(x) (same input always same output)
```

---

## Part 9: Benchmarks — Performance Testing

### 9.1 Writing Benchmarks

```go
func BenchmarkSqidsEncoder_Encode(b *testing.B) {
    enc, _ := encoder.NewSqidsEncoder(6)

    b.ResetTimer()  // don't include setup time in measurements

    for i := range b.N {
        // b.N is adjusted by the framework to get stable measurements
        _, _ = enc.Encode(int64(i))
    }
}
```

**Running benchmarks:**
```bash
# Run all benchmarks (no output by default with -bench=.)
go test -bench=. ./internal/encoder/

# Run with memory allocation stats
go test -bench=. -benchmem ./internal/encoder/

# Run for longer for stability
go test -bench=. -benchtime=5s ./internal/encoder/

# Run a specific benchmark
go test -bench=BenchmarkSqidsEncoder_Encode ./internal/encoder/

# Run multiple times for variance analysis
go test -bench=. -count=5 ./internal/encoder/
```

**Reading benchmark output:**
```
BenchmarkSqidsEncoder_Encode-8   5000000   234 ns/op   64 B/op   2 allocs/op
                            ↑            ↑           ↑          ↑
                    GOMAXPROCS     ns per op   bytes/call  allocs/call
```

### 9.2 Comparing Benchmarks with `benchstat`

```bash
# Install
go install golang.org/x/perf/cmd/benchstat@latest

# Baseline (before change)
go test -bench=. -count=10 ./internal/... > before.txt

# After change
go test -bench=. -count=10 ./internal/... > after.txt

# Compare
benchstat before.txt after.txt
```

Output:
```
name                          old time/op  new time/op  delta
SqidsEncoder_Encode-8         234ns ± 2%   198ns ± 1%  -15.38% (p=0.000 n=10+10)
SQLiteStorage_GetByCode-8    1.23ms ± 3%  1.21ms ± 2%   -1.63% (p=0.041 n=10+10)
```

### 9.3 Benchmark Patterns

```go
// Pattern 1: Simple loop
func BenchmarkSimple(b *testing.B) {
    for range b.N {
        result := doSomething()
        _ = result  // prevent compiler from optimizing away
    }
}

// Pattern 2: With setup
func BenchmarkWithSetup(b *testing.B) {
    // Setup: not counted in measurement
    db := setupDB()
    _, _ = db.CreateURL(ctx, params)

    b.ResetTimer()  // ← reset after setup
    for range b.N {
        _, _ = db.GetByCode(ctx, "bench-code")
    }
}

// Pattern 3: Parallel benchmark
func BenchmarkParallel(b *testing.B) {
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            _, _ = doWork()
        }
    })
}

// Pattern 4: Sub-benchmarks (like sub-tests)
func BenchmarkVariants(b *testing.B) {
    for _, size := range []int{10, 100, 1000} {
        b.Run(fmt.Sprintf("size_%d", size), func(b *testing.B) {
            for range b.N {
                processItems(generateItems(size))
            }
        })
    }
}
```

---

## Part 10: Advanced Patterns

### 10.1 White-box Testing via `export_test.go`

Sometimes you need to test unexported behavior from outside the package. The `export_test.go` pattern:

```go
// internal/api/export_test.go
// This file IS a _test.go file, so it's only compiled during testing.
// It uses "package api" (not api_test), so it can access unexported symbols.
// api_test package can import these exported aliases.

package api

// Expose unexported function for testing in api_test package
var CacheTTL = cacheTTL
```

```go
// internal/api/handler_test.go
package api_test

import "github.com/anIcedAntFA/goshort/internal/api"

func TestCacheTTL(t *testing.T) {
    url := &shortener.URL{ExpiresAt: nil}
    got := api.CacheTTL(url)  // uses the exported alias
    if got != 24*time.Hour {
        t.Errorf("TTL = %v, want 24h", got)
    }
}
```

### 10.2 `testing/synctest` — Deterministic Time (Go 1.24+)

```go
import "testing/synctest"

func TestCacheExpiry_Deterministic(t *testing.T) {
    synctest.Run(func() {
        // Inside synctest.Run, time is under our control
        c := cache.NewMemoryCacheWithClock(synctest.Now)

        _ = c.Set(ctx, "key", "value", 100*time.Millisecond)

        // Advance time by 200ms (instant, no real sleep)
        synctest.Advance(200 * time.Millisecond)

        _, ok := c.Get(ctx, "key")
        if ok {
            t.Error("expected expired entry")
        }
    })
}
```

**Requirements for `testing/synctest`:**
- Your code must accept a clock function as a parameter (dependency injection)
- The clock function must be replaceable: `type ClockFn func() time.Time`
- Replace `time.Now()` with the injected clock everywhere in the production code

### 10.3 In-Process Transport Testing (MCP Pattern)

When testing protocol-level behavior without network:

```go
// Official pattern from modelcontextprotocol/go-sdk
func newTestClient(t *testing.T) *sdkmcp.ClientSession {
    t.Helper()

    // Create two connected transports (like two ends of a pipe)
    clientTransport, serverTransport := sdkmcp.NewInMemoryTransports()

    // Connect server
    mcpSrv := mcpserver.NewServer(svc, "http://localhost:8080")
    serverSession, _ := mcpSrv.MCPServer().Connect(ctx, serverTransport, nil)
    t.Cleanup(func() { _ = serverSession.Close() })

    // Connect client
    client := sdkmcp.NewClient(&sdkmcp.Implementation{Name: "test"}, nil)
    clientSession, _ := client.Connect(ctx, clientTransport, nil)
    t.Cleanup(func() { _ = clientSession.Close() })

    return clientSession
}

// Now test any tool call without network
func TestShortenURL(t *testing.T) {
    cs := newTestClient(t)
    res, err := cs.CallTool(ctx, &sdkmcp.CallToolParams{
        Name:      "shorten_url",
        Arguments: map[string]any{"url": "https://example.com"},
    })
    // assert on res...
}
```

**Why this is powerful:**
- Tests the full MCP protocol (request serialization, routing, response)
- No network, no ports, no race conditions on port binding
- Each test gets its own isolated server instance

### 10.4 TCP Lifecycle Tests (Graceful Shutdown)

For servers that must handle graceful shutdown:

```go
func TestServer_GracefulShutdown(t *testing.T) {
    // Find a free port without binding
    ln, _ := net.Listen("tcp", "127.0.0.1:0")
    addr := ln.Addr().String()
    _ = ln.Close()  // release it — tiny window, acceptable in tests

    srv := mcpserver.NewServer(svc, baseURL)
    ctx, cancel := context.WithCancel(context.Background())
    errCh := make(chan error, 1)

    go func() {
        errCh <- srv.RunHTTP(ctx, addr, "")
    }()

    // Wait for server to accept connections
    deadline := time.Now().Add(2 * time.Second)
    for time.Now().Before(deadline) {
        resp, err := http.Post("http://"+addr+"/mcp", "application/json", nil)
        if err == nil {
            resp.Body.Close()
            break
        }
        time.Sleep(5 * time.Millisecond)
    }

    // Cancel context (trigger shutdown)
    cancel()

    // Verify clean shutdown
    select {
    case err := <-errCh:
        if err != nil {
            t.Fatalf("RunHTTP after cancel: %v", err)
        }
    case <-time.After(3 * time.Second):
        t.Fatal("server did not stop after context cancellation")
    }
}
```

### 10.5 Stdin/Stdout Testing (CLI Pattern)

Testing CLI commands that read from stdin or write to stdout:

```go
func TestRunShorten_Stdin(t *testing.T) {
    // No t.Parallel() — mutates os.Stdin global

    // Fake HTTP server
    srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(http.StatusCreated)
        json.NewEncoder(w).Encode(CreateResponse{ShortCode: "abc123", ShortURL: "http://localhost/abc123"})
    }))
    defer srv.Close()

    // Override global server URL
    origServerURL := serverURL
    serverURL = srv.URL
    defer func() { serverURL = origServerURL }()

    // Fake stdin
    pr, pw, _ := os.Pipe()
    _, _ = pw.WriteString("https://example.com/\n")
    _ = pw.Close()
    origStdin := os.Stdin
    os.Stdin = pr
    defer func() { os.Stdin = origStdin; _ = pr.Close() }()

    // Run the command
    cmd := &cobra.Command{}
    cmd.SetContext(context.Background())
    err := runShorten(cmd, nil)  // nil args = read from stdin

    if err != nil {
        t.Fatalf("runShorten: %v", err)
    }
}
```

### 10.6 The Compile-Time Interface Check

```go
// Verifies at COMPILE TIME that T implements Interface.
// If T is missing a method, you get a compile error, not a runtime panic.
var _ shortener.Storage = (*SQLiteStorage)(nil)
//   ↑ blank identifier  ↑ interface       ↑ nil pointer of concrete type
```

This is a zero-cost assertion — the compiler optimizes it away at runtime. If `SQLiteStorage` stops implementing `Storage`, the build fails immediately rather than waiting for a test to run.

**Put these at package level in implementation files, not test files.**

---

## Part 11: GoShort Test Suite — Complete Map

### 11.1 Architecture Overview

```
                    ┌──────────────────────────────────────┐
                    │           TEST STRATEGY               │
                    │                                      │
                    │  Domain (shortener/)                  │
                    │    → London School (mock storage)    │
                    │    → 98.4% coverage                  │
                    │                                      │
                    │  Storage (storage/)                   │
                    │    → Chicago School (real SQLite)    │
                    │    → 91.6% coverage                  │
                    │                                      │
                    │  HTTP (api/)                          │
                    │    → Chicago School (real stack)     │
                    │    → 89.6% coverage                  │
                    │                                      │
                    │  MCP (mcp/)                           │
                    │    → In-process transport             │
                    │    → 87.1% coverage                  │
                    │                                      │
                    │  CLI (cmd/cli/)                       │
                    │    → Fake HTTP server                │
                    │    → 55.4% coverage                  │
                    └──────────────────────────────────────┘
```

### 11.2 `internal/encoder/` — 100% Coverage

**What's tested:** Encode, Decode, roundtrip, non-sequential output, alphanumeric-only, min-length, negative ID error, invalid code error, all-blocked-permutations error.

**Pattern:** Mix of black-box (`sqids_test.go`) and white-box (`sqids_internal_test.go`).

**Key insight:** The white-box test forces `newSqidsEncoder(opts)` with a blocklist covering all permutations — a state impossible to reach through the public `NewSqidsEncoder(minLength)` API.

```go
// sqids_internal_test.go — tests the unexported constructor
func TestSqidsEncoder_Encode_AllPermutationsBlockedReturnsError(t *testing.T) {
    s, _ := sqids.New(sqids.Options{
        Alphabet:  "abc",
        Blocklist: []string{"cab", "abc", "bca"},  // ALL permutations blocked
    })
    enc := &SqidsEncoder{s: s}
    _, err := enc.Encode(0)
    if err == nil {
        t.Fatal("expected error when all permutations are blocked")
    }
}
```

### 11.3 `internal/shortener/` — 98.4% Coverage

**What's tested:**

```
service_test.go (London School):
  ├── Create
  │   ├── generated code (alphanumeric, not custom)
  │   ├── custom alias (code = alias, IsCustom = true)
  │   ├── reserved path (ErrReservedPath)
  │   ├── invalid URL (ErrInvalidURL)
  │   ├── expiry in days (ExpiresAt set correctly)
  │   ├── expiry in hours
  │   ├── invalid expiry (ErrInvalidExpires)
  │   ├── duplicate alias (ErrAliasTaken)
  │   ├── counter error propagated
  │   ├── encoder error propagated
  │   ├── store create error propagated
  │   └── alias check unexpected error propagated
  ├── GetByCode
  │   ├── found → returns URL
  │   ├── not found → ErrNotFound
  │   └── expired → ErrExpired (via insertDirect)
  ├── Delete
  │   ├── success → not findable
  │   └── not found → ErrNotFound
  ├── List
  │   ├── paginated results
  │   ├── default options
  │   ├── store error propagated
  │   └── count error propagated
  └── IncrementClicks
      ├── success → count increases
      └── error propagated

validator_test.go (25 URL cases, 17 alias cases, 20 expiry cases):
  ├── ValidateURL: all schemes, private IPs, length boundaries
  ├── ValidateAlias: length, charset, reserved paths
  └── ValidateExpiresIn: units, boundaries, formats

service_internal_test.go:
  └── parseExpiresIn: 7 cases including edge cases
```

### 11.4 `internal/storage/` — 91.6% Coverage

**Architecture:**
```
sqlite_test.go (black-box, real SQLite :memory:)
  Tests the full CRUD cycle through the public Storage API

storage_internal_test.go (white-box)
  Tests internal type conversion helpers:
  parseTime, parseNullTime, toURL, boolToInt, timeToNullString
  → catches SQLite timestamp format bugs before they reach service layer
```

**Why white-box here?** The `toURL` function handles `"2006-01-02 15:04:05"` format conversion. If this fails, the error appears in `GetByCode` with a confusing message. Testing `toURL` directly gives precise failure messages.

### 11.5 `internal/api/` — 89.6% Coverage

**What makes this test interesting:**
```
setupTestDeps wires: SQLiteStorage(:memory:) → ServiceImpl → Handler → Chi Router
                                                           → NoopCache

This is a deliberate choice:
  - NoopCache: makes tests deterministic (no cache state to manage)
  - Real SQLiteStorage: catches integration bugs between service and storage
  - Real ServiceImpl: tests the full business logic in context
  - Real Chi Router: tests route matching, parameter extraction, middleware
```

**Coverage gaps and why:**
```
cacheTTL: 33.3%
  → Handler always uses NoopCache, so cacheTTL is never called
  → Fix: export via export_test.go and test directly

startLimiterCleanup: 44.4%
  → Background goroutine with 5-minute ticker
  → Can't test without either time.Sleep(5min) or clock injection
  → Decision: accept as technical debt, document it

extractIP: 75%
  → RemoteAddr parsing edge case (no port in address)
  → Low risk: SplitHostPort is stdlib
```

### 11.6 `internal/mcp/` — 87.1% Coverage

**The in-process transport pattern:**
```
tools_test.go setup:
  newTestClient(t) creates:
    1. SQLiteStorage(:memory:)
    2. shortener.Service
    3. mcpserver.NewServer (our MCP server)
    4. sdkmcp.NewInMemoryTransports() → (clientT, serverT)
    5. mcpSrv.MCPServer().Connect(ctx, serverT) → server session
    6. sdkmcp.NewClient().Connect(ctx, clientT) → client session
    
  Tests call cs.CallTool(ctx, &CallToolParams{Name: "shorten_url", ...})
  The call flows: clientT → serverT → our handler → SQLite → response
```

**Coverage gap:**
```
RunStdio: 0%
  → Calls s.server.Run(ctx, &sdkmcp.StdioTransport{})
  → StdioTransport blocks on os.Stdin
  → Untestable without process-level redirection
  → Decision: accept 0%, test via smoke test only
  → Future: refactor to RunWithTransport(ctx, transport)
```

### 11.7 `cmd/cli/` — 55.4% Coverage

**What's tested vs not:**
```
✅ TESTED:
  APIClient all 4 methods (Create, List, Get, Delete)
  APIClient error handling (4xx, 5xx, connection refused, non-JSON body)
  APIClient header behavior (X-API-Key present/absent)
  loadConfig (valid, missing, invalid TOML)
  resolveValue helper
  formatTime helper
  runShorten via stdin and empty stdin

❌ NOT TESTED:
  runDelete (0%)
  runStats (0%)
  runList (0%)  ← only formatTime is tested, not the full command
  initConfig (Cobra init hook, 0%)
  Execute (Cobra entry, 0%)
  version command (4 lines, 0%)
```

**Why the gap?** The `runX` functions are thin wrappers calling `APIClient` methods. They're testable with the same `httptest.Server` pattern used in `client_test.go`. The gap is a test coverage debt, not a structural limitation.

---

## Part 12: Anti-Pattern Field Guide

### 🚫 Anti-Pattern #1: Testing Implementation, Not Behavior

```go
// ❌ BAD: Tests how something is done (brittle to refactors)
func TestService_Create_CallsIncrementCounter(t *testing.T) {
    store := &mockStorage{}
    // ... after Create, assert store.incrementCounterCalled == true
    // If we change the encoding strategy, this breaks even if behavior is correct
}

// ✅ GOOD: Tests what the outcome is (survives refactors)
func TestService_Create_GeneratesShortCode(t *testing.T) {
    svc, _ := newTestService(t)
    url, err := svc.Create(ctx, CreateRequest{URL: "https://example.com"})
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
    if url.ShortCode == "" {
        t.Error("expected non-empty ShortCode")
    }
    // We don't care HOW the short code was generated
}
```

### 🚫 Anti-Pattern #2: Ignoring Errors in Tests

```go
// ❌ BAD: Silently ignoring error — test may give false pass
store, _ := storage.NewSQLiteStorage(ctx, ":memory:")  // error ignored!
url, _ := svc.Create(ctx, req)                          // url might be nil!
if url.ShortCode != "abc" { ... }                       // panic!

// ✅ GOOD: Fail fast on unexpected errors
store, err := storage.NewSQLiteStorage(ctx, ":memory:")
if err != nil {
    t.Fatalf("NewSQLiteStorage: %v", err)
}
url, err := svc.Create(ctx, req)
if err != nil {
    t.Fatalf("Create: %v", err)
}
```

### 🚫 Anti-Pattern #3: `strings.Contains` on Error Messages

```go
// ❌ BAD: Brittle — breaks if error message changes
if !strings.Contains(err.Error(), "not found") {
    t.Error("expected not found error")
}

// ✅ GOOD: Check error identity, not message text
if !errors.Is(err, shortener.ErrNotFound) {
    t.Errorf("got %v, want ErrNotFound", err)
}
```

### 🚫 Anti-Pattern #4: Shared Mutable State Between Tests

```go
// ❌ BAD: global state pollutes parallel tests
var globalStore *mockStorage

func TestA(t *testing.T) {
    t.Parallel()
    globalStore.Create(...)  // race with TestB!
}

func TestB(t *testing.T) {
    t.Parallel()
    globalStore.Create(...)  // race with TestA!
}

// ✅ GOOD: each test creates its own state
func TestA(t *testing.T) {
    t.Parallel()
    store := newMockStorage()  // fresh per test
    store.Create(...)
}
```

### 🚫 Anti-Pattern #5: Sleeping to Wait for Side Effects

```go
// ❌ BAD: flaky, slow, CI-dependent
go func() { doAsyncWork() }()
time.Sleep(time.Second)  // hope async work finished

// ✅ BETTER: use a channel to signal completion
done := make(chan struct{})
go func() {
    defer close(done)
    doAsyncWork()
}()
<-done  // guaranteed to wait

// ✅ BEST: for goroutines you don't control, use short sleep + retry
// (acceptable when the goroutine has a documented SLA)
for i := range 20 {
    if conditionMet() { break }
    if i == 19 { t.Fatal("timed out waiting for condition") }
    time.Sleep(10 * time.Millisecond)
}
```

**GoShort's click increment test** uses `time.Sleep(50ms)` — acceptable because:
1. `IncrementClicks` is a single DB write with no blocking
2. 50ms is >> typical goroutine + DB write latency
3. goleak ensures the goroutine actually completes

### 🚫 Anti-Pattern #6: Testing Private Methods Directly (when unnecessary)

```go
// ❌ BAD: white-box test of something reachable via public API
package shortener  // white-box

func TestValidateURLScheme_internal(t *testing.T) {
    // validateURLScheme is called inside ValidateURL
    // Testing it directly couples tests to internal structure
    err := validateURLScheme("ftp")
    // ...
}

// ✅ GOOD: test via the public API
package shortener_test  // black-box

func TestValidateURL_FTPReturnsError(t *testing.T) {
    err := shortener.ValidateURL("ftp://example.com")
    if !errors.Is(err, shortener.ErrInvalidURL) {
        t.Error("expected ErrInvalidURL")
    }
}

// White-box is ONLY justified for:
// - Functions unreachable via public API
// - Error branches requiring impossible conditions
```

### 🚫 Anti-Pattern #7: Test Size Inflation

```go
// ❌ BAD: one giant test that tests everything
func TestEverything(t *testing.T) {
    // Create URL
    // Verify created
    // Update URL (if feature exists)
    // Redirect
    // Check click count
    // Delete
    // Verify deleted
    // Test error cases
    // Test rate limiting
    // ... 200 lines
}

// ✅ GOOD: each test does one thing
func TestCreateURL_Success(t *testing.T) { ... }
func TestRedirect_ExistingCode(t *testing.T) { ... }
func TestRedirect_IncreasesClickCount(t *testing.T) { ... }
func TestDeleteURL_RemovesFromDB(t *testing.T) { ... }
```

### 🚫 Anti-Pattern #8: Log-and-Return in Tests

```go
// ❌ BAD: logs the error AND returns it — creates noise
func (m *mockStorage) GetByCode(ctx context.Context, code string) (*URL, error) {
    if m.errGetByCode != nil {
        t.Logf("GetByCode returning injected error: %v", m.errGetByCode)  // noise
        return nil, m.errGetByCode
    }
    // ...
}

// ✅ GOOD: just return the error, let the test assert on it
func (m *mockStorage) GetByCode(ctx context.Context, code string) (*URL, error) {
    if m.errGetByCode != nil {
        return nil, m.errGetByCode
    }
    // ...
}
```

### 🚫 Anti-Pattern #9: Coverage Farming

```go
// ❌ BAD: test that exists only to hit lines, not to test behavior
func TestDeleteByCode_Coverage(t *testing.T) {
    s := newTestStorage(t)
    _ = s.DeleteByCode(context.Background(), "any-code")
    // No assertion! Just runs the code to increase coverage %.
}

// ✅ GOOD: test that verifies behavior
func TestDeleteByCode_NonExistentCode(t *testing.T) {
    s := newTestStorage(t)
    err := s.DeleteByCode(context.Background(), "does-not-exist")
    if !errors.Is(err, shortener.ErrNotFound) {
        t.Errorf("expected ErrNotFound, got %v", err)
    }
}
```

### 🚫 Anti-Pattern #10: No t.Helper() on Helpers

```go
// ❌ BAD: failure points to the wrong line
func createTestURL(t *testing.T, svc Service, url string) *URL {
    // No t.Helper() call
    u, err := svc.Create(ctx, CreateRequest{URL: url})
    if err != nil {
        t.Fatalf("createTestURL: %v", err)
        // ↑ Error: "createTestURL: ..." points to THIS line
        //   You have to trace back to find WHICH test called createTestURL
    }
    return u
}

// ✅ GOOD: failure points to the calling test
func createTestURL(t *testing.T, svc Service, url string) *URL {
    t.Helper()  // ← first line of every helper
    u, err := svc.Create(ctx, CreateRequest{URL: url})
    if err != nil {
        t.Fatalf("createTestURL: %v", err)
        // ↑ Now points to line in the TEST that called createTestURL
    }
    return u
}
```

---

## Part 13: CI & Infrastructure

### 13.1 The Ideal CI Test Pipeline

```yaml
# .github/workflows/ci.yml

jobs:
  lint:        # fast, runs first, fails early
  test-unit:   # go test -race ./..., no external deps
  test-redis:  # go test -race -tags redis ./..., needs Redis service
  build:       # go build ./..., runs after tests pass
  security:    # gitleaks, runs in parallel
  fuzz:        # nightly only, not per-PR
  benchmark:   # per-PR, with regression detection
```

**Current GoShort CI:**
```yaml
lint:
  golangci-lint run

test:
  go test -race -tags redis -coverprofile=coverage.out ./...
  codecov upload

build:
  go build -o goshort ./cmd/server

security:
  gitleaks-action
```

**Recommended additions:**
```yaml
fuzz:
  if: schedule  # nightly
  go test -fuzz=FuzzValidateURL -fuzztime=60s ./internal/shortener/
  go test -fuzz=FuzzValidateAlias -fuzztime=60s ./internal/shortener/
  go test -fuzz=FuzzSqidsEncoder_Encode -fuzztime=60s ./internal/encoder/

benchmark:
  if: pull_request
  go test -bench=. -benchmem ./... | tee bench.txt
  benchstat compare
```

### 13.2 Codecov Best Practices

```yaml
# .codecov.yml — the configuration file matters

coverage:
  status:
    project:
      default:
        target: 82%       # what % we expect overall
        threshold: 3%     # how much it can drop before failing
    patch:
      default:
        target: 75%       # new code in a PR must be 75% covered

ignore:
  - "internal/db/**"        # generated — never test
  - "cmd/server/main.go"    # composition root
  - "cmd/cli/main.go"       # process entrypoint
  - "cmd/cli/version.go"    # trivial, no logic

comment:
  layout: "reach,diff,flags,files"
  require_changes: true     # only comment when coverage changes
```

**Coverage is a proxy metric, not a goal:**
- 95% coverage with useless tests = low quality
- 80% coverage with well-targeted tests = high quality
- Never write tests just to hit a coverage number

### 13.3 Lefthook — Pre-commit Hooks

```yaml
# lefthook.yml

pre-commit:
  parallel: true
  commands:
    lint:
      glob: "*.go"
      run: golangci-lint run --fix
      stage_fixed: true
    test-changed:
      glob: "*.go"
      # Only test packages that changed — much faster than full suite
      run: |
        changed=$(git diff --cached --name-only | grep '\.go$' \
          | xargs -I{} dirname {} | sort -u \
          | sed 's|^|github.com/anIcedAntFA/goshort/|')
        [ -n "$changed" ] && go test -race -count=1 -timeout=30s $changed
    secrets:
      run: gitleaks git --no-banner --staged

pre-push:
  commands:
    full-test:
      run: go test -race -count=1 -timeout=60s ./...
    vet:
      run: go vet ./...
```

**Why the split:**
- Pre-commit: tests only changed packages (3-8s). Gives feedback as you work.
- Pre-push: full suite (15-30s). Final safety net before sharing.

### 13.4 Test Makefile Targets

```makefile
# Testing targets
.PHONY: test test/race test/cover test/redis test/all test/bench

test: ## Run all tests (unit only)
	go test ./...

test/race: ## Run with race detector
	go test -race -count=1 ./...

test/cover: ## HTML coverage report
	go test -coverprofile=coverage.out -coverpkg=./... ./...
	go tool cover -html=coverage.out

test/redis: ## Include Redis integration tests
	go test -race -tags redis -count=1 ./...

test/all: ## Auto-detect Redis and run accordingly
	@if docker compose -f docker-compose.dev.yml ps --status running 2>/dev/null | grep -q redis; then \
		echo "Redis detected — running with -tags redis"; \
		go test -race -tags redis -count=1 ./...; \
	else \
		echo "Redis not running — unit tests only"; \
		go test -race -count=1 ./...; \
	fi

test/bench: ## Run benchmarks
	go test -bench=. -benchmem -benchtime=3s ./...

test/fuzz/url: ## Fuzz URL validator for 30s
	go test -fuzz=FuzzValidateURL -fuzztime=30s ./internal/shortener/

test/fuzz/alias: ## Fuzz alias validator for 30s
	go test -fuzz=FuzzValidateAlias -fuzztime=30s ./internal/shortener/

test/fuzz/encoder: ## Fuzz encoder for 30s
	go test -fuzz=FuzzSqidsEncoder_Encode -fuzztime=30s ./internal/encoder/
```

---

## Appendix A: Test Naming Convention

```
Format: TestType_Method_Scenario

Examples:
  TestValidateURL_EmptyString
  TestValidateURL_PrivateIP_127_0_0_1
  TestService_Create_CustomAlias
  TestService_Create_DuplicateAliasReturnsConflict
  TestHandler_Redirect_ExpiredReturns410
  TestHandler_Redirect_CacheHit
  TestSQLiteStorage_CreateURL_DuplicateShortCode
  TestRedisCache_TTLExpiry
  TestAPIClient_CreateURL_ErrorResponse

For subtests (table cases):
  TestValidateURL/empty_string
  TestValidateURL/private_ip_127_0_0_1
  TestService_Create/custom_alias_succeeds
  TestService_Create/duplicate_alias_returns_conflict
```

**Rules:**
1. Start with `Test` (required by Go)
2. Include the receiver type (not just the method)
3. Include the scenario — what makes this test unique
4. Use `_` to separate parts (Go replaces spaces with `_` in `-run` flag)
5. Be specific: `DuplicateAliasReturnsConflict` beats `Error` or `Fail`

---

## Appendix B: Cheat Sheet

### Quick Reference: Which approach for what?

```
Testing a pure function with no dependencies?
  → Unit test, no mock needed
  → table-driven if multiple input cases
  → fuzz if user-facing or security-sensitive

Testing business logic with external dependencies?
  → Unit test with mock injection
  → London School: mock all collaborators
  → Per-method error injection for error paths
  → insertDirect for impossible-to-reach states

Testing storage/database behavior?
  → Integration test with real :memory: DB
  → Test SQL queries, constraints, ordering
  → White-box for type conversion helpers

Testing HTTP handlers?
  → Full-stack with httptest.ResponseRecorder
  → Wire real service + real SQLite + NoopCache
  → Test: status codes, JSON shape, headers, error codes

Testing an HTTP client?
  → httptest.NewServer as fake API
  → Assert outgoing requests + handle error responses

Testing a CLI command?
  → httptest.NewServer + direct function call
  → os.Pipe() for stdin/stdout testing
  → No t.Parallel() if mutating global state

Testing concurrent code?
  → go test -race (always)
  → goroutine leak detection with goleak
  → time.Sleep with 3× margin OR testing/synctest

Testing protocol behavior without network?
  → In-process transport (sdkmcp.NewInMemoryTransports)
  → Or httptest.Server on loopback
```

### Quick Reference: t.* methods

```
t.Error / t.Errorf    → log failure, continue
t.Fatal / t.Fatalf    → log failure, stop test
t.Log   / t.Logf      → log info (visible with -v or on fail)
t.Skip  / t.Skipf     → skip test (not failed)
t.Parallel()          → run concurrently
t.Cleanup(fn)         → register teardown (preferred over defer)
t.Helper()            → mark as helper (failure points to caller)
t.TempDir()           → temp dir, auto-cleaned
t.Setenv(k, v)        → set env, auto-restored (not parallel-safe!)
t.Run(name, fn)       → subtest
```

### Quick Reference: Build Tags

```bash
# Normal run — no tags
go test ./...

# Redis tests
go test -tags redis ./...

# Integration tests
go test -tags integration ./...

# Skip race detector (emergency only)
go test -tags=!race ./...

# Combined
go test -race -tags redis -count=1 -timeout=60s ./...
```

---

## Appendix C: Glossary

| Term | Definition |
|------|------------|
| **Arrange-Act-Assert** | The three-phase structure of every test: setup, execute, verify |
| **Black-box test** | Tests only public API (`package foo_test`) |
| **Build tag** | `//go:build redis` — conditionally compile test file |
| **Compile-time interface check** | `var _ I = (*T)(nil)` — verifies T implements I at build time |
| **Fake** | Working implementation unsuitable for production (e.g., in-memory DB) |
| **Fuzz test** | Generates random inputs to find crashes/invariant violations |
| **goleak** | Library that detects goroutine leaks after tests complete |
| **London School** | Mock all dependencies, test in strict isolation |
| **Chicago School** | Use real (or realistic fake) dependencies where practical |
| **Mock** | Test double implementing an interface with controllable behavior |
| **Race detector** | Go's runtime tool (`-race`) that detects data races |
| **Stub** | Test double that returns hardcoded responses |
| **T.Helper** | Marks function as test helper so failures point to caller |
| **T.Parallel** | Marks test as safe to run concurrently with others |
| **Table-driven test** | Single test function with a slice of input/expected pairs |
| **Test double** | Generic term for any replacement of a real dependency in tests |
| **Testing pyramid** | Unit (many) → Integration (some) → E2E (few) |
| **White-box test** | Tests internals via `package foo` (same package) |
| **`:memory:`** | SQLite DSN for in-memory database (no file, auto-cleaned) |

---

```
╭──────────────────────────────────────────────────────────╮
│                                                          │
│   "Write tests. Not too many. Mostly integration."      │
│                        — Guillermo Rauch (adapted)       │
│                                                          │
│   (But actually: write a LOT of unit tests,             │
│    enough integration tests, and few E2E tests.         │
│    The pyramid matters.)                                 │
│                                                          │
╰──────────────────────────────────────────────────────────╯
```

---

## Part 14: TDD — Test-Driven Development

### 14.1 What TDD Actually Is (vs What People Think It Is)

```
╭────────────────────────────────────────────────────────────────╮
│  What people think TDD means:                                  │
│    "Write tests before writing code."                          │
│                                                                │
│  What TDD actually means:                                      │
│    "Let tests DRIVE the design of your code."                  │
│                                                                │
│  The difference is enormous.                                   │
╰────────────────────────────────────────────────────────────────╯
```

TDD is not a testing technique. It's a **design methodology** that uses tests as the primary tool for thinking about software. The tests are a side effect. Better design is the goal.

> 💡 **ELI5:** Imagine building with Lego. TDD approach: first describe "I want to build a house with a door that opens," then pick the pieces that make that possible. Non-TDD: pile up all the Lego first, then figure out if it looks like a house.

**Three properties of code written with TDD:**

```
1. TESTABLE     → You were forced to think about seams before writing
2. MINIMAL      → You only wrote what was needed to pass the test
3. DOCUMENTED   → Each test is a specification of intended behavior
```

### 14.2 The Red-Green-Refactor Loop

```
┌──────────────────────────────────────────────────────────────┐
│                                                              │
│  🔴 RED     Write a failing test                             │
│             (test must FAIL — if it passes, you             │
│              tested nothing or it already existed)           │
│                    │                                         │
│                    ▼                                         │
│  🟢 GREEN   Write MINIMAL code to make it pass               │
│             (ugly is fine — just make it green)              │
│             (no speculative features, no "while I'm here")   │
│                    │                                         │
│                    ▼                                         │
│  🔵 REFACTOR Clean up the code                               │
│             (tests stay green throughout)                    │
│             (only refactor when GREEN)                       │
│                    │                                         │
│                    ▼                                         │
│             Repeat for next behavior                         │
│                                                              │
└──────────────────────────────────────────────────────────────┘
```

**Each phase has a specific discipline:**

```
RED phase rules:
  → Test must FAIL first — compile error counts as red
  → Test name describes behavior, not implementation
  → Test uses public API only
  → One behavior per test — not three scenarios in one

GREEN phase rules:
  → Write the MINIMUM code to pass
  → Duplication is acceptable — you'll remove it in refactor
  → Don't add features the test doesn't require
  → If you anticipate a future test, you're doing it wrong

REFACTOR phase rules:
  → Never refactor while RED — get to green first
  → Tests must remain green after every change
  → Look for: duplication, long methods, poor names
  → Small steps — don't refactor + add feature simultaneously
```

### 14.3 🚫 The Horizontal Slicing Anti-Pattern

This is the **most common TDD mistake**. It produces tests that look like TDD but aren't:

```
❌ WRONG — Horizontal Slicing:

RED phase:
  Write test1, test2, test3, test4, test5 ... (ALL tests up front)

GREEN phase:
  Write impl1, impl2, impl3, impl4, impl5 ... (ALL implementations)

Why this is bad:
  - Tests written in bulk test IMAGINED behavior, not actual behavior
  - You outrun your headlights — committing to test structure
    before understanding the implementation
  - Tests end up testing the SHAPE of things (data structures,
    function signatures) rather than user-facing behavior
  - Tests become insensitive to real changes:
    pass when behavior breaks, fail when behavior is fine
```

```
✅ CORRECT — Vertical Slices (Tracer Bullet):

Cycle 1: test1 → impl1 (RED → GREEN → REFACTOR)
Cycle 2: test2 → impl2 (RED → GREEN → REFACTOR)
Cycle 3: test3 → impl3 (RED → GREEN → REFACTOR)
...

Why this works:
  - Each test responds to what you learned from previous cycle
  - You know exactly what behavior matters because you just wrote it
  - Tests are grounded in real implementation, not imagination
  - Small, focused cycles — each one adds one working behavior
```

> 💡 **ELI5:** Horizontal slicing is like a chef who writes ALL the recipes for a restaurant before cooking anything. By the time they actually cook, they discover the recipes don't work together and half the ingredients aren't available. Vertical slicing: write one recipe → cook it → adjust → write the next.

### 14.4 The Tracer Bullet Pattern

A **tracer bullet** is your first test: the simplest possible thing that confirms the path works end-to-end.

```
Purpose of the tracer bullet:
  → Prove the basic plumbing works (imports, wiring, compilation)
  → Establish the shape of the public API
  → Give you a GREEN baseline to build on
  → Identify integration surprises early

What makes a good tracer bullet:
  → Tests one thing — the "happy path" baseline
  → Uses the same interface callers will use
  → Is the simplest possible passing case
  → Not the edge case — that comes later
```

**Example — TDD'ing `ValidateURL` in GoShort:**

```
Step 1: Tracer Bullet
  Test: "https://example.com" → no error

  Write minimal implementation:
    func ValidateURL(url string) error {
        return nil  // just return nil — enough to pass
    }

Step 2: First real constraint
  Test: "" → error

  Update implementation:
    func ValidateURL(url string) error {
        if url == "" {
            return ErrInvalidURL
        }
        return nil
    }

Step 3: Scheme validation
  Test: "ftp://example.com" → error

  Update implementation:
    func ValidateURL(url string) error {
        if url == "" {
            return ErrInvalidURL
        }
        u, _ := url.Parse(url)
        if u.Scheme != "http" && u.Scheme != "https" {
            return ErrInvalidURL
        }
        return nil
    }

Step 4: Private IP
  Test: "http://192.168.1.1" → error
  ... and so on

Notice: each step builds on the last, driven by ONE failing test.
The implementation shape emerges from the tests, not the other way around.
```

### 14.5 Planning Before Writing Tests

TDD doesn't mean "write tests blindly." Before the first test:

```
Planning checklist (do this BEFORE any code):

□ What is the public interface? (function signature, return types)
□ What behaviors matter most? (prioritize: correctness first, edge cases later)
□ What are the dependencies? (identify seams for isolation)
□ What does "done" look like? (list the behaviors you'll test)
□ Are there opportunities for deep modules? (small interface, rich behavior)

Do NOT plan:
  → Every test case up front (you'll discover them as you go)
  → Internal implementation structure (let tests drive this)
  → Error messages exactly (verify behavior, not strings)
```

**🎯 In GoShort — How the service was TDD'd:**

The `shortener.Service` interface was designed before any implementation:
```go
// First: define the contract
type Service interface {
    Create(ctx context.Context, req CreateRequest) (*URL, error)
    GetByCode(ctx context.Context, code string) (*URL, error)
    Delete(ctx context.Context, code string) error
    List(ctx context.Context, opts ListOptions) ([]URL, int, error)
    IncrementClicks(ctx context.Context, code string) error
}
```

The mock storage was written to enable testing without SQLite. The tests drove the service API — not the other way around.

### 14.6 TDD in Practice: Full GoShort Example

```go
// TDD cycle for shortener.Service.Create

// ─────────────── Cycle 1: Tracer Bullet ───────────────
// RED: write the failing test first
func TestService_Create_GeneratesShortCode(t *testing.T) {
    t.Parallel()
    svc, _ := newTestService(t)

    url, err := svc.Create(context.Background(), shortener.CreateRequest{
        URL: "https://example.com",
    })
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
    if url.ShortCode == "" {
        t.Error("expected non-empty ShortCode")
    }
}

// GREEN: minimal implementation
// (just return a hardcoded code — enough to pass)
func (s *ServiceImpl) Create(ctx context.Context, req CreateRequest) (*URL, error) {
    return &URL{ShortCode: "abc123"}, nil
}

// ─────────────── Cycle 2: URL Validation ───────────────
// RED: test that invalid URLs are rejected
func TestService_Create_InvalidURL(t *testing.T) {
    t.Parallel()
    svc, _ := newTestService(t)

    _, err := svc.Create(context.Background(), shortener.CreateRequest{
        URL: "not-a-url",
    })
    if !errors.Is(err, shortener.ErrInvalidURL) {
        t.Errorf("got %v, want ErrInvalidURL", err)
    }
}

// GREEN: add URL validation
func (s *ServiceImpl) Create(ctx context.Context, req CreateRequest) (*URL, error) {
    if err := ValidateURL(req.URL); err != nil {
        return nil, err
    }
    return &URL{ShortCode: "abc123"}, nil
}

// ─────────────── Cycle 3: Real Code Generation ───────────────
// RED: test that different URLs get different codes
func TestService_Create_DifferentURLsDifferentCodes(t *testing.T) {
    t.Parallel()
    svc, _ := newTestService(t)
    ctx := context.Background()

    url1, _ := svc.Create(ctx, shortener.CreateRequest{URL: "https://a.com"})
    url2, _ := svc.Create(ctx, shortener.CreateRequest{URL: "https://b.com"})

    if url1.ShortCode == url2.ShortCode {
        t.Error("different URLs should produce different short codes")
    }
}

// GREEN: use real counter + encoder
func (s *ServiceImpl) Create(ctx context.Context, req CreateRequest) (*URL, error) {
    if err := ValidateURL(req.URL); err != nil {
        return nil, err
    }
    counter, err := s.store.IncrementCounter(ctx)
    if err != nil {
        return nil, fmt.Errorf("create url: increment counter: %w", err)
    }
    code, err := s.encoder.Encode(counter)
    if err != nil {
        return nil, fmt.Errorf("create url: encode: %w", err)
    }
    // ...persist and return
}

// REFACTOR: extract resolveCode, improve error messages, etc.
// Tests stay green throughout refactoring.
```

### 14.7 When TDD Fits vs When It Doesn't

```
TDD WORKS WELL FOR:
  ✓ Business logic with clear inputs/outputs
  ✓ Data validation and transformation
  ✓ Error handling and edge cases
  ✓ Algorithms with defined correctness criteria
  ✓ API design (TDD forces you to think about the caller)

TDD IS HARDER FOR:
  ~ Exploratory code (you don't know the shape yet)
    → Spike first (throw-away code), then TDD the real implementation
  ~ UI/frontend (feedback loop is visual, not programmatic)
    → Use integration/E2E tests instead of strict TDD
  ~ Performance optimization (tests don't describe perf)
    → Write correctness tests first, then benchmark separately
  ~ Infrastructure/wiring code (main.go, DI, config)
    → Integration tests work better than unit TDD here

TDD DOESN'T MEAN:
  ✗ 100% test coverage (some code doesn't need tests)
  ✗ No refactoring after tests (refactoring is built in)
  ✗ Tests never change (behavior changes → tests change)
  ✗ Mocking everything (sometimes real deps are better)
```

### 14.8 Recognizing Good TDD Tests (vs Test-Shaped Objects)

```go
// ❌ TEST-SHAPED OBJECT — looks like a test, isn't
// Tests implementation, not behavior
// Will break on any internal refactor
func TestService_Create_CallsIncrementCounterOnce(t *testing.T) {
    store := &mockStorage{}
    svc := shortener.NewService(store, enc)
    svc.Create(ctx, req)
    if store.incrementCounterCalls != 1 {
        t.Error("expected IncrementCounter called once")
    }
}

// ✅ REAL TDD TEST — tests behavior
// Will survive complete internal reimplementation
func TestService_Create_ShortCodeIsAlphanumeric(t *testing.T) {
    svc, _ := newTestService(t)
    url, err := svc.Create(ctx, shortener.CreateRequest{URL: "https://example.com"})
    if err != nil {
        t.Fatalf("Create: %v", err)
    }
    if !alphanumericRe.MatchString(url.ShortCode) {
        t.Errorf("ShortCode %q contains non-alphanumeric chars", url.ShortCode)
    }
}
```

**The test-shaped object test:**
- Breaks if you change the internal counter strategy
- Passes even if the short code is wrong
- Tests HOW, not WHAT

**The real TDD test:**
- Survives any reimplementation that produces alphanumeric codes
- Fails if the output is wrong
- Tests the observable contract

---

## Part 15: Property-Based vs Example-Based Testing

### 15.1 The Two Fundamental Testing Philosophies

```
╭──────────────────────────────────────────────────────────╮
│  EXAMPLE-BASED TESTING                                   │
│  "I know exactly what input produces what output."       │
│                                                          │
│  ValidateURL("https://example.com") → nil                │
│  ValidateURL("")                    → ErrInvalidURL      │
│  ValidateURL("ftp://bad.com")       → ErrInvalidURL      │
│                                                          │
│  ✓ Readable, precise, documents specific cases           │
│  ✗ Only as good as the examples you can think of         │
╰──────────────────────────────────────────────────────────╯

╭──────────────────────────────────────────────────────────╮
│  PROPERTY-BASED TESTING                                  │
│  "No matter the input, these invariants always hold."    │
│                                                          │
│  ∀ valid URL → Encode(Decode(code)) == code             │
│  ∀ input    → ValidateURL never panics                   │
│  ∀ n ≥ 0   → len(Encode(n)) >= minLength               │
│                                                          │
│  ✓ Finds edge cases you'd never think to write           │
│  ✗ Harder to write, output is less readable when fails   │
╰──────────────────────────────────────────────────────────╯
```

> 💡 **ELI5:** Example-based = "I tested the car by driving it to the supermarket and back." Property-based = "I tested the car by driving it in every possible direction, weather, and speed — anything the laws of physics allow."

### 15.2 Example-Based Testing — Deep Dive

This is what 95% of Go tests are. You've seen it throughout this file. The key insight is choosing *which* examples to write:

**The 5 categories of examples that matter:**

```
1. HAPPY PATH
   The canonical correct usage.
   ValidateURL("https://example.com") → nil

2. BOUNDARY VALUES
   At the exact edge of valid/invalid.
   ValidateAlias("abc")                    → nil   (min 3 chars)
   ValidateAlias("ab")                     → error (min-1)
   ValidateAlias(strings.Repeat("a", 30))  → nil   (max 30 chars)
   ValidateAlias(strings.Repeat("a", 31))  → error (max+1)

3. ERROR CASES
   Every distinct error the function can return.
   ValidateURL("") → ErrInvalidURL (empty)
   ValidateURL("ftp://x") → ErrInvalidURL (bad scheme)
   ValidateURL("http://127.0.0.1") → ErrInvalidURL (private IP)

4. ZERO / NIL / EMPTY VALUES
   What happens with the zero value of each type.
   Service.List(ctx, ListOptions{}) → defaults applied, not panic

5. INTEGRATION SCENARIOS
   End-to-end flows that combine multiple behaviors.
   Create → Redirect → VerifyClickCount
```

**🎯 In GoShort — example quality:**

`TestValidateURL` has 25 carefully chosen examples covering all 5 categories. This is the standard to aim for.

### 15.3 Property-Based Testing in Go

Go's built-in `testing/fuzz` is a form of property-based testing. But for richer property testing with shrinking, there are dedicated libraries.

**Go's built-in fuzzing (already covered in Part 8):**
```go
// Property: ValidateURL never panics on any string
f.Fuzz(func(t *testing.T, input string) {
    _ = shortener.ValidateURL(input)
})
```

**Property-based testing with `pgregory.net/rapid`:**

`rapid` is the most ergonomic property-based testing library for Go. It generates structured random inputs, and when a test fails, it *shrinks* the input to the minimal failing case.

```bash
go get pgregory.net/rapid
```

```go
import "pgregory.net/rapid"

// Property: Encode then Decode is a roundtrip
func TestSqidsEncoder_Roundtrip_Property(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        // rapid generates arbitrary int64 values
        id := rapid.Int64Range(0, math.MaxInt64).Draw(t, "id")

        enc, _ := encoder.NewSqidsEncoder(6)
        code, err := enc.Encode(id)
        if err != nil {
            t.Skip() // skip values that legitimately error
        }

        got, err := enc.Decode(code)
        if err != nil {
            t.Fatalf("Decode(Encode(%d)) failed: %v", id, err)
        }
        if got != id {
            t.Fatalf("roundtrip: Encode(%d) → %q → Decode → %d", id, code, got)
        }
    })
}
```

**Shrinking — the killer feature:**

When `rapid` finds a failing input, it automatically shrinks it to the simplest possible failing case:

```
Initial failure: id = 9223372036854775807 (math.MaxInt64)
After shrinking: id = 42

→ The minimal failing case is always shown first
→ Makes debugging dramatically easier
→ Go's built-in fuzzer does NOT automatically shrink
```

**More `rapid` generators:**

```go
// Primitive generators
rapid.Int()                     // any int
rapid.Int64Range(0, 100)        // int64 in range
rapid.StringMatching(`^[a-z]+$`)// string matching regex
rapid.SliceOf(rapid.Int())      // []int
rapid.MapOf(rapid.String(), rapid.Int()) // map[string]int

// Custom generators
validURL := rapid.Custom(func(t *rapid.T) string {
    scheme := rapid.SampledFrom([]string{"http", "https"}).Draw(t, "scheme")
    host := rapid.StringMatching(`^[a-z0-9]+\.[a-z]{2,4}$`).Draw(t, "host")
    return scheme + "://" + host
})

// Usage
func TestValidateURL_AlwaysAcceptsValidURLs(t *testing.T) {
    rapid.Check(t, func(t *rapid.T) {
        url := validURL.Draw(t, "url")
        if err := shortener.ValidateURL(url); err != nil {
            t.Fatalf("valid URL %q rejected: %v", url, err)
        }
    })
}
```

### 15.4 When to Use Each Approach

```
┌──────────────────────┬────────────────────────────────────────────┐
│  Scenario            │  Approach                                  │
├──────────────────────┼────────────────────────────────────────────┤
│  Known correct       │  Example-based — you know exactly what's   │
│  input/output pairs  │  correct                                   │
│                      │                                            │
│  Security-sensitive  │  Fuzz test — find unexpected inputs that   │
│  parsing/validation  │  panic or bypass validation                │
│                      │                                            │
│  Algebraic           │  Property-based (rapid) — roundtrip,       │
│  invariants          │  idempotency, commutativity                │
│  (encode/decode,     │                                            │
│   compress/decompress│                                            │
│   serialize/parse)   │                                            │
│                      │                                            │
│  Complex domain      │  Both: examples for known cases + property │
│  logic               │  for invariants                            │
│                      │                                            │
│  Performance-critical│  Benchmark — not correctness testing       │
│  hot path            │                                            │
│                      │                                            │
│  UI / visual output  │  Snapshot testing (see 15.5)               │
└──────────────────────┴────────────────────────────────────────────┘
```

**Decision tree:**

```
Does the function have clear, known correct outputs?
  YES → Example-based (table-driven)

Does the function have inputs from untrusted sources?
  YES → Add fuzz test on top of examples

Does the function implement an algebraic law?
  (roundtrip, idempotent, monotone, etc.)
  YES → Property-based with rapid

Is the output too complex to assert field-by-field?
  (JSON blobs, rendered HTML, SQL queries)
  YES → Snapshot testing
```

### 15.5 Snapshot Testing

Snapshot testing captures the output of a function and stores it as a "golden file." Future test runs compare against the stored snapshot.

**When to use:** Testing complex output that would require 50+ assertions to verify fully — JSON API responses, CLI output, generated SQL, rendered templates.

```bash
go get github.com/bradleyjkemp/cupaloy/v2
```

```go
import "github.com/bradleyjkemp/cupaloy/v2"

func TestCreateURLResponse_Snapshot(t *testing.T) {
    t.Parallel()
    router := setupTestDeps(t)

    body := mustMarshal(t, map[string]string{"url": "https://example.com"})
    rec := serve(t, router, "POST", "/api/v1/urls", body)

    // Normalize non-deterministic fields before snapshot
    var resp map[string]any
    json.NewDecoder(rec.Body).Decode(&resp)
    resp["short_code"] = "<generated>"
    resp["created_at"] = "<timestamp>"

    // First run: creates testdata/.snapshots/TestCreateURLResponse_Snapshot
    // Subsequent runs: compares against stored snapshot
    cupaloy.SnapshotT(t, resp)
}
```

**Workflow:**

```bash
# First run — creates the snapshot
go test ./...
# → creates testdata/.snapshots/TestCreateURLResponse_Snapshot

# Update snapshots when behavior intentionally changes
UPDATE_SNAPSHOTS=true go test ./...

# Commit snapshots to version control
git add testdata/.snapshots/
git commit -m "update API response snapshots"
```

**⚠️ Snapshot pitfalls:**
- Non-deterministic output (timestamps, UUIDs, random data) must be normalized before snapshot
- Snapshots can hide bugs if they capture incorrect behavior
- Commit snapshots to git — they are part of the spec

### 15.6 Property Categories — Taxonomy

When writing property-based tests, use these patterns to identify what property to test:

```go
// 1. ROUNDTRIP (most common)
//    encode(decode(x)) == x
//    compress(decompress(x)) == x
Decode(Encode(n)) == n

// 2. IDEMPOTENT
//    f(f(x)) == f(x)
//    normalize(normalize(url)) == normalize(url)
Normalize(Normalize(url)) == Normalize(url)

// 3. ORACLE — two implementations agree
//    slowButCorrect(x) == fastOptimized(x)
SlowValidate(url) == FastValidate(url)

// 4. INVARIANT — property always holds regardless of input
//    len(Encode(n)) >= minLength for all valid n
len(Encode(n)) >= 6

// 5. METAMORPHIC — changing input in predictable way changes output predictably
//    if URL1 is longer than URL2, that doesn't affect short code uniqueness
Encode(n) != Encode(n+1)  // different inputs → different codes

// 6. MONOTONIC — ordering preserved
//    larger counters don't produce smaller codes (not required for Sqids)

// 7. COMMUTATIVITY / ASSOCIATIVITY
//    f(a, b) == f(b, a)
//    (not usually relevant for URL shorteners)
```

---

## Part 16: stretchr/testify — The Ecosystem Standard

### 16.1 Why testify Exists

Go's standard library `testing` package is deliberately minimal. It gives you `t.Error`, `t.Fatal`, and `t.Log` — nothing more. This is a conscious design choice: no assertions, no matchers, no magic.

The result: most projects end up writing the same helpers repeatedly:
```go
// Everyone writes this over and over
if got != want {
    t.Errorf("got %v, want %v", got, want)
}
```

`testify` codifies these patterns into a reusable library. It's used in the majority of production Go codebases and is effectively the community standard for assertions.

```
╭──────────────────────────────────────────────────────────╮
│  GoShort uses stdlib testing only.                       │
│  This section covers testify for the broader Go world.   │
│  You'll encounter it in every serious Go codebase.       │
╰──────────────────────────────────────────────────────────╯
```

### 16.2 Installation

```bash
go get github.com/stretchr/testify
```

```go
import (
    "github.com/stretchr/testify/assert"   // continue after failure
    "github.com/stretchr/testify/require"  // stop on failure
    "github.com/stretchr/testify/mock"     // mocking
    "github.com/stretchr/testify/suite"    // test suites
)
```

### 16.3 `assert` vs `require` — The Critical Distinction

This is the most important thing to understand about testify:

```
assert  →  records failure, TEST CONTINUES
require →  records failure, TEST STOPS IMMEDIATELY (calls t.FailNow)
```

```go
func TestParseConfig(t *testing.T) {
    is   := assert.New(t)   // convention: name it "is"
    must := require.New(t)  // convention: name it "must"

    cfg, err := ParseConfig("testdata/valid.yaml")
    must.NoError(err)   // ← STOP if error — cfg would be nil below
    must.NotNil(cfg)    // ← STOP if nil — accessing fields would panic

    // These continue even if one fails — collect all failures at once
    is.Equal("production", cfg.Environment)
    is.Equal(8080, cfg.Port)
    is.True(cfg.TLS.Enabled)
    is.Len(cfg.Hosts, 3)
}
```

**The rule:**
```
Use require (must) for:
  - Error checks before using the result
  - Nil checks before dereferencing
  - Preconditions where failure makes subsequent assertions meaningless

Use assert (is) for:
  - Verifications where all failures are independently valuable
  - When you want to see ALL failures in one test run
```

**Failing to follow this rule causes panics:**
```go
// ❌ PANIC if err != nil (cfg is nil, cfg.Port panics)
cfg, err := ParseConfig("bad.yaml")
assert.NoError(t, err)    // records failure, continues
assert.Equal(t, 8080, cfg.Port)  // PANIC: nil pointer dereference

// ✅ STOPS before the panic
cfg, err := ParseConfig("bad.yaml")
require.NoError(t, err)   // stops test here if error
assert.Equal(t, 8080, cfg.Port)  // only reached if cfg is valid
```

### 16.4 Core Assertions

```go
is := assert.New(t)  // fluent API — omit "t" from each call
// or
assert.Equal(t, expected, actual)  // direct API — pass "t" each time

// ─────── EQUALITY ───────
is.Equal(expected, actual)           // DeepEqual, exact type match
is.NotEqual(unexpected, actual)
is.EqualValues(42, int64(42))        // converts to common type first
is.Same(ptr1, ptr2)                  // same pointer identity

// ─────── NIL / BOOL ───────
is.Nil(val)          is.NotNil(val)
is.True(cond)        is.False(cond)
is.Zero(val)         is.NotZero(val)

// ─────── ERRORS ───────
is.Error(err)                        // err != nil
is.NoError(err)                      // err == nil
is.ErrorIs(err, ErrNotFound)         // errors.Is — walks chain
is.ErrorAs(err, &target)             // errors.As — walks chain
is.ErrorContains(err, "not found")   // err.Error() contains string

// ─────── COLLECTIONS ───────
is.Empty(slice)          is.NotEmpty(slice)
is.Len(slice, 5)
is.Contains([]int{1,2,3}, 2)         // slice contains element
is.Contains("hello world", "world")  // string contains substring
is.Contains(map[string]int{"a":1},"a") // map contains key
is.ElementsMatch([]int{3,1,2}, []int{1,2,3}) // same elements, any order
is.Subset([]int{1,2,3,4}, []int{1,3}) // second is subset of first

// ─────── NUMERIC ───────
is.Greater(actual, 0)
is.GreaterOrEqual(actual, 0)
is.Less(actual, 100)
is.LessOrEqual(actual, 100)
is.InDelta(3.14159, computed, 0.001)  // float tolerance
is.InEpsilon(expected, actual, 0.01) // relative tolerance (1%)
is.Positive(val)
is.Negative(val)

// ─────── STRINGS ───────
is.EqualFold("Hello", "hello")       // case-insensitive
is.HasPrefix(str, "https://")
is.HasSuffix(str, ".com")
is.Regexp(`^[a-z0-9]+$`, str)
is.NotRegexp(`[^a-z]`, str)

// ─────── TIME ───────
is.WithinDuration(expected, actual, 5*time.Second)

// ─────── TYPES ───────
is.IsType(&User{}, obj)
is.Implements((*io.Reader)(nil), obj)

// ─────── JSON ───────
is.JSONEq(`{"name":"alice"}`, `{"name": "alice"}`)  // ignores whitespace/order

// ─────── PANIC ───────
is.Panics(func() { mustNotPanic() })
is.NotPanics(func() { safeFunc() })
is.PanicsWithValue("oops", func() { panic("oops") })
```

**⚠️ Argument order matters: always `(expected, actual)`**

```go
// ✅ CORRECT — expected first
is.Equal(200, rec.Code)         // "expected 200, got 404"

// ❌ WRONG — reversed — confusing diff output
is.Equal(rec.Code, 200)         // "expected 404, got 200" (backwards!)
```

### 16.5 Advanced Assertions

```go
// Eventually — polling with timeout (for async operations)
is.Eventually(func() bool {
    status, _ := client.GetJobStatus(jobID)
    return status == "completed"
}, 5*time.Second, 100*time.Millisecond)
// → polls every 100ms, fails after 5s

// EventuallyWithT — Eventually with assertions inside
is.EventuallyWithT(func(c *assert.CollectT) {
    resp, err := client.GetOrder(orderID)
    assert.NoError(c, err)
    assert.Equal(c, "shipped", resp.Status)
    assert.Greater(c, resp.Version, 0)
}, 10*time.Second, 500*time.Millisecond)
// → on timeout, shows the LAST set of assertion failures

// Never — inverse of Eventually
is.Never(func() bool {
    return errorQueue.HasErrors()
}, 2*time.Second, 100*time.Millisecond)

// Condition — custom predicate
is.Condition(func() bool {
    return complexBusinessLogic(obj)
}, "object does not satisfy business rule X")
```

### 16.6 `testify/mock` — Mocking with Expectations

The key difference from GoShort's hand-written mocks: testify/mock supports **expectation-based verification** — you declare what calls *should* happen, and `AssertExpectations` verifies all of them happened.

```go
// Step 1: Define the interface (same as always)
type URLStore interface {
    GetByCode(ctx context.Context, code string) (*URL, error)
    CreateURL(ctx context.Context, params CreateParams) (URL, error)
}

// Step 2: Generate or write the mock
type MockURLStore struct {
    mock.Mock  // ← embed mock.Mock
}

func (m *MockURLStore) GetByCode(ctx context.Context, code string) (*URL, error) {
    args := m.Called(ctx, code)  // ← registers the call, returns configured values
    if args.Get(0) == nil {
        return nil, args.Error(1)
    }
    return args.Get(0).(*URL), args.Error(1)
}

func (m *MockURLStore) CreateURL(ctx context.Context, params CreateParams) (URL, error) {
    args := m.Called(ctx, params)
    return args.Get(0).(URL), args.Error(1)
}

// Step 3: Use in tests
func TestService_GetByCode(t *testing.T) {
    is := assert.New(t)
    store := new(MockURLStore)

    // Configure: when GetByCode("abc") is called, return this URL
    expectedURL := &URL{ShortCode: "abc", OriginalURL: "https://example.com"}
    store.On("GetByCode", mock.Anything, "abc").Return(expectedURL, nil)

    svc := NewService(store, encoder)
    url, err := svc.GetByCode(context.Background(), "abc")

    is.NoError(err)
    is.Equal(expectedURL.OriginalURL, url.OriginalURL)
    store.AssertExpectations(t)  // ← VERIFY all configured expectations were met
}
```

**Argument matchers:**

```go
// mock.Anything — match any value of any type
store.On("GetByCode", mock.Anything, mock.Anything).Return(url, nil)

// mock.AnythingOfType — match by type name string
store.On("CreateURL", mock.Anything, mock.AnythingOfType("CreateParams")).Return(url, nil)

// mock.MatchedBy — custom predicate
store.On("CreateURL", mock.Anything, mock.MatchedBy(func(p CreateParams) bool {
    return strings.HasPrefix(p.OriginalURL, "https://")
})).Return(url, nil)
```

**Call modifiers:**

```go
// How many times
store.On("GetByCode", mock.Anything, "abc").Return(url, nil).Once()
store.On("GetByCode", mock.Anything, "abc").Return(url, nil).Times(3)
store.On("GetByCode", mock.Anything, "abc").Return(url, nil).Maybe() // optional

// Side effects (e.g., record what was passed)
var capturedCode string
store.On("GetByCode", mock.Anything, mock.Anything).
    Run(func(args mock.Arguments) {
        capturedCode = args.String(1)
    }).
    Return(url, nil)

// Different returns per call (for retry testing)
store.On("GetByCode", mock.Anything, "abc").
    Return(nil, errors.New("timeout")).Once()  // first call fails
store.On("GetByCode", mock.Anything, "abc").
    Return(url, nil).Once()                    // second call succeeds
```

**Verification:**

```go
store.AssertExpectations(t)                           // all expected calls happened
store.AssertCalled(t, "GetByCode", mock.Anything, "abc") // specific call was made
store.AssertNotCalled(t, "CreateURL", mock.Anything)  // specific call was NOT made
store.AssertNumberOfCalls(t, "GetByCode", 2)          // exact call count
```

### 16.7 Hand-Written Mock vs `testify/mock` — Comparison

```
┌──────────────────────────┬────────────────────────────────────────┐
│  Hand-Written Mock       │  testify/mock                          │
│  (GoShort's approach)    │                                        │
├──────────────────────────┼────────────────────────────────────────┤
│  Pure Go, no deps        │  Requires testify dependency           │
│  Full control of logic   │  Framework handles wiring              │
│  Readable at a glance    │  More boilerplate per method           │
│  No magic                │  Some magic in m.Called()              │
│                          │                                        │
│  Per-method error fields │  .Return() values per expectation      │
│  → simple, transparent   │  → more flexible, more complex         │
│                          │                                        │
│  No call verification    │  AssertExpectations verifies calls     │
│  (behavior-focused)      │  (useful for London School tests)      │
│                          │                                        │
│  Best when:              │  Best when:                            │
│  - Small interface       │  - Large interface (many methods)      │
│  - Testing behavior      │  - Need strict call verification       │
│  - Team knows Go well    │  - Already using testify elsewhere     │
│  - Chicago School style  │  - London School strict mocking        │
└──────────────────────────┴────────────────────────────────────────┘
```

**🎯 GoShort's choice of hand-written mocks is valid and deliberate.** The `mockStorage` in `service_test.go` is 200 lines but completely transparent — no framework magic, no indirect control flow.

### 16.8 `testify/suite` — Test Suites

Suites group tests that share setup/teardown. Most useful for integration tests that need a database or external service.

**Lifecycle:**
```
SetupSuite()     → runs ONCE before all tests in the suite
  SetupTest()    → runs before EACH test
    TestFoo()    → each test method
  TearDownTest() → runs after EACH test
TearDownSuite()  → runs ONCE after all tests
```

```go
// Integration test suite with shared database
type URLServiceSuite struct {
    suite.Suite                    // ← embed suite.Suite
    db      *storage.SQLiteStorage
    svc     shortener.Service
}

// Runs once — expensive setup
func (s *URLServiceSuite) SetupSuite() {
    store, err := storage.NewSQLiteStorage(context.Background(), ":memory:")
    s.Require().NoError(err)      // ← s.Require() works like require.New(t)
    s.db = store
}

// Runs before each test — ensure clean state
func (s *URLServiceSuite) SetupTest() {
    enc, _ := encoder.NewSqidsEncoder(4)
    s.svc = shortener.NewService(s.db, enc)
    // If DB had data from previous test, clear it here
}

// Runs once — cleanup
func (s *URLServiceSuite) TearDownSuite() {
    _ = s.db.Close()
}

// Tests — any method starting with "Test"
func (s *URLServiceSuite) TestCreate_GeneratesUniqueCode() {
    ctx := context.Background()

    url1, err := s.svc.Create(ctx, shortener.CreateRequest{URL: "https://a.com"})
    s.Require().NoError(err)

    url2, err := s.svc.Create(ctx, shortener.CreateRequest{URL: "https://b.com"})
    s.Require().NoError(err)

    s.NotEqual(url1.ShortCode, url2.ShortCode)
}

func (s *URLServiceSuite) TestCreate_InvalidURL_ReturnsError() {
    _, err := s.svc.Create(context.Background(), shortener.CreateRequest{
        URL: "not-a-url",
    })
    s.ErrorIs(err, shortener.ErrInvalidURL)
}

// ⚠️ REQUIRED: The launcher function — without this, NO tests run
func TestURLServiceSuite(t *testing.T) {
    suite.Run(t, new(URLServiceSuite))
}
```

**Suite vs plain tests — when to use which:**

```
Use suite when:
  ✓ Multiple tests need shared, expensive setup (DB connection, HTTP server)
  ✓ Integration tests that need consistent state between SetupTest calls
  ✓ You want lifecycle hooks (BeforeTest, AfterTest)

Use plain tests when:
  ✓ Tests are independent and fast
  ✓ Each test can set up its own state cheaply (like newTestStorage)
  ✓ You want maximum parallelism (suites don't parallelize subtests easily)

GoShort uses plain tests — each test creates its own :memory: SQLite
in ~1ms. No need for shared setup.
```

### 16.9 Fixtures — Reusable Test Data

When many tests need similar objects, extract fixtures:

```go
// testdata/fixtures/fixtures.go (or a fixtures package)
package fixtures

import "time"

// Fixed test data — deterministic, no randomness
var (
    ValidURL = shortener.URL{
        ID:          1,
        ShortCode:   "abc123",
        OriginalURL: "https://example.com",
        IsCustom:    false,
        CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
        ClickCount:  0,
    }

    ExpiredURL = shortener.URL{
        ID:          2,
        ShortCode:   "expired",
        OriginalURL: "https://old.example.com",
        IsCustom:    false,
        CreatedAt:   time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
        ExpiresAt: func() *time.Time {
            t := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
            return &t
        }(),
    }

    CustomAliasURL = shortener.URL{
        ID:          3,
        ShortCode:   "my-link",
        OriginalURL: "https://custom.example.com",
        IsCustom:    true,
        CreatedAt:   time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
    }
)

// Builder — for tests that need slight variations
func NewURL(opts ...func(*shortener.URL)) shortener.URL {
    u := ValidURL // start from known good state
    for _, opt := range opts {
        opt(&u)
    }
    return u
}

func WithCustomAlias(alias string) func(*shortener.URL) {
    return func(u *shortener.URL) {
        u.ShortCode = alias
        u.IsCustom = true
    }
}

func WithExpiry(t time.Time) func(*shortener.URL) {
    return func(u *shortener.URL) {
        u.ExpiresAt = &t
    }
}
```

**Usage:**
```go
func TestService_Delete_CustomAlias(t *testing.T) {
    t.Parallel()
    svc, store := newTestService(t)

    // Use fixture with modification
    url := fixtures.NewURL(fixtures.WithCustomAlias("my-link"))
    store.insertDirect(&url)

    err := svc.Delete(context.Background(), "my-link")
    if err != nil {
        t.Fatalf("Delete: %v", err)
    }
}
```

### 16.10 Time Mocking with `clockwork`

For production code (not just tests), inject a clock interface. This is the "proper" alternative to `testing/synctest`:

```bash
go get github.com/jonboulle/clockwork
```

```go
// Production code — accept a clock
type MemoryCache struct {
    m     sync.Map
    clock clockwork.Clock  // ← injectable
}

func NewMemoryCache(clock clockwork.Clock) *MemoryCache {
    return &MemoryCache{clock: clock}
}

// Production usage — real clock
cache := NewMemoryCache(clockwork.NewRealClock())

// Test usage — fake clock
func TestMemoryCache_Expiry(t *testing.T) {
    t.Parallel()
    fakeClock := clockwork.NewFakeClock()
    c := NewMemoryCache(fakeClock)
    ctx := context.Background()

    _ = c.Set(ctx, "key", "value", 100*time.Millisecond)

    // Advance time without sleeping
    fakeClock.Advance(200 * time.Millisecond)

    _, ok := c.Get(ctx, "key")
    if ok {
        t.Error("expected cache entry to be expired")
    }
    // No time.Sleep! Instant, deterministic.
}
```

**`clockwork` vs `testing/synctest`:**

```
clockwork:
  + Stable, production-ready, no compatibility warnings
  + Works with any Go version
  + Explicit control — you decide when time advances
  - Requires modifying production code (inject clock)
  - Extra dependency

testing/synctest (Go 1.24+):
  + No production code changes needed
  + Time advances automatically when goroutines block
  - Experimental API (may change)
  - Only Go 1.24+
  - Requires understanding of goroutine scheduling
```

**🎯 For GoShort:** `testing/synctest` requires injecting clock into `MemoryCache`. `clockwork` also requires the same injection. The choice is `testing/synctest` (no new dependency) once GoShort targets Go 1.24+.

---

## Part 17: Test Tooling — Generators, Linters, Utilities

### 17.1 `gotests` — Table-Driven Test Generation

`gotests` analyzes your Go source and generates table-driven test scaffolding.

```bash
# Install
go install github.com/cweill/gotests/gotests@latest

# Generate tests for all functions in a file
gotests -all internal/shortener/validator.go

# Generate test for a specific function
gotests -run ValidateURL internal/shortener/validator.go

# Generate and write to file
gotests -all -w internal/shortener/validator.go

# Generate with exported-functions-only filter
gotests -exported internal/shortener/service_impl.go
```

**What it generates:**

```go
// Given this function:
func ValidateURL(rawURL string) error { ... }

// gotests generates:
func TestValidateURL(t *testing.T) {
    tests := []struct {
        name    string
        rawURL  string
        wantErr bool
    }{
        // TODO: Add test cases.
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if err := ValidateURL(tt.rawURL); (err != nil) != tt.wantErr {
                t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Your job after gotests:**
1. Fill in the `// TODO: Add test cases.` table
2. Add `t.Parallel()` to both outer and inner test
3. Replace `wantErr bool` with `wantErr error` to use `errors.Is`
4. Add `checkResult func(*testing.T, *ReturnType)` if needed

**🎯 In GoShort:** The table-driven test pattern in `validator_test.go` and `service_test.go` was scaffolded with `gotests` and then enriched manually.

### 17.2 Test Linters — Enforcing Best Practices

Many test best practices can be automated via linters. These run as part of `golangci-lint`:

#### `thelper` — Enforce `t.Helper()`

Detects test helper functions that don't call `t.Helper()`. Without it, failure messages point to the wrong line.

```yaml
# .golangci.yml
linters:
  enable:
    - thelper
```

```go
// ❌ Caught by thelper — missing t.Helper()
func createURL(t *testing.T, svc Service, url string) *URL {
    // no t.Helper() → failure points HERE instead of caller
    u, err := svc.Create(ctx, req)
    if err != nil {
        t.Fatalf("createURL: %v", err)
    }
    return u
}

// ✅ Fixed
func createURL(t *testing.T, svc Service, url string) *URL {
    t.Helper()  // ← thelper requires this
    u, err := svc.Create(ctx, req)
    if err != nil {
        t.Fatalf("createURL: %v", err)
    }
    return u
}
```

#### `paralleltest` — Enforce `t.Parallel()`

Detects tests that don't call `t.Parallel()` and subtests that don't call it inside `t.Run`.

```yaml
linters:
  enable:
    - paralleltest

linters-settings:
  paralleltest:
    # Don't require t.Parallel() for tests that use t.Setenv or t.Chdir
    # (those are intentionally not parallel)
    ignore-missing-subtests: true
```

```go
// ❌ Caught by paralleltest
func TestValidateURL(t *testing.T) {
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            // missing t.Parallel() in subtest
        })
    }
}

// ✅ Fixed
func TestValidateURL(t *testing.T) {
    t.Parallel()
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            // ...
        })
    }
}
```

#### `testifylint` — testify Best Practices

Catches common testify mistakes:

```yaml
linters:
  enable:
    - testifylint
```

```go
// ❌ Wrong argument order (caught by testifylint)
assert.Equal(t, actual, expected)    // should be (expected, actual)

// ❌ Should use ErrorIs (caught by testifylint)
assert.Equal(t, ErrNotFound, err)   // doesn't walk error chain

// ❌ Should use require for preconditions (caught by testifylint)
assert.NoError(t, err)
url.DoSomething()  // if err != nil, this panics

// ❌ Useless nil check (caught by testifylint)
assert.True(t, err == nil)  // use assert.NoError instead
```

#### `govet` — Test-Specific Checks

`go vet` catches several test-related issues:

```bash
go vet ./...
```

Catches:
- Tests with wrong signature (`func TestFoo(t testing.T)` — missing `*`)
- Benchmarks with wrong signature
- `t.Fatal` called from goroutines inside tests (wrong — goroutines can't call Fatal)

#### Full `.golangci.yml` test section:

```yaml
linters:
  enable:
    - thelper        # t.Helper() in test helpers
    - paralleltest   # t.Parallel() everywhere
    - testifylint    # testify best practices
    - govet          # test signature errors
    - errcheck       # unchecked errors (important in tests too)

linters-settings:
  paralleltest:
    ignore-missing-subtests: true  # for Setenv/Chdir tests
  thelper:
    test:
      first: true   # t.Helper() must be first line
      begin: true   # must be at function start
```

### 17.3 Coverage Tools — Beyond the Basics

```bash
# Standard coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
go tool cover -html=coverage.out

# Per-package coverage with threshold
go test -coverprofile=coverage.out -coverpkg=./... ./...
# Note: -coverpkg=./... counts lines in ALL packages,
# not just the one being tested — more accurate but lower %

# Find uncovered lines programmatically
go tool cover -func=coverage.out | grep -v "100.0%"

# Coverage by function — find the worst offenders
go tool cover -func=coverage.out | sort -k3 -n | head -20
```

**Understanding coverage numbers:**

```
Statement coverage (default):
  → Counts executed statements
  → go test -cover

Branch coverage:
  → Counts executed branches (if/else paths)
  → Not natively supported — use mutation testing instead

Function coverage:
  → go tool cover -func shows per-function %
  → 0.0% = function never called in tests at all
  → 100% = all statements in function executed
```

**Coverage lies — examples:**

```go
// This function has 100% statement coverage with this test:
func Max(a, b int) int {
    if a > b {
        return a
    }
    return b
}

func TestMax(t *testing.T) {
    if Max(3, 1) != 3 { t.Error("wrong") }
    if Max(1, 3) != 3 { t.Error("wrong") }
}
// 100% covered! But what about Max(3, 3)?  Max(-1, -2)?
// Coverage doesn't tell you about correctness — only line execution.
```

### 17.4 `go test` Flags — Complete Reference

```bash
# ─── SELECTION ───
go test -run TestName          # filter by test name (regex)
go test -run TestName/sub      # filter by subtest name
go test -run 'TestA|TestB'     # multiple tests (OR)
go test -run 'Test[A-Z]'       # regex character class
go test -bench BenchmarkName   # run benchmarks (regex)
go test -fuzz FuzzName         # run fuzzing

# ─── EXECUTION ───
go test -race                  # race detector (always use in CI)
go test -count=5               # run N times (catches flaky tests)
go test -parallel=4            # max concurrent tests (default GOMAXPROCS)
go test -timeout=30s           # test timeout (default 10m!)
go test -short                 # skip tests that check testing.Short()
go test -shuffle=on            # randomize test order (catches order deps)
go test -shuffle=1234          # specific seed for reproducibility

# ─── OUTPUT ───
go test -v                     # verbose: show each test name + result
go test -json                  # JSON output (for CI tooling)

# ─── COVERAGE ───
go test -cover                 # print coverage summary
go test -coverprofile=out      # write coverage data to file
go test -covermode=atomic      # thread-safe coverage (for -race)
go test -coverpkg=./...        # count coverage across all packages

# ─── BUILD ───
go test -tags redis            # enable build tags
go test -gcflags="-m"          # pass flags to compiler
go test -ldflags="-X main.v=1" # pass flags to linker

# ─── PROFILING ───
go test -cpuprofile=cpu.prof   # CPU profile
go test -memprofile=mem.prof   # memory profile
go test -trace=trace.out       # execution trace

# ─── COMBINED EXAMPLES ───
# The gold standard for CI
go test -race -count=1 -timeout=60s -coverprofile=coverage.out ./...

# Find flaky tests (run 20 times, look for failures)
go test -race -count=20 -timeout=5m ./...

# Run only fast tests
go test -short -timeout=10s ./...

# Randomize order to find ordering dependencies
go test -shuffle=on -count=3 ./...

# Benchmark with comparison data
go test -bench=. -benchmem -count=10 ./... | tee bench.txt
```

### 17.5 `go test -shuffle` — Finding Order Dependencies

One of Go's most underused flags:

```bash
go test -shuffle=on ./...
```

Tests that pass in the default order can fail when shuffled if they share state:

```go
// ❌ Order-dependent: TestB depends on state left by TestA
var globalCounter int  // shared state!

func TestA(t *testing.T) {
    globalCounter++       // sets globalCounter = 1
}

func TestB(t *testing.T) {
    if globalCounter != 1 { // only passes after TestA
        t.Error("counter wrong")
    }
}

// With -shuffle=on, TestB might run before TestA → FAIL
// → reveals the hidden dependency
```

**Run with shuffle in CI to prevent ordering bugs from hiding:**
```yaml
# Add to CI
- run: go test -shuffle=on -count=3 ./...
```

### 17.6 `benchstat` — Statistical Benchmark Comparison

```bash
go install golang.org/x/perf/cmd/benchstat@latest
```

**Workflow:**

```bash
# Before change — run 10 times for statistical confidence
go test -bench=. -benchmem -count=10 ./... | tee before.txt

# After change
go test -bench=. -benchmem -count=10 ./... | tee after.txt

# Compare
benchstat before.txt after.txt
```

**Reading output:**

```
                    │  before.txt  │             after.txt             │
                    │    sec/op    │    sec/op     vs base             │
Encode/id=1-8         234ns ± 2%    198ns ± 1%   -15.38% (p=0.000 n=10)
Encode/id=max-8       891ns ± 3%    876ns ± 2%    ~       (p=0.089 n=10)
GetByCode-8          1.23ms ± 3%   1.21ms ± 2%    -1.63% (p=0.041 n=10)
```

**Reading the columns:**
```
sec/op    → median time per operation
± N%      → confidence interval (95%) — smaller is better (more stable)
vs base   → % change from before (negative = faster)
p=N       → p-value: <0.05 = statistically significant, ≥0.05 = noise
~         → no significant difference (don't claim improvement!)
n=10      → number of samples used
```

**⚠️ Never claim improvement when you see `~`** — the result could be pure noise.

### 17.7 Generating Mocks with `mockgen`

For projects using testify, `mockgen` generates mock implementations automatically:

```bash
go install go.uber.org/mock/mockgen@latest
```

```go
// Source mode: generate from interface in a file
//go:generate mockgen -source=internal/shortener/storage.go -destination=internal/shortener/mock_storage_test.go -package=shortener_test

// Reflect mode: generate by importing the package
//go:generate mockgen -destination=mocks/mock_storage.go -package=mocks github.com/yourname/goshort/internal/shortener Storage
```

Run with:
```bash
go generate ./...
```

**When to use `mockgen` vs hand-written:**
```
mockgen is worth it when:
  → Interface has 5+ methods
  → Interface changes frequently (regenerate is easier than rewrite)
  → Team already uses gomock conventions

Hand-written is better when:
  → Interface has 2-4 methods
  → You want full transparency and control
  → You need custom logic in the mock (like GoShort's insertDirect)
  → You prefer avoiding code generation in your workflow
```

### 17.8 Summary: Testing Toolkit for Go Projects

```
Essential (use always):
  go test -race        → race detector
  go test -count=1     → prevent caching
  goleak               → goroutine leak detection
  t.Parallel()         → maximize test speed
  t.Helper()           → accurate failure locations
  t.Cleanup()          → safe teardown

Recommended (for mature projects):
  testify/assert+require → readable assertions
  testify/mock           → expectation-based mocking (or hand-written)
  golangci-lint          → thelper, paralleltest, testifylint
  gotests                → scaffold table-driven tests
  benchstat              → statistical benchmark comparison
  go test -shuffle=on    → find ordering dependencies

Advanced (for specific needs):
  rapid                  → property-based testing with shrinking
  clockwork              → deterministic time in tests
  testing/synctest       → Go 1.24+ deterministic goroutine testing
  mockgen                → auto-generate mocks from interfaces
  cupaloy                → snapshot testing for complex output
  go test -fuzz          → fuzzing for security/stability
  go tool cover          → coverage analysis
```

---

```
╭──────────────────────────────────────────────────────────╮
│                                                          │
│   "Write tests. Not too many. Mostly integration."      │
│                        — Guillermo Rauch (adapted)       │
│                                                          │
│   (But actually: write a LOT of unit tests,             │
│    enough integration tests, and few E2E tests.         │
│    The pyramid matters.)                                 │
│                                                          │
╰──────────────────────────────────────────────────────────╯
```

---

*Companion to: `DESIGN.md` · `LEARNING.md` · `check_test.md`*  
*GoShort v0.5.x — May 2026*
