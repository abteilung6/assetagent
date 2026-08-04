-- name: ListMerchantSourceLabels :many
SELECT DISTINCT
    counterparty,
    purpose
FROM transactions
WHERE household_id = $1
  AND (counterparty <> '' OR purpose <> '')
ORDER BY counterparty, purpose;

-- name: GetMerchantAlias :one
SELECT a.*
FROM merchant_aliases a
JOIN merchants m ON m.id = a.merchant_id
WHERE a.match_type = $1 AND a.pattern = $2 AND m.household_id = $3;

-- name: CreateMerchant :one
INSERT INTO merchants (household_id, display_name, default_category_id)
VALUES ($1, $2, $3)
RETURNING *;

-- name: CreateMerchantAlias :one
INSERT INTO merchant_aliases (merchant_id, match_type, pattern)
VALUES ($1, $2, $3)
RETURNING *;

-- name: ListMerchants :many
SELECT
    m.id,
    m.display_name,
    m.default_category_id,
    m.created_at,
    COUNT(a.id)::bigint AS alias_count
FROM merchants m
LEFT JOIN merchant_aliases a ON a.merchant_id = m.id
WHERE m.household_id = $1
GROUP BY m.id
ORDER BY m.display_name ASC;

-- name: CountMerchantAliases :one
SELECT COUNT(*)::bigint AS count
FROM merchant_aliases a
JOIN merchants m ON m.id = a.merchant_id
WHERE m.household_id = $1;
