-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name       TEXT NOT NULL,
    bank               TEXT NOT NULL DEFAULT 'sparkasse',
    currency           TEXT NOT NULL DEFAULT 'EUR',
    order_account      TEXT,
    masked_identifier  TEXT NOT NULL DEFAULT '',
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX accounts_order_account_unique
    ON accounts (order_account)
    WHERE order_account IS NOT NULL AND order_account <> '';

CREATE TABLE import_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id      UUID NOT NULL REFERENCES accounts (id),
    source_filename TEXT NOT NULL DEFAULT '',
    file_hash       TEXT NOT NULL,
    parser_name     TEXT NOT NULL DEFAULT 'sparkasse',
    parser_version  TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL,
    period_from     DATE,
    period_to       DATE,
    row_total       INTEGER NOT NULL DEFAULT 0,
    row_valid       INTEGER NOT NULL DEFAULT 0,
    row_invalid     INTEGER NOT NULL DEFAULT 0,
    row_inserted    INTEGER NOT NULL DEFAULT 0,
    row_duplicate   INTEGER NOT NULL DEFAULT 0,
    warnings        JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    committed_at    TIMESTAMPTZ,
    rolled_back_at  TIMESTAMPTZ,
    CONSTRAINT import_runs_status_check
        CHECK (status IN ('committed', 'rolled_back', 'failed'))
);

CREATE INDEX import_runs_account_id_idx ON import_runs (account_id);
CREATE INDEX import_runs_created_at_idx ON import_runs (created_at DESC);

ALTER TABLE transactions
    ADD COLUMN account_id UUID REFERENCES accounts (id),
    ADD COLUMN import_run_id UUID REFERENCES import_runs (id);

CREATE INDEX transactions_account_id_idx ON transactions (account_id);
CREATE INDEX transactions_import_run_id_idx ON transactions (import_run_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS transactions_import_run_id_idx;
DROP INDEX IF EXISTS transactions_account_id_idx;
ALTER TABLE transactions
    DROP COLUMN IF EXISTS import_run_id,
    DROP COLUMN IF EXISTS account_id;

DROP INDEX IF EXISTS import_runs_created_at_idx;
DROP INDEX IF EXISTS import_runs_account_id_idx;
DROP TABLE IF EXISTS import_runs;

DROP INDEX IF EXISTS accounts_order_account_unique;
DROP TABLE IF EXISTS accounts;
-- +goose StatementEnd
