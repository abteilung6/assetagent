-- +goose Up
-- +goose StatementBegin
ALTER TABLE classification_rules
    ADD COLUMN confidence TEXT NOT NULL DEFAULT 'high',
    ADD COLUMN is_system BOOLEAN NOT NULL DEFAULT false;

ALTER TABLE classification_rules
    ADD CONSTRAINT classification_rules_confidence_check
    CHECK (confidence IN ('high', 'medium', 'low'));

CREATE UNIQUE INDEX classification_rules_pattern_unique
    ON classification_rules (lower(pattern))
    WHERE merchant_id IS NULL AND pattern IS NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS classification_rules_pattern_unique;
ALTER TABLE classification_rules
    DROP CONSTRAINT IF EXISTS classification_rules_confidence_check;
ALTER TABLE classification_rules
    DROP COLUMN IF EXISTS is_system,
    DROP COLUMN IF EXISTS confidence;
-- +goose StatementEnd
