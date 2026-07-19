-- name: ListPrograms :many
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
INSERT INTO programs (title, slug, description, host, image_url, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateProgram :exec
UPDATE programs
SET title = ?, slug = ?, description = ?, host = ?, image_url = ?, sort_order = ?, is_active = ?
WHERE id = ?;

-- name: DeleteProgram :exec
DELETE FROM programs WHERE id = ?;

-- name: ListSchedulesForProgram :many
SELECT * FROM program_schedules
WHERE program_id = ?
ORDER BY day_of_week ASC, start_time ASC;

-- name: ListSchedulesByDay :many
SELECT
  s.id, s.program_id, s.day_of_week, s.start_time, s.end_time, s.host AS slot_host,
  p.title AS program_title, p.slug AS program_slug, p.host AS program_host,
  p.image_url AS program_image_url
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
WHERE s.day_of_week = ? AND p.is_active = 1
ORDER BY s.start_time ASC;

-- name: ListAllSchedulesWithProgram :many
SELECT
  s.id, s.program_id, s.day_of_week, s.start_time, s.end_time, s.host AS slot_host,
  p.title AS program_title, p.slug AS program_slug, p.host AS program_host
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
WHERE p.is_active = 1
ORDER BY s.day_of_week ASC, s.start_time ASC;

-- name: CreateSchedule :exec
INSERT INTO program_schedules (program_id, day_of_week, start_time, end_time, host)
VALUES (?, ?, ?, ?, ?);

-- name: DeleteSchedulesForProgram :exec
DELETE FROM program_schedules WHERE program_id = ?;

-- name: DeleteSchedule :exec
DELETE FROM program_schedules WHERE id = ? AND program_id = ?;
