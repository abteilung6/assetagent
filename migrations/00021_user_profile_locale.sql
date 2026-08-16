-- +goose Up
-- +goose StatementBegin
ALTER TABLE users
    ADD COLUMN given_name TEXT,
    ADD COLUMN picture_url TEXT,
    ADD COLUMN preferred_locale TEXT NOT NULL DEFAULT 'de';

ALTER TABLE users
    ADD CONSTRAINT users_preferred_locale_check
    CHECK (preferred_locale IN ('de', 'en'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_preferred_locale_check;
ALTER TABLE users
    DROP COLUMN IF EXISTS preferred_locale,
    DROP COLUMN IF EXISTS picture_url,
    DROP COLUMN IF EXISTS given_name;
-- +goose StatementEnd
