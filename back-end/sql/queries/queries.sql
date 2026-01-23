-- name: CreateInstance :one
INSERT INTO instances (
    name, webhook_url
) VALUES (
    $1, $2
)
RETURNING *;

-- name: ListInstances :many
SELECT * FROM instances
ORDER BY created_at DESC;

-- name: GetInstance :one
SELECT * FROM instances
WHERE id = $1 LIMIT 1;

-- name: UpdateInstanceStatus :one
UPDATE instances
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteInstance :exec
DELETE FROM instances
WHERE id = $1;

-- name: CreateTag :one
INSERT INTO tags (
    name, color
) VALUES (
    $1, $2
)
RETURNING *;

-- name: ListTags :many
SELECT * FROM tags
ORDER BY name;
