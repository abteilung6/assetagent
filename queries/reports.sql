-- name: GetCashflow :one
SELECT
  COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0)::numeric AS income,
  COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::numeric AS expenses,
  COALESCE(SUM(amount), 0)::numeric AS net
FROM transactions
WHERE booking_date >= sqlc.arg('from_date')::date
  AND booking_date <= sqlc.arg('to_date')::date;

-- name: GetCashflowV2 :one
SELECT
  COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0)::numeric AS income,
  COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::numeric AS expenses,
  COALESCE(SUM(amount), 0)::numeric AS net
FROM transactions t
WHERE t.booking_date >= sqlc.arg('from_date')::date
  AND t.booking_date <= sqlc.arg('to_date')::date
  AND NOT EXISTS (
    SELECT 1
    FROM transfer_pairs p
    WHERE p.status = 'confirmed'
      AND (p.tx_out_id = t.id OR p.tx_in_id = t.id)
  );

-- name: ListAccountsInPeriod :many
SELECT DISTINCT COALESCE(a.display_name, t.order_account, 'unknown')::text AS account_name
FROM transactions t
LEFT JOIN accounts a ON a.id = t.account_id
WHERE t.booking_date >= sqlc.arg('from_date')::date
  AND t.booking_date <= sqlc.arg('to_date')::date
ORDER BY account_name ASC;

-- name: ListConfirmedTransferIDsInPeriod :many
SELECT p.id
FROM transfer_pairs p
JOIN transactions out_tx ON out_tx.id = p.tx_out_id
JOIN transactions in_tx ON in_tx.id = p.tx_in_id
WHERE p.status = 'confirmed'
  AND (
    (out_tx.booking_date >= sqlc.arg('from_date')::date AND out_tx.booking_date <= sqlc.arg('to_date')::date)
    OR (in_tx.booking_date >= sqlc.arg('from_date')::date AND in_tx.booking_date <= sqlc.arg('to_date')::date)
  )
ORDER BY p.created_at ASC;

-- name: ListCashflowV2TransactionIDs :many
SELECT t.id
FROM transactions t
WHERE t.booking_date >= sqlc.arg('from_date')::date
  AND t.booking_date <= sqlc.arg('to_date')::date
  AND NOT EXISTS (
    SELECT 1
    FROM transfer_pairs p
    WHERE p.status = 'confirmed'
      AND (p.tx_out_id = t.id OR p.tx_in_id = t.id)
  )
ORDER BY t.booking_date ASC, t.id ASC
LIMIT sqlc.arg('row_limit');

-- name: GetLatestBookingDate :one
SELECT MAX(booking_date)::date AS latest_booking_date
FROM transactions;

-- name: GetTopCounterparties :many
SELECT
  counterparty,
  SUM(-amount)::numeric AS total_spent,
  COUNT(*)::bigint AS transaction_count
FROM transactions
WHERE booking_date >= sqlc.arg('from_date')::date
  AND booking_date <= sqlc.arg('to_date')::date
  AND amount < 0
  AND counterparty <> ''
GROUP BY counterparty
ORDER BY total_spent DESC, counterparty ASC
LIMIT sqlc.arg('row_limit');
