-- name: CreateAccount :one
INSERT INTO accounts (
    household_id,
    display_name,
    bank,
    currency,
    order_account,
    masked_identifier
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetAccountByID :one
SELECT * FROM accounts
WHERE id = $1 AND household_id = $2;

-- name: GetAccountByOrderAccount :one
SELECT * FROM accounts
WHERE order_account = $1 AND household_id = $2;

-- name: CreateImportRun :one
INSERT INTO import_runs (
    household_id,
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
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16
)
RETURNING *;

-- name: GetImportRun :one
SELECT * FROM import_runs
WHERE id = $1 AND household_id = $2;

-- name: CountTransactionsByImportRun :one
SELECT COUNT(*)::bigint AS count
FROM transactions
WHERE import_run_id = $1 AND household_id = $2;

-- name: DeleteTransactionsByImportRun :execrows
DELETE FROM transactions
WHERE import_run_id = $1 AND household_id = $2;

-- name: MarkImportRunRolledBack :one
UPDATE import_runs
SET
    status = 'rolled_back',
    rolled_back_at = now()
WHERE id = $1
  AND household_id = $2
  AND status = 'committed'
RETURNING *;

-- name: ListImportRuns :many
SELECT *
FROM import_runs
WHERE household_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: UpdateImportRunCounts :one
UPDATE import_runs
SET
    row_inserted = $3,
    row_duplicate = $4,
    committed_at = COALESCE(committed_at, now())
WHERE id = $1
  AND household_id = $2
RETURNING *;
