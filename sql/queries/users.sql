-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, password)
VALUES (gen_random_uuid(), now(), now(), $1, $2)
RETURNING id, created_at, updated_at, email, is_chirpy_red;

-- name: ClearUsers :exec
DELETE FROM users;

-- name: GetUserByEmail :one
SELECT id, created_at, updated_at, email, is_chirpy_red FROM users
WHERE email = $1;

-- name: GetUserPassword :one
SELECT password FROM users
WHERE email = $1;

-- name: UpdateUser :one
UPDATE users
SET email = $2, password = $3, updated_at = now()
WHERE id = $1
RETURNING id, created_at, updated_at, email, is_chirpy_red;

-- name: UpdateToChirpyRed :exec
UPDATE users
SET is_chirpy_red = true
WHERE id = $1;