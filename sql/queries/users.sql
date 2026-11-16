-- name: CreateUser :one
INSERT INTO users (id, email, hashed_password, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, now(), now())
RETURNING *;

-- name: DeleteUsers :exec
DELETE from users;

-- name: GetUserPassword :one
SELECT * from users where email = $1;
