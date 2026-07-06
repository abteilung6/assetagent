-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions ADD COLUMN fingerprint TEXT NOT NULL DEFAULT '';
ALTER TABLE transactions ALTER COLUMN fingerprint DROP DEFAULT;
ALTER TABLE transactions ADD CONSTRAINT transactions_fingerprint_unique UNIQUE (fingerprint);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE transactions DROP CONSTRAINT transactions_fingerprint_unique;
ALTER TABLE transactions DROP COLUMN fingerprint;
-- +goose StatementEnd
