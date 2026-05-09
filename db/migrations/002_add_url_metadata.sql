-- +goose Up
ALTER TABLE urls ADD COLUMN title TEXT NOT NULL DEFAULT '';
ALTER TABLE urls ADD COLUMN description TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE urls DROP COLUMN description;
ALTER TABLE urls DROP COLUMN title;
