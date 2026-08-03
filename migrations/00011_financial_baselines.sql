-- +goose Up
-- +goose StatementBegin
CREATE TABLE financial_baselines (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    period_from                DATE NOT NULL,
    period_to                  DATE NOT NULL,
    algorithm_version          TEXT NOT NULL,
    status                     TEXT NOT NULL,
    regular_monthly_income     NUMERIC(18, 2) NOT NULL,
    monthly_fixed_costs        NUMERIC(18, 2) NOT NULL,
    monthly_irregular_costs    NUMERIC(18, 2) NOT NULL,
    avg_variable_spend         NUMERIC(18, 2) NOT NULL,
    sustainable_free_cashflow  NUMERIC(18, 2) NOT NULL,
    confidence                 TEXT NOT NULL,
    assumptions                JSONB NOT NULL DEFAULT '[]'::jsonb,
    evidence                   JSONB NOT NULL DEFAULT '[]'::jsonb,
    confirmed_at               TIMESTAMPTZ,
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT financial_baselines_status_check
        CHECK (status IN ('draft', 'confirmed', 'superseded')),
    CONSTRAINT financial_baselines_confidence_check
        CHECK (confidence IN ('high', 'medium', 'low')),
    CONSTRAINT financial_baselines_period_check
        CHECK (period_to >= period_from)
);

CREATE TABLE baseline_adjustments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_id     UUID NOT NULL REFERENCES financial_baselines (id) ON DELETE CASCADE,
    metric_key      TEXT NOT NULL,
    previous_value  NUMERIC(18, 2) NOT NULL,
    new_value       NUMERIC(18, 2) NOT NULL,
    reason          TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT baseline_adjustments_metric_check
        CHECK (metric_key IN (
            'regular_monthly_income',
            'monthly_fixed_costs',
            'monthly_irregular_costs',
            'avg_variable_spend',
            'sustainable_free_cashflow'
        ))
);

CREATE INDEX financial_baselines_status_created_idx
    ON financial_baselines (status, created_at DESC);
CREATE INDEX baseline_adjustments_baseline_id_idx
    ON baseline_adjustments (baseline_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS baseline_adjustments_baseline_id_idx;
DROP INDEX IF EXISTS financial_baselines_status_created_idx;
DROP TABLE IF EXISTS baseline_adjustments;
DROP TABLE IF EXISTS financial_baselines;
-- +goose StatementEnd
