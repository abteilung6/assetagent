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
    info
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
)
RETURNING id;

-- name: CountTransactions :one
SELECT COUNT(*)::bigint AS count FROM transactions;
