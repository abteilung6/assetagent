-- name: CreateAccount :one
INSERT INTO accounts (
    display_name,
    bank,
    currency,
    order_account,
    masked_identifier
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1;

-- name: GetAccountByOrderAccount :one
SELECT * FROM accounts
WHERE order_account = $1;

-- name: CreateImportRun :one
INSERT INTO import_runs (
    account_id,
    source_filename,
    file_hash,
    parser_name,
    parser_version,
    status,
    period_from,
    period_to,
    row_total,
    row_valid,
    row_invalid,
    row_inserted,
    row_duplicate,
    warnings,
    committed_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15
)
RETURNING *;

-- name: GetImportRun :one
SELECT * FROM import_runs
WHERE id = $1;

-- name: CountTransactionsByImportRun :one
SELECT COUNT(*)::bigint AS count
FROM transactions
WHERE import_run_id = $1;

-- name: DeleteTransactionsByImportRun :execrows
DELETE FROM transactions
WHERE import_run_id = $1;

-- name: MarkImportRunRolledBack :one
UPDATE import_runs
SET
    status = 'rolled_back',
    rolled_back_at = now()
WHERE id = $1
  AND status = 'committed'
RETURNING *;

-- name: ListImportRuns :many
SELECT *
FROM import_runs
ORDER BY created_at DESC
LIMIT $1;

-- name: UpdateImportRunCounts :one
UPDATE import_runs
SET
    row_inserted = $2,
    row_duplicate = $3,
    committed_at = COALESCE(committed_at, now())
WHERE id = $1
RETURNING *;
