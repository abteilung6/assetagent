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
    fingerprint,
    account_id,
    import_run_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
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
    fingerprint,
    account_id,
    import_run_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
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
    t.id,
    t.order_account,
    t.booking_date,
    t.value_date,
    t.booking_text,
    t.purpose,
    t.creditor_id,
    t.mandate_reference,
    t.end_to_end_reference,
    t.collection_reference,
    t.direct_debit_original_amount,
    t.chargeback_expense_reimbursement,
    t.counterparty,
    t.counterparty_iban,
    t.counterparty_bic,
    t.amount,
    t.currency,
    t.info,
    t.fingerprint,
    t.account_id,
    t.import_run_id,
    t.one_off,
    EXISTS (
        SELECT 1
        FROM recurring_series_members m
        WHERE m.transaction_id = t.id
    ) AS recurring
FROM transactions t
WHERE (sqlc.narg('from_date')::date IS NULL OR t.booking_date >= sqlc.narg('from_date')::date)
  AND (sqlc.narg('to_date')::date IS NULL OR t.booking_date <= sqlc.narg('to_date')::date)
  AND (sqlc.narg('account')::text IS NULL OR t.order_account = sqlc.narg('account'))
  AND (sqlc.narg('counterparty')::text IS NULL OR t.counterparty ILIKE sqlc.narg('counterparty') || '%')
  AND (sqlc.narg('min_amount')::numeric IS NULL OR t.amount >= sqlc.narg('min_amount')::numeric)
  AND (sqlc.narg('max_amount')::numeric IS NULL OR t.amount <= sqlc.narg('max_amount')::numeric)
  AND (
    sqlc.narg('search')::text IS NULL
    OR t.purpose ILIKE '%' || sqlc.narg('search') || '%'
    OR t.counterparty ILIKE '%' || sqlc.narg('search') || '%'
    OR t.booking_text ILIKE '%' || sqlc.narg('search') || '%'
  )
ORDER BY
  CASE WHEN sqlc.narg('sort_field') = 'amount' AND COALESCE(sqlc.narg('sort_asc'), false) THEN t.amount END ASC NULLS LAST,
  CASE WHEN sqlc.narg('sort_field') = 'amount' AND NOT COALESCE(sqlc.narg('sort_asc'), false) THEN t.amount END DESC NULLS LAST,
  CASE WHEN sqlc.narg('sort_field') = 'counterparty' AND COALESCE(sqlc.narg('sort_asc'), false) THEN t.counterparty END ASC NULLS LAST,
  CASE WHEN sqlc.narg('sort_field') = 'counterparty' AND NOT COALESCE(sqlc.narg('sort_asc'), false) THEN t.counterparty END DESC NULLS LAST,
  CASE WHEN (sqlc.narg('sort_field') IS NULL OR sqlc.narg('sort_field') = 'booking_date') AND COALESCE(sqlc.narg('sort_asc'), false) THEN t.booking_date END ASC NULLS LAST,
  CASE WHEN (sqlc.narg('sort_field') IS NULL OR sqlc.narg('sort_field') = 'booking_date') AND NOT COALESCE(sqlc.narg('sort_asc'), false) THEN t.booking_date END DESC NULLS LAST
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- name: SetTransactionOneOff :one
WITH updated AS (
    UPDATE transactions
    SET one_off = $2
    WHERE id = $1
    RETURNING
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
        fingerprint,
        account_id,
        import_run_id,
        one_off
)
SELECT
    updated.id,
    updated.order_account,
    updated.booking_date,
    updated.value_date,
    updated.booking_text,
    updated.purpose,
    updated.creditor_id,
    updated.mandate_reference,
    updated.end_to_end_reference,
    updated.collection_reference,
    updated.direct_debit_original_amount,
    updated.chargeback_expense_reimbursement,
    updated.counterparty,
    updated.counterparty_iban,
    updated.counterparty_bic,
    updated.amount,
    updated.currency,
    updated.info,
    updated.fingerprint,
    updated.account_id,
    updated.import_run_id,
    updated.one_off,
    EXISTS (
        SELECT 1
        FROM recurring_series_members m
        WHERE m.transaction_id = updated.id
    ) AS recurring
FROM updated;
