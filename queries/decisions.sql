-- name: InsertDecision :one
INSERT INTO decisions (
    review_id,
    scenario_id,
    title,
    assumptions,
    target_value
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetDecision :one
SELECT *
FROM decisions
WHERE id = $1;

-- name: ListDecisions :many
SELECT *
FROM decisions
ORDER BY decided_at DESC
LIMIT sqlc.arg('row_limit');

-- name: InsertAction :one
INSERT INTO actions (
    decision_id,
    title,
    expected_annual_effect,
    due_on,
    status,
    outcome_note
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetAction :one
SELECT *
FROM actions
WHERE id = $1;

-- name: ListActions :many
SELECT *
FROM actions
WHERE (sqlc.narg('status')::text IS NULL OR status = sqlc.narg('status'))
ORDER BY due_on ASC, created_at DESC
LIMIT sqlc.arg('row_limit');

-- name: ListActionsForDecision :many
SELECT *
FROM actions
WHERE decision_id = $1
ORDER BY created_at ASC;

-- name: UpdateActionStatus :one
UPDATE actions
SET
    status = $2,
    outcome_note = $3,
    verified_at = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;
