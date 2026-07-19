-- name: ListActiveClassiers :many
SELECT * FROM classiers WHERE is_active = 1 ORDER BY sort_order ASC, name ASC;

-- name: ListClassiers :many
SELECT * FROM classiers ORDER BY sort_order ASC, name ASC;

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
