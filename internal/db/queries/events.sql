-- Events are admin-authored (title, description, and uploaded banner are all real form
-- input), so unlike podcasts.sql there is no GROUP_CONCAT broadcaster subquery and no
-- importer/refresh queries. The public list is newest-first, optionally filtered to one
-- category; the admin list adds search + dynamic sort. See 0041_events.up.sql.

-- name: ListPublishedEvents :many
SELECT * FROM events
WHERE is_published = 1
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedEvents :one
SELECT COUNT(*) FROM events WHERE is_published = 1;

-- name: ListPublishedEventsByCategory :many
SELECT * FROM events
WHERE is_published = 1 AND category = ?
ORDER BY created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedEventsByCategory :one
SELECT COUNT(*) FROM events WHERE is_published = 1 AND category = ?;

-- name: GetPublishedEventBySlug :one
SELECT * FROM events WHERE slug = ? AND is_published = 1;

-- name: ListEvents :many
SELECT * FROM events
WHERE title LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'asc' THEN title END ASC,
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'desc' THEN title END DESC,
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'asc' THEN created_at END ASC,
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'desc' THEN created_at END DESC,
  created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountEvents :one
SELECT COUNT(*) FROM events WHERE title LIKE sqlc.arg(search);

-- name: GetEvent :one
SELECT * FROM events WHERE id = ?;

-- name: GetEventBySlug :one
SELECT * FROM events WHERE slug = ?;

-- name: CreateEvent :execresult
INSERT INTO events (title, slug, category, description, image_url, event_date, location, link_url, is_published)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateEvent :exec
UPDATE events
SET title = ?, slug = ?, category = ?, description = ?, image_url = ?, event_date = ?, location = ?, link_url = ?, is_published = ?
WHERE id = ?;

-- name: DeleteEvent :exec
DELETE FROM events WHERE id = ?;

-- name: SetEventPublished :exec
UPDATE events SET is_published = ? WHERE id = ?;
