-- name: ListMerchantSourceLabels :many
SELECT DISTINCT
    counterparty,
    purpose
FROM transactions
WHERE counterparty <> '' OR purpose <> ''
ORDER BY counterparty, purpose;

-- name: GetMerchantAlias :one
SELECT *
FROM merchant_aliases
WHERE match_type = $1 AND pattern = $2;

-- name: CreateMerchant :one
INSERT INTO merchants (display_name, default_category_id)
VALUES ($1, $2)
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
GROUP BY m.id
ORDER BY m.display_name ASC;

-- name: CountMerchantAliases :one
SELECT COUNT(*)::bigint AS count FROM merchant_aliases;
