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

-- name: GetClassificationRuleByPattern :one
SELECT *
FROM classification_rules
WHERE merchant_id IS NULL
  AND lower(pattern) = lower(sqlc.arg(pattern));

-- name: CreateClassificationRule :one
INSERT INTO classification_rules (
    priority,
    merchant_id,
    pattern,
    category_id,
    created_from_transaction_id,
    confidence,
    is_system
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpsertSystemPatternRule :one
INSERT INTO classification_rules (
    priority,
    merchant_id,
    pattern,
    category_id,
    created_from_transaction_id,
    confidence,
    is_system
) VALUES (
    sqlc.arg(priority),
    NULL,
    sqlc.arg(pattern),
    sqlc.arg(category_id),
    NULL,
    sqlc.arg(confidence),
    true
)
ON CONFLICT ((lower(pattern))) WHERE merchant_id IS NULL AND pattern IS NOT NULL
DO UPDATE SET
    priority = EXCLUDED.priority,
    category_id = EXCLUDED.category_id,
    confidence = EXCLUDED.confidence,
    is_system = true
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

-- name: ListClassificationQueue :many
SELECT
    t.id AS transaction_id,
    t.booking_date,
    t.amount,
    t.counterparty,
    t.purpose,
    t.booking_text,
    c.slug AS category_slug,
    c.display_name AS category_name,
    tc.source,
    tc.confidence,
    m.id AS merchant_id,
    COALESCE(m.display_name, '') AS merchant_name
FROM transaction_classifications tc
JOIN transactions t ON t.id = tc.transaction_id
JOIN categories c ON c.id = tc.category_id
LEFT JOIN merchants m ON m.id = tc.merchant_id
WHERE tc.source <> 'user_rule'
  AND c.slug <> 'transfer'
  AND t.one_off = false
  AND (
    tc.source = 'unresolved'
    OR tc.confidence = 'low'
  )
ORDER BY ABS(t.amount) DESC, t.booking_date DESC, t.id ASC
LIMIT 50;
