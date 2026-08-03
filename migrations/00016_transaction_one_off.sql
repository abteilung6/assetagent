-- +goose Up
-- +goose StatementBegin
ALTER TABLE transactions
    ADD COLUMN one_off BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX transactions_one_off_idx
    ON transactions (one_off)
    WHERE one_off = true;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS transactions_one_off_idx;
ALTER TABLE transactions
    DROP COLUMN IF EXISTS one_off;
-- +goose StatementEnd
