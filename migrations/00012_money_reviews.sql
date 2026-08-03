-- +goose Up
-- +goose StatementBegin
CREATE TABLE money_reviews (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_id     UUID NOT NULL REFERENCES financial_baselines (id),
    period_from     DATE NOT NULL,
    period_to       DATE NOT NULL,
    status          TEXT NOT NULL,
    summary         TEXT NOT NULL,
    findings        JSONB NOT NULL DEFAULT '[]'::jsonb,
    data_freshness  TEXT NOT NULL DEFAULT '',
    confirmed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT money_reviews_status_check
        CHECK (status IN ('draft', 'needs_confirmation', 'confirmed', 'superseded')),
    CONSTRAINT money_reviews_period_check
        CHECK (period_to >= period_from)
);

CREATE INDEX money_reviews_created_idx ON money_reviews (created_at DESC);
CREATE INDEX money_reviews_baseline_id_idx ON money_reviews (baseline_id);
CREATE INDEX money_reviews_status_idx ON money_reviews (status);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS money_reviews_status_idx;
DROP INDEX IF EXISTS money_reviews_baseline_id_idx;
DROP INDEX IF EXISTS money_reviews_created_idx;
DROP TABLE IF EXISTS money_reviews;
-- +goose StatementEnd
