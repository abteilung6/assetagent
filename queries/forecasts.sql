-- name: InsertForecast :one
INSERT INTO forecasts (
    baseline_id,
    horizon_days,
    starting_balance,
    assumptions,
    series,
    min_balance,
    ending_balance,
    algorithm_version
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetForecast :one
SELECT *
FROM forecasts
WHERE id = $1;

-- name: GetLatestForecastForBaseline :one
SELECT *
FROM forecasts
WHERE baseline_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: InsertScenario :one
INSERT INTO scenarios (
    forecast_id,
    kind,
    params,
    result,
    status
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListScenariosForForecast :many
SELECT *
FROM scenarios
WHERE forecast_id = $1
ORDER BY created_at DESC;

-- name: GetScenario :one
SELECT *
FROM scenarios
WHERE id = $1;
