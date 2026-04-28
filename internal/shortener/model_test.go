package shortener_test

import (
	"testing"
	"time"

	"github.com/anIcedAntFA/goshort/internal/shortener"
)

func TestURLFields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	u := shortener.URL{
		ID:          1,
		ShortCode:   "abc123",
		OriginalURL: "https://example.com",
		IsCustom:    false,
		CreatedAt:   now,
		ExpiresAt:   &now,
		ClickCount:  5,
	}

	if u.ID != 1 {
		t.Errorf("ID: got %d, want 1", u.ID)
	}

	if u.ShortCode != "abc123" {
		t.Errorf("ShortCode: got %s, want abc123", u.ShortCode)
	}

	if u.OriginalURL != "https://example.com" {
		t.Errorf("OriginalURL: got %s, want https://example.com", u.OriginalURL)
	}

	if u.ClickCount != 5 {
		t.Errorf("ClickCount: got %d, want 5", u.ClickCount)
	}
}

func TestURLExpiresAtIsNilable(t *testing.T) {
	t.Parallel()

	u := shortener.URL{}
	if u.ExpiresAt != nil {
		t.Fatal("ExpiresAt zero value must be nil")
	}
}

func TestCreateRequestZeroValues(t *testing.T) {
	t.Parallel()

	req := shortener.CreateRequest{URL: "https://example.com"}

	if req.CustomAlias != "" {
		t.Error("CustomAlias zero value must be empty string")
	}

	if req.ExpiresIn != "" {
		t.Error("ExpiresIn zero value must be empty string")
	}
}

func TestListOptionsZeroValues(t *testing.T) {
	t.Parallel()

	opts := shortener.ListOptions{}

	if opts.Page != 0 {
		t.Error("Page zero value must be 0")
	}

	if opts.PerPage != 0 {
		t.Error("PerPage zero value must be 0")
	}
}
