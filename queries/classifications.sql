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

-- name: ForceUpsertTransactionClassification :one
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
RETURNING *;

-- name: GetTransactionForClassify :one
SELECT
    id,
    counterparty,
    purpose,
    booking_text,
    amount
FROM transactions
WHERE id = $1;

-- name: GetTransactionClassification :one
SELECT *
FROM transaction_classifications
WHERE transaction_id = $1;

-- name: ListClassificationRules :many
SELECT *
FROM classification_rules
ORDER BY priority ASC, created_at ASC;

-- name: GetClassificationRuleByMerchant :one
SELECT *
FROM classification_rules
WHERE merchant_id = $1;

-- name: CreateClassificationRule :one
INSERT INTO classification_rules (
    priority,
    merchant_id,
    category_id,
    created_from_transaction_id
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateClassificationRuleCategory :one
UPDATE classification_rules
SET
    category_id = $2,
    created_from_transaction_id = $3,
    priority = $4
WHERE id = $1
RETURNING *;

-- name: UpdateMerchantDefaultCategory :one
UPDATE merchants
SET default_category_id = $2
WHERE id = $1
RETURNING *;

