-- +goose Up
-- +goose StatementBegin
CREATE TABLE transfer_pairs (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tx_out_id     UUID NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    tx_in_id      UUID NOT NULL REFERENCES transactions (id) ON DELETE CASCADE,
    status        TEXT NOT NULL,
    confidence    TEXT NOT NULL,
    rationale     JSONB NOT NULL DEFAULT '{}'::jsonb,
    confirmed_at  TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transfer_pairs_status_check
        CHECK (status IN ('suggested', 'confirmed', 'rejected')),
    CONSTRAINT transfer_pairs_confidence_check
        CHECK (confidence IN ('exact', 'probable')),
    CONSTRAINT transfer_pairs_distinct_txs_check
        CHECK (tx_out_id <> tx_in_id),
    CONSTRAINT transfer_pairs_pair_unique UNIQUE (tx_out_id, tx_in_id)
);

CREATE INDEX transfer_pairs_status_idx ON transfer_pairs (status);
CREATE INDEX transfer_pairs_tx_out_id_idx ON transfer_pairs (tx_out_id);
CREATE INDEX transfer_pairs_tx_in_id_idx ON transfer_pairs (tx_in_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS transfer_pairs_tx_in_id_idx;
DROP INDEX IF EXISTS transfer_pairs_tx_out_id_idx;
DROP INDEX IF EXISTS transfer_pairs_status_idx;
DROP TABLE IF EXISTS transfer_pairs;
-- +goose StatementEnd
