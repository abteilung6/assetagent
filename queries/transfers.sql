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
