-- name: GetUser :one
SELECT * FROM users
WHERE user_id = $1 OR username = $2 OR email = $3
AND deleted_at IS NULL;

-- name: NewUser :one
insert into users (username, email, password) values ($1, $2, $3) returning email;

-- name: UpdateUser :one
UPDATE users
SET username = COALESCE($2, username),
    -- Add other non-sensitive fields here as needed if the users table grows
    updated_at = now()
WHERE user_id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: UpdatePassword :exec
UPDATE users
SET password = $2,
    updated_at = NOW()
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: UpdateEmail :one
UPDATE users
SET email = $2,
    updated_at = now()
WHERE user_id = $1 AND deleted_at IS NULL
RETURNING *;

-- name: DeleteUser :exec
UPDATE users
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE user_id = $1
  AND deleted_at IS NULL;