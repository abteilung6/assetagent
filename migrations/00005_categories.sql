-- +goose Up
-- +goose StatementBegin
CREATE TABLE categories (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug          TEXT NOT NULL UNIQUE,
    display_name  TEXT NOT NULL,
    kind          TEXT NOT NULL,
    parent_id     UUID REFERENCES categories (id),
    is_system     BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT categories_kind_check
        CHECK (kind IN ('income', 'expense', 'transfer', 'saving', 'other'))
);

INSERT INTO categories (slug, display_name, kind, is_system) VALUES
    ('income', 'Einkommen', 'income', true),
    ('housing', 'Wohnen', 'expense', true),
    ('insurance', 'Versicherungen', 'expense', true),
    ('mobility', 'Mobilität', 'expense', true),
    ('groceries', 'Lebensmittel', 'expense', true),
    ('leisure', 'Freizeit', 'expense', true),
    ('health', 'Gesundheit', 'expense', true),
    ('saving_investing', 'Sparen/Investieren', 'saving', true),
    ('transfer', 'Transfer', 'transfer', true),
    ('other', 'Sonstiges', 'other', true);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS categories;
-- +goose StatementEnd
