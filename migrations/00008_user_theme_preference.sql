-- +goose Up
-- +goose StatementBegin
ALTER TABLE users ADD COLUMN theme TEXT NOT NULL DEFAULT 'readit';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS theme;
-- +goose StatementEnd
