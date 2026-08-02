-- name: ListTransactionsForClassify :many
SELECT
    id,
    counterparty,
    purpose,
    booking_text,
    amount
FROM transactions
ORDER BY booking_date ASC, id ASC;

-- name: ListConfirmedTransferTransactionIDs :many
SELECT tx_out_id AS transaction_id FROM transfer_pairs WHERE status = 'confirmed'
UNION
SELECT tx_in_id AS transaction_id FROM transfer_pairs WHERE status = 'confirmed';

-- name: UpsertTransactionClassification :one
INSERT INTO transaction_classifications (
    transaction_id,
    category_id,
    merchant_id,
    source,
    confidence,
    algorithm_version,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, now()
)
ON CONFLICT (transaction_id) DO UPDATE SET
    category_id = EXCLUDED.category_id,
    merchant_id = EXCLUDED.merchant_id,
    source = EXCLUDED.source,
    confidence = EXCLUDED.confidence,
    algorithm_version = EXCLUDED.algorithm_version,
    updated_at = now()
WHERE transaction_classifications.source <> 'user_rule'
RETURNING *;

-- name: CountClassificationsBySource :many
SELECT source, COUNT(*)::bigint AS count
FROM transaction_classifications
GROUP BY source
ORDER BY source;

-- name: CountClassificationsByCategorySlug :many
SELECT c.slug, COUNT(*)::bigint AS count
FROM transaction_classifications tc
JOIN categories c ON c.id = tc.category_id
GROUP BY c.slug
ORDER BY c.slug;
