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
WHERE account_id IS NOT NULL
ORDER BY booking_date ASC, id ASC;

-- name: ListRecurringMemberTransactionIDs :many
SELECT transaction_id
FROM recurring_series_members;

-- name: InsertRecurringSeries :one
INSERT INTO recurring_series (
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
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
ORDER BY display_name ASC, created_at DESC;

-- name: ListRecurringSeriesMembers :many
SELECT series_id, transaction_id, booking_date, amount
FROM recurring_series_members
ORDER BY booking_date ASC;
