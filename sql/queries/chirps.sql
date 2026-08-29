-- name: CreateChirp :one
INSERT INTO chirps (id, body, user_id, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, now(), now())
RETURNING *;

-- name: GetChirps :many
SELECT * FROM chirps order by created_at asc;

-- name: GetChirp :one
SELECT * FROM chirps where id = $1;
