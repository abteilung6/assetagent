-- name: ListTransactionsForTransferScan :many
SELECT
    id,
    account_id,
    booking_date,
    amount,
    purpose,
    booking_text,
    counterparty,
    counterparty_iban
FROM transactions
WHERE account_id IS NOT NULL
ORDER BY booking_date ASC, id ASC;

-- name: ListTransferPairLegs :many
SELECT tx_out_id, tx_in_id
FROM transfer_pairs;

-- name: InsertTransferPair :one
INSERT INTO transfer_pairs (
    tx_out_id,
    tx_in_id,
    status,
    confidence,
    rationale
) VALUES (
    $1, $2, $3, $4, $5
)
ON CONFLICT (tx_out_id, tx_in_id) DO NOTHING
RETURNING *;

-- name: ListSuggestedTransferCandidates :many
SELECT
    p.id,
    p.status,
    p.confidence,
    p.created_at,
    out_tx.id AS tx_out_id,
    out_tx.amount AS out_amount,
    out_tx.booking_date AS out_booking_date,
    out_tx.booking_text AS out_booking_text,
    out_tx.purpose AS out_purpose,
    out_tx.counterparty AS out_counterparty,
    COALESCE(out_acc.display_name, '') AS out_account_name,
    in_tx.id AS tx_in_id,
    in_tx.amount AS in_amount,
    in_tx.booking_date AS in_booking_date,
    in_tx.booking_text AS in_booking_text,
    in_tx.purpose AS in_purpose,
    in_tx.counterparty AS in_counterparty,
    COALESCE(in_acc.display_name, '') AS in_account_name
FROM transfer_pairs p
JOIN transactions out_tx ON out_tx.id = p.tx_out_id
JOIN transactions in_tx ON in_tx.id = p.tx_in_id
LEFT JOIN accounts out_acc ON out_acc.id = out_tx.account_id
LEFT JOIN accounts in_acc ON in_acc.id = in_tx.account_id
WHERE p.status = 'suggested'
ORDER BY p.created_at DESC;

-- name: ListTransferPairs :many
SELECT *
FROM transfer_pairs
ORDER BY created_at DESC;

-- name: GetTransferPair :one
SELECT *
FROM transfer_pairs
WHERE id = $1;

-- name: ConfirmTransferPair :one
UPDATE transfer_pairs
SET
    status = 'confirmed',
    confirmed_at = now()
WHERE id = $1
  AND status = 'suggested'
RETURNING *;

-- name: RejectTransferPair :one
UPDATE transfer_pairs
SET
    status = 'rejected',
    confirmed_at = NULL
WHERE id = $1
  AND status = 'suggested'
RETURNING *;
