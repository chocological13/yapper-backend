-- name: CreateYap :one
INSERT INTO yaps (
  user_id,
  content,
  media,
  hashtags,
  mentions,
  location_point
) VALUES (
  $1, $2, $3, $4, $5,
  CASE
    WHEN $6::DOUBLE PRECISION IS NOT NULL AND $7::DOUBLE PRECISION IS NOT NULL
    THEN ST_SetSRID(ST_MakePoint($7::DOUBLE PRECISION, $6::DOUBLE PRECISION), 4326)
    ELSE NULL
  END
)
RETURNING
  yap_id,
  user_id,
  content,
  media,
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
       media,
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
       media,
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
    media = CASE
        WHEN $3::jsonb IS NOT NULL THEN $3::jsonb
        ELSE media
    END,
    hashtags = CASE
        WHEN $4::text[] IS NOT NULL THEN $4::text[]
        ELSE hashtags
    END,
    mentions = CASE
        WHEN $5::text[] IS NOT NULL THEN $5::text[]
        ELSE mentions
    END,
    location_point = CASE
        WHEN $6::DOUBLE PRECISION IS NOT NULL AND $7::DOUBLE PRECISION IS NOT NULL
        AND ($6::DOUBLE PRECISION != 0 OR $7::DOUBLE PRECISION != 0)
        THEN ST_SetSRID(ST_MakePoint($7::DOUBLE PRECISION, $6::DOUBLE PRECISION), 4326)
        WHEN $8::boolean THEN NULL
        ELSE location_point
    END
WHERE yap_id = $1
    AND user_id = $9
    AND deleted_at IS NULL
RETURNING yap_id,
          user_id,
          content,
          media,
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

