-- name: ListActiveClassiers :many
SELECT * FROM classiers WHERE is_active = 1 ORDER BY sort_order ASC, name ASC;

-- name: ListClassiers :many
SELECT * FROM classiers
WHERE name LIKE sqlc.arg(search) OR slug LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'name' AND sqlc.arg(dir) = 'asc' THEN name END ASC,
  CASE WHEN sqlc.arg(sort) = 'name' AND sqlc.arg(dir) = 'desc' THEN name END DESC,
  CASE WHEN sqlc.arg(sort) = 'slug' AND sqlc.arg(dir) = 'asc' THEN slug END ASC,
  CASE WHEN sqlc.arg(sort) = 'slug' AND sqlc.arg(dir) = 'desc' THEN slug END DESC,
  CASE WHEN sqlc.arg(sort) = 'role' AND sqlc.arg(dir) = 'asc' THEN role END ASC,
  CASE WHEN sqlc.arg(sort) = 'role' AND sqlc.arg(dir) = 'desc' THEN role END DESC,
  CASE WHEN sqlc.arg(sort) = 'sort_order' AND sqlc.arg(dir) = 'asc' THEN sort_order END ASC,
  CASE WHEN sqlc.arg(sort) = 'sort_order' AND sqlc.arg(dir) = 'desc' THEN sort_order END DESC,
  sort_order ASC, name ASC, id ASC
LIMIT ? OFFSET ?;

-- name: CountClassiers :one
SELECT COUNT(*) FROM classiers
WHERE name LIKE sqlc.arg(search) OR slug LIKE sqlc.arg(search);

-- name: GetClassier :one
SELECT * FROM classiers WHERE id = ?;

-- name: GetActiveClassierBySlug :one
SELECT * FROM classiers WHERE slug = ? AND is_active = 1;

-- name: CreateClassier :execresult
INSERT INTO classiers (name, slug, role, photo_url, bio, birth_place, birth_date, instagram, twitter, facebook, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateClassier :exec
UPDATE classiers
SET name=?, slug=?, role=?, photo_url=?, bio=?, birth_place=?, birth_date=?, instagram=?, twitter=?, facebook=?, sort_order=?, is_active=?
WHERE id=?;

-- name: DeleteClassier :exec
DELETE FROM classiers WHERE id = ?;
