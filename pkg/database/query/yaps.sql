-- name: CreateYap :one
INSERT INTO yaps (
  user_id,
  content
) VALUES ($1, $2)
RETURNING *;

-- name: GetYapByID :one
SELECT *
FROM yaps
WHERE yap_id = $1 AND deleted_at IS NULL;

-- name: ListYapsByUserID :many
SELECT *
FROM yaps
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT COALESCE($2, 10) OFFSET COALESCE($3, 0);

-- name: UpdateYap :one
UPDATE yaps
SET content = $2
WHERE yap_id = $1
AND user_id = $3
AND deleted_at IS NULL
RETURNING *;

-- name: DeleteYap :exec
UPDATE yaps
SET deleted_at = NOW()
WHERE yap_id = $1
AND user_id = $2
AND deleted_at IS NULL;

