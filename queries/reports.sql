-- name: GetCashflow :one
SELECT
  COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0)::numeric AS income,
  COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0)::numeric AS expenses,
  COALESCE(SUM(amount), 0)::numeric AS net
FROM transactions
WHERE booking_date >= sqlc.arg('from_date')::date
  AND booking_date <= sqlc.arg('to_date')::date;

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
