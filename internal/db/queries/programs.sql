-- name: ListPrograms :many
SELECT * FROM programs
WHERE title LIKE sqlc.arg(search) OR slug LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'asc' THEN title END ASC,
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'desc' THEN title END DESC,
  CASE WHEN sqlc.arg(sort) = 'slug' AND sqlc.arg(dir) = 'asc' THEN slug END ASC,
  CASE WHEN sqlc.arg(sort) = 'slug' AND sqlc.arg(dir) = 'desc' THEN slug END DESC,
  CASE WHEN sqlc.arg(sort) = 'sort_order' AND sqlc.arg(dir) = 'asc' THEN sort_order END ASC,
  CASE WHEN sqlc.arg(sort) = 'sort_order' AND sqlc.arg(dir) = 'desc' THEN sort_order END DESC,
  sort_order ASC, title ASC, id ASC
LIMIT ? OFFSET ?;

-- name: CountPrograms :one
SELECT COUNT(*) FROM programs
WHERE title LIKE sqlc.arg(search) OR slug LIKE sqlc.arg(search);

-- name: ListAllPrograms :many
SELECT * FROM programs
ORDER BY sort_order ASC, title ASC;

-- name: ListActivePrograms :many
SELECT * FROM programs
WHERE is_active = 1
ORDER BY sort_order ASC, title ASC;

-- name: GetProgram :one
SELECT * FROM programs WHERE id = ?;

-- name: GetProgramBySlug :one
SELECT * FROM programs WHERE slug = ?;

-- name: CreateProgram :execresult
INSERT INTO programs (title, slug, description, image_url, broadcaster_id, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProgram :exec
UPDATE programs
SET title = ?, slug = ?, description = ?, image_url = ?, broadcaster_id = ?, sort_order = ?, is_active = ?
WHERE id = ?;

-- name: DeleteProgram :exec
DELETE FROM programs WHERE id = ?;

-- broadcaster_name below is the *effective* broadcaster: the slot's own if set,
-- otherwise the program's default (programs.broadcaster_id). The selected
-- s.broadcaster_id stays the slot's own value so admin views can tell an
-- inherited broadcaster from an explicitly assigned one.

-- name: ListSchedulesForProgram :many
SELECT s.*, b.name AS broadcaster_name
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
LEFT JOIN broadcasters b ON b.id = COALESCE(s.broadcaster_id, p.broadcaster_id)
WHERE s.program_id = ?
ORDER BY s.day_of_week ASC, s.start_time ASC;

-- name: ListSchedulesByDay :many
SELECT
  s.id, s.program_id, s.day_of_week, s.start_time, s.end_time, s.broadcaster_id,
  b.name AS broadcaster_name,
  p.title AS program_title, p.slug AS program_slug,
  p.image_url AS program_image_url
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
LEFT JOIN broadcasters b ON b.id = COALESCE(s.broadcaster_id, p.broadcaster_id)
WHERE s.day_of_week = ? AND p.is_active = 1
ORDER BY s.start_time ASC;

-- name: ListAllSchedulesWithProgram :many
SELECT
  s.id, s.program_id, s.day_of_week, s.start_time, s.end_time, s.broadcaster_id,
  b.name AS broadcaster_name,
  p.title AS program_title, p.slug AS program_slug,
  p.image_url AS program_image_url
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
LEFT JOIN broadcasters b ON b.id = COALESCE(s.broadcaster_id, p.broadcaster_id)
WHERE p.is_active = 1
ORDER BY s.day_of_week ASC, s.start_time ASC;

-- name: CreateSchedule :exec
INSERT INTO program_schedules (program_id, day_of_week, start_time, end_time, broadcaster_id)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteSchedulesForProgram :exec
DELETE FROM program_schedules WHERE program_id = ?;

-- name: DeleteSchedule :exec
DELETE FROM program_schedules WHERE id = ? AND program_id = ?;
