-- name: ListTransactionsForClassify :many
SELECT
    id,
    counterparty,
    purpose,
    booking_text,
    amount
FROM transactions
WHERE household_id = $1
ORDER BY booking_date ASC, id ASC;

-- name: ListConfirmedTransferTransactionIDs :many
SELECT p.tx_out_id AS transaction_id
FROM transfer_pairs p
WHERE p.status = 'confirmed' AND p.household_id = $1
UNION
SELECT p.tx_in_id AS transaction_id
FROM transfer_pairs p
WHERE p.status = 'confirmed' AND p.household_id = $1;

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
SELECT tc.source, COUNT(*)::bigint AS count
FROM transaction_classifications tc
JOIN transactions t ON t.id = tc.transaction_id
WHERE t.household_id = $1
GROUP BY tc.source
ORDER BY tc.source;

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
WHERE id = $1 AND household_id = $2;

-- name: GetTransactionClassification :one
SELECT tc.*
FROM transaction_classifications tc
JOIN transactions t ON t.id = tc.transaction_id
WHERE tc.transaction_id = $1 AND t.household_id = $2;

-- name: ListClassificationRules :many
SELECT *
FROM classification_rules
WHERE household_id = $1
ORDER BY priority ASC, created_at ASC;

-- name: GetClassificationRuleByMerchant :one
SELECT *
FROM classification_rules
WHERE merchant_id = $1 AND household_id = $2;

-- name: GetClassificationRuleByPattern :one
SELECT *
FROM classification_rules
WHERE merchant_id IS NULL
  AND lower(pattern) = lower(sqlc.arg(pattern))
  AND household_id = sqlc.arg(household_id);

-- name: CreateClassificationRule :one
INSERT INTO classification_rules (
    household_id,
    priority,
    merchant_id,
    pattern,
    category_id,
    created_from_transaction_id,
    confidence,
    is_system
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpsertSystemPatternRule :one
INSERT INTO classification_rules (
    household_id,
    priority,
    merchant_id,
    pattern,
    category_id,
    created_from_transaction_id,
    confidence,
    is_system
) VALUES (
    sqlc.arg(household_id),
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
    category_id = $3,
    created_from_transaction_id = $4,
    priority = $5
WHERE id = $1 AND household_id = $2
RETURNING *;

-- name: UpdateMerchantDefaultCategory :one
UPDATE merchants
SET default_category_id = $3
WHERE id = $1 AND household_id = $2
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
WHERE t.household_id = $1
  AND tc.source <> 'user_rule'
  AND c.slug <> 'transfer'
  AND t.one_off = false
  AND (
    tc.source = 'unresolved'
    OR tc.confidence = 'low'
  )
ORDER BY ABS(t.amount) DESC, t.booking_date DESC, t.id ASC
LIMIT 100;
