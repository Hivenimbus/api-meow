-- name: CreateInstance :one
INSERT INTO instances (
    name, webhook_url, tag_id, ignore_groups, receive_messages
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: ListInstances :many
SELECT * FROM instances
ORDER BY created_at DESC;

-- name: GetInstance :one
SELECT * FROM instances
WHERE id = $1 LIMIT 1;

-- name: GetInstanceByName :one
SELECT * FROM instances
WHERE name = $1 LIMIT 1;

-- name: UpdateInstanceStatus :one
UPDATE instances
SET status = $2, phone_number = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateInstanceStatusByName :one
UPDATE instances
SET status = $2, phone_number = $3, updated_at = NOW()
WHERE name = $1
RETURNING *;

-- name: UpdateInstanceSettings :one
UPDATE instances
SET webhook_url = $2, ignore_groups = $3, receive_messages = $4, tag_id = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateInstanceSettingsByName :one
UPDATE instances
SET webhook_url = $2, ignore_groups = $3, receive_messages = $4, tag_id = $5, updated_at = NOW()
WHERE name = $1
RETURNING *;

-- name: DeleteInstance :exec
DELETE FROM instances
WHERE id = $1;

-- name: DeleteInstanceByName :exec
DELETE FROM instances
WHERE name = $1;

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

-- name: GetTag :one
SELECT * FROM tags
WHERE id = $1 LIMIT 1;

-- name: UpdateTag :one
UPDATE tags
SET name = $2, color = $3
WHERE id = $1
RETURNING *;

-- name: DeleteTag :exec
DELETE FROM tags
WHERE id = $1;
