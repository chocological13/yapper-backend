-- name: CreateMedia :one
INSERT INTO media (
    type,
    url,
    content_id,
    content_type
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetMediaForContent :many
SELECT * FROM media
WHERE content_id = $1 AND content_type = $2
ORDER BY created_at ASC;

-- name: OrphanMedia :exec
UPDATE media
SET content_id = NULL,
    content_type = NULL
WHERE content_id = $1
AND content_type = $2;

-- name: UpdateMediaContent :exec
UPDATE media
SET content_id = $1,
    content_type = $2
WHERE media_id = $3;

-- name: GetOrphanedMedia :many
SELECT * FROM media
WHERE content_id IS NULL
AND created_at < $1;

-- name: DeleteMedia :exec
DELETE FROM media
WHERE media_id = $1;

-- name: GetMediaByID :one
SELECT * FROM media
WHERE media_id = $1;

-- name: GetUnassociatedMediaByID :one
SELECT * FROM media
WHERE media_id = $1
AND content_id IS NULL
AND content_type IS NULL;

