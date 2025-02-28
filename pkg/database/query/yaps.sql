-- name: CreateYap :one
INSERT INTO yaps (
  user_id,
  content,
  hashtags,
  mentions,
  location_point
) VALUES (
  $1, $2, $3, $4,
  CASE
    WHEN $5::DOUBLE PRECISION IS NOT NULL AND $6::DOUBLE PRECISION IS NOT NULL
    THEN ST_SetSRID(ST_MakePoint($6::DOUBLE PRECISION, $5::DOUBLE PRECISION), 4326)
    ELSE NULL
  END
)
RETURNING
  yap_id,
  user_id,
  content,
  hashtags,
  mentions,
  ST_X(location_point::geometry) as longitude,
  ST_Y(location_point::geometry) as latitude,
  created_at,
  updated_at;

-- name: GetYapByID :one
SELECT yap_id,
       user_id,
       content,
       hashtags,
       mentions,
       ST_X(location_point::geometry) as longitude,
       ST_Y(location_point::geometry) as latitude,
       created_at,
       updated_at
FROM yaps
WHERE yap_id = $1 AND deleted_at IS NULL;

-- name: ListYapsByUser :many
SELECT yap_id,
       user_id,
       content,
       hashtags,
       mentions,
       ST_X(location_point::geometry) as longitude,
       ST_Y(location_point::geometry) as latitude,
       created_at,
       updated_at
FROM yaps
WHERE user_id = $1 AND deleted_at IS NULL
ORDER BY created_at DESC
LIMIT COALESCE($2, 20) OFFSET COALESCE($3, 0);

-- name: UpdateYap :one
UPDATE yaps
SET
    content = CASE
        WHEN $2::text IS NOT NULL THEN $2::text
        ELSE content
    END,
    hashtags = CASE
        WHEN $3::text[] IS NOT NULL THEN $3::text[]
        ELSE hashtags
    END,
    mentions = CASE
        WHEN $4::text[] IS NOT NULL THEN $4::text[]
        ELSE mentions
    END,
    location_point = CASE
        WHEN $5::DOUBLE PRECISION IS NOT NULL AND $6::DOUBLE PRECISION IS NOT NULL
        AND ($5::DOUBLE PRECISION != 0 OR $6::DOUBLE PRECISION != 0)
        THEN ST_SetSRID(ST_MakePoint($6::DOUBLE PRECISION, $5::DOUBLE PRECISION), 4326)
        WHEN $7::boolean THEN NULL
        ELSE location_point
    END
WHERE yap_id = $1
    AND user_id = $8
    AND deleted_at IS NULL
RETURNING yap_id,
          user_id,
          content,
          hashtags,
          mentions,
          ST_X(location_point::geometry) as longitude,
          ST_Y(location_point::geometry) as latitude,
          created_at,
          updated_at;

-- name: DeleteYap :exec
UPDATE yaps
SET deleted_at = NOW()
WHERE yap_id = $1
AND user_id = $2
AND deleted_at IS NULL;

