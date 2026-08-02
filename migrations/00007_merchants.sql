-- +goose Up
-- +goose StatementBegin
CREATE TABLE merchants (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name         TEXT NOT NULL,
    default_category_id  UUID REFERENCES categories (id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE merchant_aliases (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    merchant_id  UUID NOT NULL REFERENCES merchants (id) ON DELETE CASCADE,
    match_type   TEXT NOT NULL,
    pattern      TEXT NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT merchant_aliases_match_type_check
        CHECK (match_type IN ('exact', 'normalized')),
    CONSTRAINT merchant_aliases_pattern_unique UNIQUE (match_type, pattern)
);

CREATE INDEX merchant_aliases_merchant_id_idx ON merchant_aliases (merchant_id);
CREATE INDEX merchants_display_name_idx ON merchants (display_name);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS merchants_display_name_idx;
DROP INDEX IF EXISTS merchant_aliases_merchant_id_idx;
DROP TABLE IF EXISTS merchant_aliases;
DROP TABLE IF EXISTS merchants;
-- +goose StatementEnd
