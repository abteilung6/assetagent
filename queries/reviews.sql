-- name: InsertMoneyReview :one
INSERT INTO money_reviews (
    household_id,
    baseline_id,
    period_from,
    period_to,
    status,
    summary,
    findings,
    data_freshness
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetMoneyReview :one
SELECT *
FROM money_reviews
WHERE id = $1 AND household_id = $2;

-- name: ListMoneyReviews :many
SELECT *
FROM money_reviews
WHERE household_id = sqlc.arg('household_id')
ORDER BY created_at DESC
LIMIT sqlc.arg('row_limit');

-- name: SupersedeOpenMoneyReviews :exec
UPDATE money_reviews
SET
    status = 'superseded',
    updated_at = now()
WHERE household_id = $1
  AND status IN ('draft', 'needs_confirmation', 'confirmed');

-- name: ConfirmMoneyReview :one
UPDATE money_reviews
SET
    status = 'confirmed',
    confirmed_at = now(),
    updated_at = now()
WHERE id = $1
  AND household_id = $2
  AND status IN ('draft', 'needs_confirmation')
RETURNING *;
