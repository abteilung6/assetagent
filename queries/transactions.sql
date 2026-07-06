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
  AND (sqlc.narg('to_date')::date IS NULL OR booking_date <= sqlc.narg('to_date')::date);

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
ORDER BY booking_date DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');
