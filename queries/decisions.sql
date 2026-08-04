-- name: InsertDecision :one
INSERT INTO decisions (
    household_id,
    review_id,
    scenario_id,
    title,
    assumptions,
    target_value
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: GetDecision :one
SELECT *
FROM decisions
WHERE id = $1 AND household_id = $2;

-- name: ListDecisions :many
SELECT *
FROM decisions
WHERE household_id = sqlc.arg('household_id')
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
SELECT a.*
FROM actions a
JOIN decisions d ON d.id = a.decision_id
WHERE a.id = $1 AND d.household_id = $2;

-- name: ListActions :many
SELECT a.*
FROM actions a
JOIN decisions d ON d.id = a.decision_id
WHERE d.household_id = sqlc.arg('household_id')
  AND (sqlc.narg('status')::text IS NULL OR a.status = sqlc.narg('status'))
ORDER BY a.due_on ASC, a.created_at DESC
LIMIT sqlc.arg('row_limit');

-- name: ListActionsForDecision :many
SELECT a.*
FROM actions a
JOIN decisions d ON d.id = a.decision_id
WHERE a.decision_id = $1 AND d.household_id = $2
ORDER BY a.created_at ASC;

-- name: UpdateActionStatus :one
UPDATE actions a
SET
    status = $3,
    outcome_note = $4,
    verified_at = $5,
    updated_at = now()
FROM decisions d
WHERE a.id = $1
  AND d.id = a.decision_id
  AND d.household_id = $2
RETURNING a.*;
