-- name: InsertFinancialBaseline :one
INSERT INTO financial_baselines (
    household_id,
    period_from,
    period_to,
    algorithm_version,
    status,
    regular_monthly_income,
    monthly_fixed_costs,
    monthly_irregular_costs,
    avg_variable_spend,
    sustainable_free_cashflow,
    confidence,
    assumptions,
    evidence
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)
RETURNING *;

-- name: GetFinancialBaseline :one
SELECT *
FROM financial_baselines
WHERE id = $1 AND household_id = $2;

-- name: GetCurrentFinancialBaseline :one
SELECT *
FROM financial_baselines
WHERE household_id = $1
  AND status IN ('draft', 'confirmed')
ORDER BY
    CASE status WHEN 'confirmed' THEN 0 WHEN 'draft' THEN 1 ELSE 2 END,
    created_at DESC
LIMIT 1;

-- name: SupersedeOpenFinancialBaselines :exec
UPDATE financial_baselines
SET
    status = 'superseded',
    updated_at = now()
WHERE household_id = $1
  AND status IN ('draft', 'confirmed');

-- name: ConfirmFinancialBaseline :one
UPDATE financial_baselines
SET
    status = 'confirmed',
    confirmed_at = now(),
    updated_at = now()
WHERE id = $1
  AND household_id = $2
  AND status = 'draft'
RETURNING *;

-- name: InsertBaselineAdjustment :one
INSERT INTO baseline_adjustments (
    baseline_id,
    metric_key,
    previous_value,
    new_value,
    reason
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;
