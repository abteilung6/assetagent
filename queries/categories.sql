-- name: ListCategories :many
SELECT *
FROM categories
ORDER BY slug ASC;

-- name: GetCategoryBySlug :one
SELECT *
FROM categories
WHERE slug = $1;
