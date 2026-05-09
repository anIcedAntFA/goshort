-- +goose Up
CREATE TABLE IF NOT EXISTS urls (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    short_code   TEXT    UNIQUE NOT NULL,
    original_url TEXT    NOT NULL,
    is_custom    INTEGER NOT NULL DEFAULT 0,
    created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
    expires_at   TEXT,
    click_count  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS counter (
    id    INTEGER PRIMARY KEY CHECK (id = 1),
    value INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_short_code ON urls(short_code);
CREATE INDEX IF NOT EXISTS idx_expires_at ON urls(expires_at)
    WHERE expires_at IS NOT NULL;

INSERT OR IGNORE INTO counter (id, value) VALUES (1, 0);

-- +goose Down
DROP INDEX IF EXISTS idx_expires_at;
DROP INDEX IF EXISTS idx_short_code;
DROP TABLE IF EXISTS counter;
DROP TABLE IF EXISTS urls;
