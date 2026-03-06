-- +goose Up
ALTER TABLE instances ADD COLUMN IF NOT EXISTS proxy_enabled BOOLEAN DEFAULT FALSE;
ALTER TABLE instances ADD COLUMN IF NOT EXISTS proxy_url TEXT;

-- +goose Down
ALTER TABLE instances DROP COLUMN IF EXISTS proxy_url;
ALTER TABLE instances DROP COLUMN IF EXISTS proxy_enabled;
