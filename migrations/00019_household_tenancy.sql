-- +goose Up
-- +goose StatementBegin
INSERT INTO households (name)
SELECT 'Local seed'
WHERE NOT EXISTS (SELECT 1 FROM households);

ALTER TABLE accounts ADD COLUMN household_id UUID;
ALTER TABLE import_runs ADD COLUMN household_id UUID;
ALTER TABLE transactions ADD COLUMN household_id UUID;
ALTER TABLE merchants ADD COLUMN household_id UUID;
ALTER TABLE classification_rules ADD COLUMN household_id UUID;
ALTER TABLE transfer_pairs ADD COLUMN household_id UUID;
ALTER TABLE recurring_series ADD COLUMN household_id UUID;
ALTER TABLE financial_baselines ADD COLUMN household_id UUID;
ALTER TABLE money_reviews ADD COLUMN household_id UUID;
ALTER TABLE forecasts ADD COLUMN household_id UUID;
ALTER TABLE scenarios ADD COLUMN household_id UUID;
ALTER TABLE decisions ADD COLUMN household_id UUID;

DO $$
DECLARE
    seed_id UUID;
BEGIN
    SELECT id INTO seed_id FROM households ORDER BY created_at LIMIT 1;

    UPDATE accounts SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE import_runs SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE transactions SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE merchants SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE classification_rules SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE transfer_pairs SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE recurring_series SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE financial_baselines SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE money_reviews SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE forecasts SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE scenarios SET household_id = seed_id WHERE household_id IS NULL;
    UPDATE decisions SET household_id = seed_id WHERE household_id IS NULL;
END $$;

ALTER TABLE accounts
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT accounts_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE import_runs
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT import_runs_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE transactions
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT transactions_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE merchants
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT merchants_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE classification_rules
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT classification_rules_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE transfer_pairs
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT transfer_pairs_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE recurring_series
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT recurring_series_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE financial_baselines
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT financial_baselines_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE money_reviews
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT money_reviews_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE forecasts
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT forecasts_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE scenarios
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT scenarios_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);
ALTER TABLE decisions
    ALTER COLUMN household_id SET NOT NULL,
    ADD CONSTRAINT decisions_household_id_fkey
        FOREIGN KEY (household_id) REFERENCES households (id);

CREATE INDEX accounts_household_id_idx ON accounts (household_id);
CREATE INDEX import_runs_household_id_idx ON import_runs (household_id);
CREATE INDEX transactions_household_id_idx ON transactions (household_id);
CREATE INDEX merchants_household_id_idx ON merchants (household_id);
CREATE INDEX classification_rules_household_id_idx ON classification_rules (household_id);
CREATE INDEX transfer_pairs_household_id_idx ON transfer_pairs (household_id);
CREATE INDEX recurring_series_household_id_idx ON recurring_series (household_id);
CREATE INDEX financial_baselines_household_id_idx ON financial_baselines (household_id);
CREATE INDEX money_reviews_household_id_idx ON money_reviews (household_id);
CREATE INDEX forecasts_household_id_idx ON forecasts (household_id);
CREATE INDEX scenarios_household_id_idx ON scenarios (household_id);
CREATE INDEX decisions_household_id_idx ON decisions (household_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS decisions_household_id_idx;
DROP INDEX IF EXISTS scenarios_household_id_idx;
DROP INDEX IF EXISTS forecasts_household_id_idx;
DROP INDEX IF EXISTS money_reviews_household_id_idx;
DROP INDEX IF EXISTS financial_baselines_household_id_idx;
DROP INDEX IF EXISTS recurring_series_household_id_idx;
DROP INDEX IF EXISTS transfer_pairs_household_id_idx;
DROP INDEX IF EXISTS classification_rules_household_id_idx;
DROP INDEX IF EXISTS merchants_household_id_idx;
DROP INDEX IF EXISTS transactions_household_id_idx;
DROP INDEX IF EXISTS import_runs_household_id_idx;
DROP INDEX IF EXISTS accounts_household_id_idx;

ALTER TABLE decisions DROP CONSTRAINT IF EXISTS decisions_household_id_fkey;
ALTER TABLE decisions DROP COLUMN IF EXISTS household_id;
ALTER TABLE scenarios DROP CONSTRAINT IF EXISTS scenarios_household_id_fkey;
ALTER TABLE scenarios DROP COLUMN IF EXISTS household_id;
ALTER TABLE forecasts DROP CONSTRAINT IF EXISTS forecasts_household_id_fkey;
ALTER TABLE forecasts DROP COLUMN IF EXISTS household_id;
ALTER TABLE money_reviews DROP CONSTRAINT IF EXISTS money_reviews_household_id_fkey;
ALTER TABLE money_reviews DROP COLUMN IF EXISTS household_id;
ALTER TABLE financial_baselines DROP CONSTRAINT IF EXISTS financial_baselines_household_id_fkey;
ALTER TABLE financial_baselines DROP COLUMN IF EXISTS household_id;
ALTER TABLE recurring_series DROP CONSTRAINT IF EXISTS recurring_series_household_id_fkey;
ALTER TABLE recurring_series DROP COLUMN IF EXISTS household_id;
ALTER TABLE transfer_pairs DROP CONSTRAINT IF EXISTS transfer_pairs_household_id_fkey;
ALTER TABLE transfer_pairs DROP COLUMN IF EXISTS household_id;
ALTER TABLE classification_rules DROP CONSTRAINT IF EXISTS classification_rules_household_id_fkey;
ALTER TABLE classification_rules DROP COLUMN IF EXISTS household_id;
ALTER TABLE merchants DROP CONSTRAINT IF EXISTS merchants_household_id_fkey;
ALTER TABLE merchants DROP COLUMN IF EXISTS household_id;
ALTER TABLE transactions DROP CONSTRAINT IF EXISTS transactions_household_id_fkey;
ALTER TABLE transactions DROP COLUMN IF EXISTS household_id;
ALTER TABLE import_runs DROP CONSTRAINT IF EXISTS import_runs_household_id_fkey;
ALTER TABLE import_runs DROP COLUMN IF EXISTS household_id;
ALTER TABLE accounts DROP CONSTRAINT IF EXISTS accounts_household_id_fkey;
ALTER TABLE accounts DROP COLUMN IF EXISTS household_id;
-- +goose StatementEnd
