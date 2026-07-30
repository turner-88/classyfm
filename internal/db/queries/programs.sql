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
INSERT INTO programs (title, slug, description, image_url, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?);

-- name: UpdateProgram :exec
UPDATE programs
SET title = ?, slug = ?, description = ?, image_url = ?, sort_order = ?, is_active = ?
WHERE id = ?;

-- name: DeleteProgram :exec
DELETE FROM programs WHERE id = ?;

-- broadcaster_name below is the *effective* broadcaster line for a slot, joined as
-- "Anda, Yeni": the slot's own set (schedule_broadcasters) if it has one, otherwise the
-- program's whole default set (program_broadcasters). The NOT EXISTS guard on the second
-- half of the UNION is that set-level override. Every consumer treats this as one
-- display string, so the join happens here rather than in Go.
--
-- It has to be one scalar subquery rather than the more obvious COALESCE of two: sqlc
-- types a lone GROUP_CONCAT subquery as sql.NullString, but wrapping it in COALESCE
-- defeats its inference and the field lands as interface{}.
--
-- has_own_broadcasters lets admin views tell an inherited line from an explicitly
-- assigned one; it replaced the old scalar s.broadcaster_id.

-- name: ListSchedulesForProgram :many
SELECT s.*,
  (SELECT GROUP_CONCAT(b.name ORDER BY b.sort_order, b.name SEPARATOR ', ')
     FROM broadcasters b
    WHERE b.id IN (
      SELECT sb.broadcaster_id FROM schedule_broadcasters sb WHERE sb.schedule_id = s.id
      UNION
      SELECT pb.broadcaster_id FROM program_broadcasters pb
       WHERE pb.program_id = p.id
         AND NOT EXISTS (SELECT 1 FROM schedule_broadcasters x WHERE x.schedule_id = s.id)
    )) AS broadcaster_name,
  EXISTS (SELECT 1 FROM schedule_broadcasters sb WHERE sb.schedule_id = s.id) AS has_own_broadcasters
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
WHERE s.program_id = ?
ORDER BY s.day_of_week ASC, s.start_time ASC;

-- name: ListSchedulesByDay :many
SELECT
  s.id, s.program_id, s.day_of_week, s.start_time, s.end_time,
  (SELECT GROUP_CONCAT(b.name ORDER BY b.sort_order, b.name SEPARATOR ', ')
     FROM broadcasters b
    WHERE b.id IN (
      SELECT sb.broadcaster_id FROM schedule_broadcasters sb WHERE sb.schedule_id = s.id
      UNION
      SELECT pb.broadcaster_id FROM program_broadcasters pb
       WHERE pb.program_id = p.id
         AND NOT EXISTS (SELECT 1 FROM schedule_broadcasters x WHERE x.schedule_id = s.id)
    )) AS broadcaster_name,
  p.title AS program_title, p.slug AS program_slug,
  p.image_url AS program_image_url
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
WHERE s.day_of_week = ? AND p.is_active = 1
ORDER BY s.start_time ASC;

-- name: ListAllSchedulesWithProgram :many
SELECT
  s.id, s.program_id, s.day_of_week, s.start_time, s.end_time,
  (SELECT GROUP_CONCAT(b.name ORDER BY b.sort_order, b.name SEPARATOR ', ')
     FROM broadcasters b
    WHERE b.id IN (
      SELECT sb.broadcaster_id FROM schedule_broadcasters sb WHERE sb.schedule_id = s.id
      UNION
      SELECT pb.broadcaster_id FROM program_broadcasters pb
       WHERE pb.program_id = p.id
         AND NOT EXISTS (SELECT 1 FROM schedule_broadcasters x WHERE x.schedule_id = s.id)
    )) AS broadcaster_name,
  p.title AS program_title, p.slug AS program_slug,
  p.image_url AS program_image_url
FROM program_schedules s
JOIN programs p ON p.id = s.program_id
WHERE p.is_active = 1
ORDER BY s.day_of_week ASC, s.start_time ASC;

-- CreateSchedule is :execresult rather than :exec because the caller needs the new
-- slot's id to write its schedule_broadcasters rows.

-- name: CreateSchedule :execresult
INSERT INTO program_schedules (program_id, day_of_week, start_time, end_time)
VALUES (?, ?, ?, ?);

-- name: UpdateSchedule :exec
UPDATE program_schedules
SET day_of_week = ?, start_time = ?, end_time = ?
WHERE id = ? AND program_id = ?;

-- name: DeleteSchedulesForProgram :exec
DELETE FROM program_schedules WHERE program_id = ?;

-- name: DeleteSchedule :exec
DELETE FROM program_schedules WHERE id = ? AND program_id = ?;

-- Broadcaster assignments. Both sets are written clear-then-insert, so there is no
-- update query; the junction rows go away with their parent via ON DELETE CASCADE.

-- name: ListProgramBroadcasters :many
SELECT b.* FROM program_broadcasters pb
JOIN broadcasters b ON b.id = pb.broadcaster_id
WHERE pb.program_id = ?
ORDER BY b.sort_order ASC, b.name ASC;

-- name: AddProgramBroadcaster :exec
INSERT INTO program_broadcasters (program_id, broadcaster_id) VALUES (?, ?);

-- name: ClearProgramBroadcasters :exec
DELETE FROM program_broadcasters WHERE program_id = ?;

-- name: AddScheduleBroadcaster :exec
INSERT INTO schedule_broadcasters (schedule_id, broadcaster_id) VALUES (?, ?);

-- name: ClearScheduleBroadcasters :exec
DELETE FROM schedule_broadcasters WHERE schedule_id = ?;

-- ListScheduleBroadcasterIDsForProgram returns every slot assignment of one program in
-- a single round trip, so the edit form can mark each slot's <select multiple> without
-- a query per row.

-- name: ListScheduleBroadcasterIDsForProgram :many
SELECT sb.schedule_id, sb.broadcaster_id
FROM schedule_broadcasters sb
JOIN program_schedules s ON s.id = sb.schedule_id
JOIN broadcasters b ON b.id = sb.broadcaster_id
WHERE s.program_id = ?
ORDER BY b.sort_order ASC, b.name ASC;
