-- name: InsertTransaction :one
INSERT INTO transactions (
    order_account,
    booking_date,
    value_date,
    booking_text,
    purpose,
    creditor_id,
    mandate_reference,
    end_to_end_reference,
    collection_reference,
    direct_debit_original_amount,
    chargeback_expense_reimbursement,
    counterparty,
    counterparty_iban,
    counterparty_bic,
    amount,
    currency,
    info,
    fingerprint
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
RETURNING id;

-- name: InsertTransactionIfNew :one
INSERT INTO transactions (
    order_account,
    booking_date,
    value_date,
    booking_text,
    purpose,
    creditor_id,
    mandate_reference,
    end_to_end_reference,
    collection_reference,
    direct_debit_original_amount,
    chargeback_expense_reimbursement,
    counterparty,
    counterparty_iban,
    counterparty_bic,
    amount,
    currency,
    info,
    fingerprint
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
)
ON CONFLICT (fingerprint) DO NOTHING
RETURNING id;

-- name: CountTransactions :one
SELECT COUNT(*)::bigint AS count FROM transactions;

-- name: CountTransactionsFiltered :one
SELECT COUNT(*)::bigint AS count
FROM transactions
WHERE (sqlc.narg('from_date')::date IS NULL OR booking_date >= sqlc.narg('from_date')::date)
  AND (sqlc.narg('to_date')::date IS NULL OR booking_date <= sqlc.narg('to_date')::date)
  AND (sqlc.narg('account')::text IS NULL OR order_account = sqlc.narg('account'))
  AND (sqlc.narg('counterparty')::text IS NULL OR counterparty ILIKE sqlc.narg('counterparty') || '%')
  AND (sqlc.narg('min_amount')::numeric IS NULL OR amount >= sqlc.narg('min_amount')::numeric)
  AND (sqlc.narg('max_amount')::numeric IS NULL OR amount <= sqlc.narg('max_amount')::numeric)
  AND (
    sqlc.narg('search')::text IS NULL
    OR purpose ILIKE '%' || sqlc.narg('search') || '%'
    OR counterparty ILIKE '%' || sqlc.narg('search') || '%'
    OR booking_text ILIKE '%' || sqlc.narg('search') || '%'
  );

-- name: ListTransactions :many
SELECT
    id,
    order_account,
    booking_date,
    value_date,
    booking_text,
    purpose,
    creditor_id,
    mandate_reference,
    end_to_end_reference,
    collection_reference,
    direct_debit_original_amount,
    chargeback_expense_reimbursement,
    counterparty,
    counterparty_iban,
    counterparty_bic,
    amount,
    currency,
    info,
    fingerprint
FROM transactions
WHERE (sqlc.narg('from_date')::date IS NULL OR booking_date >= sqlc.narg('from_date')::date)
  AND (sqlc.narg('to_date')::date IS NULL OR booking_date <= sqlc.narg('to_date')::date)
  AND (sqlc.narg('account')::text IS NULL OR order_account = sqlc.narg('account'))
  AND (sqlc.narg('counterparty')::text IS NULL OR counterparty ILIKE sqlc.narg('counterparty') || '%')
  AND (sqlc.narg('min_amount')::numeric IS NULL OR amount >= sqlc.narg('min_amount')::numeric)
  AND (sqlc.narg('max_amount')::numeric IS NULL OR amount <= sqlc.narg('max_amount')::numeric)
  AND (
    sqlc.narg('search')::text IS NULL
    OR purpose ILIKE '%' || sqlc.narg('search') || '%'
    OR counterparty ILIKE '%' || sqlc.narg('search') || '%'
    OR booking_text ILIKE '%' || sqlc.narg('search') || '%'
  )
ORDER BY
  CASE WHEN sqlc.narg('sort_field') = 'amount' AND COALESCE(sqlc.narg('sort_asc'), false) THEN amount END ASC NULLS LAST,
  CASE WHEN sqlc.narg('sort_field') = 'amount' AND NOT COALESCE(sqlc.narg('sort_asc'), false) THEN amount END DESC NULLS LAST,
  CASE WHEN sqlc.narg('sort_field') = 'counterparty' AND COALESCE(sqlc.narg('sort_asc'), false) THEN counterparty END ASC NULLS LAST,
  CASE WHEN sqlc.narg('sort_field') = 'counterparty' AND NOT COALESCE(sqlc.narg('sort_asc'), false) THEN counterparty END DESC NULLS LAST,
  CASE WHEN (sqlc.narg('sort_field') IS NULL OR sqlc.narg('sort_field') = 'booking_date') AND COALESCE(sqlc.narg('sort_asc'), false) THEN booking_date END ASC NULLS LAST,
  CASE WHEN (sqlc.narg('sort_field') IS NULL OR sqlc.narg('sort_field') = 'booking_date') AND NOT COALESCE(sqlc.narg('sort_asc'), false) THEN booking_date END DESC NULLS LAST
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
