-- name: CreateURL :one
INSERT INTO urls (short_code, original_url, is_custom, expires_at, title, description)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: UpdateMetadata :one
UPDATE urls SET title = ?, description = ? WHERE short_code = ? RETURNING *;

-- name: GetByCode :one
SELECT * FROM urls WHERE short_code = ?;

-- name: DeleteByCode :execrows
DELETE FROM urls WHERE short_code = ?;

-- name: ListURLs :many
SELECT * FROM urls ORDER BY id DESC LIMIT ? OFFSET ?;

-- name: CountURLs :one
SELECT COUNT(*) FROM urls;

-- name: IncrementClicks :exec
UPDATE urls SET click_count = click_count + 1 WHERE short_code = ?;

-- name: UpdateExpiry :one
UPDATE urls SET expires_at = ? WHERE short_code = ? RETURNING *;

-- name: DeleteExpired :execrows
DELETE FROM urls WHERE id IN (
    SELECT id FROM urls
    WHERE expires_at IS NOT NULL AND expires_at < datetime('now')
    LIMIT ?
);

-- name: GetCounter :one
SELECT value FROM counter WHERE id = 1;

-- name: IncrementCounter :one
UPDATE counter SET value = value + 1 WHERE id = 1 RETURNING value;
