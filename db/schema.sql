CREATE TABLE urls (
  id           INTEGER PRIMARY KEY AUTOINCREMENT,
  short_code   TEXT    UNIQUE NOT NULL,
  original_url TEXT    NOT NULL,
  is_custom    INTEGER NOT NULL DEFAULT 0,
  created_at   TEXT    NOT NULL DEFAULT (datetime('now')),
  expires_at   TEXT,
  click_count  INTEGER NOT NULL DEFAULT 0,
  title        TEXT    NOT NULL DEFAULT '',
  description  TEXT    NOT NULL DEFAULT ''
);

CREATE TABLE counter (
  id    INTEGER PRIMARY KEY CHECK (id = 1),
  value INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_short_code ON urls(short_code);
CREATE INDEX idx_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;
