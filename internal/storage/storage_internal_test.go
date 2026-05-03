package storage

import (
	"database/sql"
	"testing"
	"time"

	idb "github.com/anIcedAntFA/goshort/internal/db"
)

func TestParseTime_Valid(t *testing.T) {
	t.Parallel()

	got, err := parseTime("2025-06-01 12:00:00")
	if err != nil {
		t.Fatalf("parseTime: %v", err)
	}
	if got.Year() != 2025 || got.Month() != 6 || got.Day() != 1 {
		t.Errorf("parseTime = %v, want 2025-06-01", got)
	}
	if got.Location() != time.UTC {
		t.Errorf("parseTime location = %v, want UTC", got.Location())
	}
}

func TestParseTime_InvalidFormat(t *testing.T) {
	t.Parallel()

	_, err := parseTime("not-a-timestamp")
	if err == nil {
		t.Error("parseTime(invalid) must return an error")
	}
}

func TestParseNullTime_Null(t *testing.T) {
	t.Parallel()

	got, err := parseNullTime(sql.NullString{})
	if err != nil {
		t.Fatalf("parseNullTime(null): %v", err)
	}
	if got != nil {
		t.Errorf("parseNullTime(null) = %v, want nil", got)
	}
}

func TestParseNullTime_Valid(t *testing.T) {
	t.Parallel()

	ns := sql.NullString{String: "2025-06-01 12:00:00", Valid: true}
	got, err := parseNullTime(ns)
	if err != nil {
		t.Fatalf("parseNullTime(valid): %v", err)
	}
	if got == nil {
		t.Fatal("parseNullTime(valid) returned nil, want non-nil")
	}
	if got.Year() != 2025 {
		t.Errorf("year = %d, want 2025", got.Year())
	}
}

func TestParseNullTime_InvalidFormat(t *testing.T) {
	t.Parallel()

	ns := sql.NullString{String: "garbage", Valid: true}
	_, err := parseNullTime(ns)
	if err == nil {
		t.Error("parseNullTime(invalid) must return an error")
	}
}

func TestBoolToInt(t *testing.T) {
	t.Parallel()

	if boolToInt(true) != 1 {
		t.Error("boolToInt(true) = 0, want 1")
	}
	if boolToInt(false) != 0 {
		t.Error("boolToInt(false) = 1, want 0")
	}
}

func TestTimeToNullString_Nil(t *testing.T) {
	t.Parallel()

	ns := timeToNullString(nil)
	if ns.Valid {
		t.Error("timeToNullString(nil).Valid = true, want false")
	}
}

func TestTimeToNullString_NonNil(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	ns := timeToNullString(&now)
	if !ns.Valid {
		t.Error("timeToNullString(non-nil).Valid = false, want true")
	}
	if ns.String == "" {
		t.Error("timeToNullString(non-nil).String is empty")
	}
}

func TestToURL_InvalidCreatedAt(t *testing.T) {
	t.Parallel()

	row := &idb.Url{
		ShortCode:   "abc",
		OriginalUrl: "https://example.com",
		CreatedAt:   "bad-timestamp",
	}
	_, err := toURL(row)
	if err == nil {
		t.Error("toURL(bad created_at) must return an error")
	}
}

func TestToURL_InvalidExpiresAt(t *testing.T) {
	t.Parallel()

	row := &idb.Url{
		ShortCode:   "abc",
		OriginalUrl: "https://example.com",
		CreatedAt:   "2025-06-01 12:00:00",
		ExpiresAt:   sql.NullString{String: "bad-timestamp", Valid: true},
	}
	_, err := toURL(row)
	if err == nil {
		t.Error("toURL(bad expires_at) must return an error")
	}
}
