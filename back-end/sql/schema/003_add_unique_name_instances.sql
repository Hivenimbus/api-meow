-- +goose Up
ALTER TABLE instances ADD CONSTRAINT instances_name_key UNIQUE (name);

-- +goose Down
ALTER TABLE instances DROP CONSTRAINT instances_name_key;
