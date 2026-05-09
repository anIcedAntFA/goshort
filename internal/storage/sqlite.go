package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/anIcedAntFA/goshort/db/migrations"
	idb "github.com/anIcedAntFA/goshort/internal/db"
	"github.com/anIcedAntFA/goshort/internal/shortener"
	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // register "sqlite" driver
)

// sqliteTimeLayout is the format SQLite's datetime() function produces.
// All times are stored and parsed as UTC.
const sqliteTimeLayout = "2006-01-02 15:04:05"

// gooseOnce ensures goose global configuration is set exactly once across all
// goroutines (tests, concurrent server startup) since goose uses global state.
var gooseOnce sync.Once

func initGoose() {
	gooseOnce.Do(func() {
		goose.SetBaseFS(migrations.FS)
		goose.SetLogger(goose.NopLogger())
		if err := goose.SetDialect("sqlite3"); err != nil {
			panic(fmt.Sprintf("storage: goose set dialect: %v", err))
		}
	})
}

// SQLiteStorage implements Storage using a SQLite database via sqlc-generated queries.
type SQLiteStorage struct {
	db *sql.DB
	q  *idb.Queries
}

// compile-time interface check.
var _ shortener.Storage = (*SQLiteStorage)(nil)

// NewSQLiteStorage opens (or creates) the database at dsn, applies goose migrations,
// and returns a ready-to-use SQLiteStorage. Caller must call Close when done.
func NewSQLiteStorage(ctx context.Context, dsn string) (*SQLiteStorage, error) {
	initGoose()

	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// Single connection keeps `:memory:` consistent and avoids SQLite write-lock contention.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if err := goose.Up(sqlDB, "."); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("apply migrations: %w", err)
	}

	return &SQLiteStorage{db: sqlDB, q: idb.New(sqlDB)}, nil
}

// Close releases the underlying database connection.
func (s *SQLiteStorage) Close() error {
	return s.db.Close()
}

// CreateURL inserts a new URL record and returns the created row.
func (s *SQLiteStorage) CreateURL(ctx context.Context, params *shortener.CreateParams) (shortener.URL, error) {
	row, err := s.q.CreateURL(ctx, idb.CreateURLParams{
		ShortCode:   params.ShortCode,
		OriginalUrl: params.OriginalURL,
		IsCustom:    boolToInt(params.IsCustom),
		ExpiresAt:   timeToNullString(params.ExpiresAt),
		Title:       params.Title,
		Description: params.Description,
	})
	if err != nil {
		return shortener.URL{}, fmt.Errorf("create url: %w", err)
	}

	return toURL(&row)
}

// GetByCode retrieves a URL record by its short code.
// Returns ErrNotFound (wrapping shortener.ErrNotFound) if not found.
func (s *SQLiteStorage) GetByCode(ctx context.Context, code string) (*shortener.URL, error) {
	row, err := s.q.GetByCode(ctx, code)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("get by code %q: %w", code, shortener.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("get by code: %w", err)
	}

	u, err := toURL(&row)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// DeleteByCode removes a URL record by its short code.
// Returns shortener.ErrNotFound if no matching record exists.
func (s *SQLiteStorage) DeleteByCode(ctx context.Context, code string) error {
	n, err := s.q.DeleteByCode(ctx, code)
	if err != nil {
		return fmt.Errorf("delete by code: %w", err)
	}

	if n == 0 {
		return fmt.Errorf("delete by code %q: %w", code, shortener.ErrNotFound)
	}

	return nil
}

// ListURLs returns a page of URL records ordered by creation time descending.
func (s *SQLiteStorage) ListURLs(ctx context.Context, limit, offset int) ([]shortener.URL, error) {
	rows, err := s.q.ListURLs(ctx, idb.ListURLsParams{
		Limit:  int64(limit),
		Offset: int64(offset),
	})
	if err != nil {
		return nil, fmt.Errorf("list urls: %w", err)
	}

	result := make([]shortener.URL, 0, len(rows))
	for i := range rows {
		u, err := toURL(&rows[i])
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}

	return result, nil
}

// CountURLs returns the total number of URL records.
func (s *SQLiteStorage) CountURLs(ctx context.Context) (int, error) {
	n, err := s.q.CountURLs(ctx)
	if err != nil {
		return 0, fmt.Errorf("count urls: %w", err)
	}

	return int(n), nil //nolint:gosec // COUNT(*) is always non-negative and fits int
}

// IncrementClicks atomically increments the click counter for a URL.
func (s *SQLiteStorage) IncrementClicks(ctx context.Context, code string) error {
	if err := s.q.IncrementClicks(ctx, code); err != nil {
		return fmt.Errorf("increment clicks: %w", err)
	}

	return nil
}

// DeleteExpired removes up to batchSize expired URL records and returns the count deleted.
func (s *SQLiteStorage) DeleteExpired(ctx context.Context, batchSize int) (int64, error) {
	n, err := s.q.DeleteExpired(ctx, int64(batchSize)) //nolint:gosec // batchSize is caller-controlled and positive
	if err != nil {
		return 0, fmt.Errorf("delete expired: %w", err)
	}

	return n, nil
}

// UpdateExpiry sets or clears the expiry timestamp for a URL by short code.
// Returns shortener.ErrNotFound if no matching record exists.
func (s *SQLiteStorage) UpdateExpiry(ctx context.Context, code string, expiresAt *time.Time) (*shortener.URL, error) {
	row, err := s.q.UpdateExpiry(ctx, idb.UpdateExpiryParams{
		ExpiresAt: timeToNullString(expiresAt),
		ShortCode: code,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("update expiry %q: %w", code, shortener.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("update expiry: %w", err)
	}

	u, err := toURL(&row)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetCounter returns the current value of the global URL counter.
func (s *SQLiteStorage) GetCounter(ctx context.Context) (int64, error) {
	n, err := s.q.GetCounter(ctx)
	if err != nil {
		return 0, fmt.Errorf("get counter: %w", err)
	}

	return n, nil
}

// IncrementCounter atomically increments the global URL counter and returns the new value.
func (s *SQLiteStorage) IncrementCounter(ctx context.Context) (int64, error) {
	n, err := s.q.IncrementCounter(ctx)
	if err != nil {
		return 0, fmt.Errorf("increment counter: %w", err)
	}

	return n, nil
}

// Type conversion helpers.

func boolToInt(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func timeToNullString(t *time.Time) sql.NullString {
	if t == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: t.UTC().Format(sqliteTimeLayout), Valid: true}
}

func parseTime(s string) (time.Time, error) {
	t, err := time.ParseInLocation(sqliteTimeLayout, s, time.UTC)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return t, nil
}

func parseNullTime(ns sql.NullString) (*time.Time, error) {
	if !ns.Valid {
		return nil, nil
	}
	t, err := parseTime(ns.String)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func toURL(row *idb.Url) (shortener.URL, error) {
	createdAt, err := parseTime(row.CreatedAt)
	if err != nil {
		return shortener.URL{}, fmt.Errorf("decode created_at: %w", err)
	}

	expiresAt, err := parseNullTime(row.ExpiresAt)
	if err != nil {
		return shortener.URL{}, fmt.Errorf("decode expires_at: %w", err)
	}

	return shortener.URL{
		ID:          row.ID,
		ShortCode:   row.ShortCode,
		OriginalURL: row.OriginalUrl,
		IsCustom:    row.IsCustom != 0,
		CreatedAt:   createdAt,
		ExpiresAt:   expiresAt,
		ClickCount:  row.ClickCount,
		Title:       row.Title,
		Description: row.Description,
	}, nil
}
