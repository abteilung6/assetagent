-- name: ListTransactionsForRecurringScan :many
SELECT
    id,
    account_id,
    booking_date,
    amount,
    purpose,
    booking_text,
    counterparty
FROM transactions
WHERE household_id = $1
  AND account_id IS NOT NULL
ORDER BY booking_date ASC, id ASC;

-- name: ListRecurringMemberTransactionIDs :many
SELECT m.transaction_id
FROM recurring_series_members m
JOIN recurring_series s ON s.id = m.series_id
WHERE s.household_id = $1;

-- name: InsertRecurringSeries :one
INSERT INTO recurring_series (
    household_id,
    fingerprint,
    display_name,
    cadence,
    kind,
    status,
    amount_typical,
    amount_last,
    amount_changed,
    next_expected,
    uncertainty,
    member_count
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
)
ON CONFLICT (fingerprint) DO NOTHING
RETURNING *;

-- name: InsertRecurringSeriesMember :exec
INSERT INTO recurring_series_members (
    series_id,
    transaction_id,
    booking_date,
    amount
) VALUES (
    $1, $2, $3, $4
)
ON CONFLICT (transaction_id) DO NOTHING;

-- name: ListRecurringSeries :many
SELECT *
FROM recurring_series
WHERE household_id = $1
ORDER BY display_name ASC, created_at DESC;

-- name: ListUncertainRecurringSeries :many
SELECT *
FROM recurring_series
WHERE household_id = $1
  AND status = 'uncertain'
ORDER BY amount_typical DESC, display_name ASC;

-- name: GetRecurringSeries :one
SELECT *
FROM recurring_series
WHERE id = $1 AND household_id = $2;

-- name: ConfirmRecurringSeries :one
UPDATE recurring_series
SET
    status = 'active',
    updated_at = now()
WHERE id = $1
  AND household_id = $2
  AND status = 'uncertain'
RETURNING *;

-- name: RejectRecurringSeries :one
UPDATE recurring_series
SET
    status = 'ended',
    updated_at = now()
WHERE id = $1
  AND household_id = $2
  AND status = 'uncertain'
RETURNING *;

-- name: ListRecurringSeriesMembers :many
SELECT
    m.transaction_id,
    m.booking_date,
    m.amount,
    t.counterparty,
    t.purpose
FROM recurring_series_members m
JOIN transactions t ON t.id = m.transaction_id
JOIN recurring_series s ON s.id = m.series_id
WHERE m.series_id = sqlc.arg(series_id)
  AND s.household_id = sqlc.arg(household_id)
ORDER BY m.booking_date DESC, m.transaction_id DESC
LIMIT sqlc.arg(row_limit);
