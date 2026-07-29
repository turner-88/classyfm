-- name: ListActiveBroadcasters :many
SELECT * FROM broadcasters WHERE is_active = 1 ORDER BY sort_order ASC, name ASC;

-- name: ListAllBroadcasters :many
SELECT * FROM broadcasters ORDER BY sort_order ASC, name ASC;

-- name: ListBroadcasters :many
SELECT * FROM broadcasters
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

-- name: CountBroadcasters :one
SELECT COUNT(*) FROM broadcasters
WHERE name LIKE sqlc.arg(search) OR slug LIKE sqlc.arg(search);

-- name: GetBroadcaster :one
SELECT * FROM broadcasters WHERE id = ?;

-- name: GetActiveBroadcasterBySlug :one
SELECT * FROM broadcasters WHERE slug = ? AND is_active = 1;

-- name: CreateBroadcaster :execresult
INSERT INTO broadcasters (name, slug, role, photo_url, bio, birth_place, birth_date, instagram, twitter, facebook, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateBroadcaster :exec
UPDATE broadcasters
SET name=?, slug=?, role=?, photo_url=?, bio=?, birth_place=?, birth_date=?, instagram=?, twitter=?, facebook=?, sort_order=?, is_active=?
WHERE id=?;

-- name: DeleteBroadcaster :exec
DELETE FROM broadcasters WHERE id = ?;

-- name: ListProgramsForBroadcaster :many
SELECT DISTINCT p.* FROM program_schedules ps
JOIN programs p ON p.id = ps.program_id
WHERE ps.broadcaster_id = ? AND p.is_active = 1
ORDER BY p.sort_order ASC, p.title ASC;

-- name: ListBroadcastersForProgram :many
SELECT DISTINCT b.* FROM program_schedules ps
JOIN broadcasters b ON b.id = ps.broadcaster_id
WHERE ps.program_id = ? AND b.is_active = 1
ORDER BY b.sort_order ASC, b.name ASC;

-- name: ListBroadcasterProgramLinks :many
SELECT DISTINCT ps.broadcaster_id, ps.program_id FROM program_schedules ps
JOIN programs p ON p.id = ps.program_id
WHERE ps.broadcaster_id IS NOT NULL AND p.is_active = 1;
