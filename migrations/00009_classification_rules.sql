-- +goose Up
-- +goose StatementBegin
CREATE TABLE classification_rules (
    id                           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    priority                     INTEGER NOT NULL DEFAULT 100,
    merchant_id                  UUID REFERENCES merchants (id) ON DELETE CASCADE,
    pattern                      TEXT,
    category_id                  UUID NOT NULL REFERENCES categories (id),
    created_from_transaction_id  UUID REFERENCES transactions (id) ON DELETE SET NULL,
    created_at                   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT classification_rules_match_check
        CHECK (merchant_id IS NOT NULL OR (pattern IS NOT NULL AND pattern <> ''))
);

CREATE UNIQUE INDEX classification_rules_merchant_unique
    ON classification_rules (merchant_id)
    WHERE merchant_id IS NOT NULL;

CREATE INDEX classification_rules_category_id_idx ON classification_rules (category_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS classification_rules_category_id_idx;
DROP INDEX IF EXISTS classification_rules_merchant_unique;
DROP TABLE IF EXISTS classification_rules;
-- +goose StatementEnd
