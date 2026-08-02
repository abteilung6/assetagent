-- +goose Up
-- +goose StatementBegin
CREATE TABLE recurring_series (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fingerprint      TEXT NOT NULL,
    display_name     TEXT NOT NULL,
    cadence          TEXT NOT NULL,
    kind             TEXT NOT NULL,
    status           TEXT NOT NULL,
    amount_typical   NUMERIC(18, 2) NOT NULL,
    amount_last      NUMERIC(18, 2) NOT NULL,
    amount_changed   BOOLEAN NOT NULL DEFAULT false,
    next_expected    DATE,
    uncertainty      TEXT NOT NULL,
    member_count     INT NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT recurring_series_cadence_check
        CHECK (cadence IN ('monthly', 'quarterly', 'yearly')),
    CONSTRAINT recurring_series_kind_check
        CHECK (kind IN ('fixed', 'variable_regular', 'income')),
    CONSTRAINT recurring_series_status_check
        CHECK (status IN ('active', 'uncertain', 'ended')),
    CONSTRAINT recurring_series_uncertainty_check
        CHECK (uncertainty IN ('low', 'medium', 'high')),
    CONSTRAINT recurring_series_fingerprint_unique UNIQUE (fingerprint)
);

CREATE TABLE recurring_series_members (
    series_id      UUID NOT NULL REFERENCES recurring_series (id) ON DELETE CASCADE,
    transaction_id UUID NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    booking_date   DATE NOT NULL,
    amount         NUMERIC(18, 2) NOT NULL,
    PRIMARY KEY (series_id, transaction_id),
    CONSTRAINT recurring_series_members_tx_unique UNIQUE (transaction_id)
);

CREATE INDEX recurring_series_status_idx ON recurring_series (status);
CREATE INDEX recurring_series_members_series_id_idx ON recurring_series_members (series_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS recurring_series_members_series_id_idx;
DROP INDEX IF EXISTS recurring_series_status_idx;
DROP TABLE IF EXISTS recurring_series_members;
DROP TABLE IF EXISTS recurring_series;
-- +goose StatementEnd
