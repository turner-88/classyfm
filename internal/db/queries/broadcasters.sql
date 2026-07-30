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

-- The two queries below answer "who presents this program" / "what does this broadcaster
-- present" for chip lists, and deliberately take the loose reading: a program's default
-- broadcaster counts even if every slot happens to override them. They are display
-- lists, and someone assigned to the program is one of its broadcasters regardless of
-- which slots they actually cover.

-- name: ListProgramsForBroadcaster :many
SELECT p.* FROM programs p
WHERE p.is_active = 1 AND (
  EXISTS (SELECT 1 FROM program_broadcasters pb
           WHERE pb.program_id = p.id AND pb.broadcaster_id = sqlc.arg(broadcaster_id))
  OR EXISTS (SELECT 1 FROM schedule_broadcasters sb
               JOIN program_schedules s ON s.id = sb.schedule_id
              WHERE s.program_id = p.id AND sb.broadcaster_id = sqlc.arg(broadcaster_id))
)
ORDER BY p.sort_order ASC, p.title ASC;

-- name: ListBroadcastersForProgram :many
SELECT b.* FROM broadcasters b
WHERE b.is_active = 1 AND (
  EXISTS (SELECT 1 FROM program_broadcasters pb
           WHERE pb.broadcaster_id = b.id AND pb.program_id = sqlc.arg(program_id))
  OR EXISTS (SELECT 1 FROM schedule_broadcasters sb
               JOIN program_schedules s ON s.id = sb.schedule_id
              WHERE sb.broadcaster_id = b.id AND s.program_id = sqlc.arg(program_id))
)
ORDER BY b.sort_order ASC, b.name ASC;

-- ListBroadcasterProgramLinks, by contrast, must stay exactly effective-per-slot: it
-- feeds the "on air now" badge, which needs a real time slot to measure against. Hence
-- the UNION - slot assignments, plus the program's defaults for the slots that have no
-- assignment of their own.

-- name: ListBroadcasterProgramLinks :many
SELECT DISTINCT sb.broadcaster_id, s.program_id
FROM schedule_broadcasters sb
JOIN program_schedules s ON s.id = sb.schedule_id
JOIN programs p ON p.id = s.program_id
WHERE p.is_active = 1
UNION
SELECT DISTINCT pb.broadcaster_id, s.program_id
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
JOIN program_broadcasters pb ON pb.program_id = s.program_id
WHERE p.is_active = 1
  AND NOT EXISTS (SELECT 1 FROM schedule_broadcasters sb WHERE sb.schedule_id = s.id);
