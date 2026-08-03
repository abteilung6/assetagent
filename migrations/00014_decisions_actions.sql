-- +goose Up
-- +goose StatementBegin
CREATE TABLE decisions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    review_id     UUID REFERENCES money_reviews (id),
    scenario_id   UUID REFERENCES scenarios (id),
    title         TEXT NOT NULL,
    assumptions   JSONB NOT NULL DEFAULT '{}'::jsonb,
    target_value  NUMERIC(18, 2),
    decided_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT decisions_source_check CHECK (
        review_id IS NOT NULL OR scenario_id IS NOT NULL
    )
);

CREATE TABLE actions (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    decision_id              UUID NOT NULL REFERENCES decisions (id) ON DELETE CASCADE,
    title                    TEXT NOT NULL,
    expected_annual_effect   NUMERIC(18, 2) NOT NULL,
    due_on                   DATE NOT NULL,
    status                   TEXT NOT NULL DEFAULT 'planned',
    outcome_note             TEXT NOT NULL DEFAULT '',
    verified_at              TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT actions_status_check
        CHECK (status IN ('planned', 'done', 'skipped', 'irrelevant'))
);

CREATE INDEX decisions_review_id_idx ON decisions (review_id);
CREATE INDEX decisions_scenario_id_idx ON decisions (scenario_id);
CREATE INDEX decisions_decided_at_idx ON decisions (decided_at DESC);
CREATE INDEX actions_decision_id_idx ON actions (decision_id);
CREATE INDEX actions_status_due_idx ON actions (status, due_on);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS actions_status_due_idx;
DROP INDEX IF EXISTS actions_decision_id_idx;
DROP INDEX IF EXISTS decisions_decided_at_idx;
DROP INDEX IF EXISTS decisions_scenario_id_idx;
DROP INDEX IF EXISTS decisions_review_id_idx;
DROP TABLE IF EXISTS actions;
DROP TABLE IF EXISTS decisions;
-- +goose StatementEnd
