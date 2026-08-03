-- +goose Up
-- +goose StatementBegin
CREATE TABLE forecasts (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_id       UUID NOT NULL REFERENCES financial_baselines (id),
    horizon_days      INT NOT NULL DEFAULT 90,
    starting_balance  NUMERIC(18, 2) NOT NULL,
    assumptions       JSONB NOT NULL DEFAULT '{}'::jsonb,
    series            JSONB NOT NULL DEFAULT '[]'::jsonb,
    min_balance       NUMERIC(18, 2) NOT NULL,
    ending_balance    NUMERIC(18, 2) NOT NULL,
    algorithm_version TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT forecasts_horizon_check CHECK (horizon_days > 0)
);

CREATE TABLE scenarios (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    forecast_id  UUID NOT NULL REFERENCES forecasts (id) ON DELETE CASCADE,
    kind         TEXT NOT NULL,
    params       JSONB NOT NULL DEFAULT '{}'::jsonb,
    result       JSONB NOT NULL DEFAULT '{}'::jsonb,
    status       TEXT NOT NULL DEFAULT 'confirmed',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT scenarios_kind_check
        CHECK (kind IN ('new_monthly_obligation', 'income_gap', 'one_off_plus_goal')),
    CONSTRAINT scenarios_status_check
        CHECK (status IN ('proposed', 'confirmed', 'discarded'))
);

CREATE INDEX forecasts_baseline_id_idx ON forecasts (baseline_id);
CREATE INDEX forecasts_created_idx ON forecasts (created_at DESC);
CREATE INDEX scenarios_forecast_id_idx ON scenarios (forecast_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS scenarios_forecast_id_idx;
DROP INDEX IF EXISTS forecasts_created_idx;
DROP INDEX IF EXISTS forecasts_baseline_id_idx;
DROP TABLE IF EXISTS scenarios;
DROP TABLE IF EXISTS forecasts;
-- +goose StatementEnd
