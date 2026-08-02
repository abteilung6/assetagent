-- +goose Up
-- +goose StatementBegin
CREATE TABLE transaction_classifications (
    transaction_id     UUID PRIMARY KEY REFERENCES transactions (id) ON DELETE CASCADE,
    category_id        UUID NOT NULL REFERENCES categories (id),
    merchant_id        UUID REFERENCES merchants (id),
    source             TEXT NOT NULL,
    confidence         TEXT NOT NULL,
    algorithm_version  TEXT NOT NULL,
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT transaction_classifications_source_check
        CHECK (source IN ('user_rule', 'exact_rule', 'merchant', 'heuristic', 'unresolved')),
    CONSTRAINT transaction_classifications_confidence_check
        CHECK (confidence IN ('high', 'medium', 'low'))
);

CREATE INDEX transaction_classifications_category_id_idx
    ON transaction_classifications (category_id);
CREATE INDEX transaction_classifications_source_idx
    ON transaction_classifications (source);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS transaction_classifications_source_idx;
DROP INDEX IF EXISTS transaction_classifications_category_id_idx;
DROP TABLE IF EXISTS transaction_classifications;
-- +goose StatementEnd
