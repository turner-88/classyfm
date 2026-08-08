-- broadcaster_name is the podcast's broadcaster set joined as "Anda, Yeni". Every
-- consumer treats it as one display string, so the join happens here. It has to be a
-- single scalar GROUP_CONCAT subquery, not a COALESCE of two: sqlc types a lone
-- GROUP_CONCAT subquery as sql.NullString, but wrapping it in COALESCE defeats its
-- inference and the field lands as interface{}. See the same note in programs.sql.

-- name: ListPublishedPodcasts :many
SELECT p.*,
  s.name AS series_name, s.slug AS series_slug,
  (SELECT GROUP_CONCAT(b.name ORDER BY b.sort_order, b.name SEPARATOR ', ')
     FROM broadcasters b
     JOIN podcast_broadcasters pb ON pb.broadcaster_id = b.id
    WHERE pb.podcast_id = p.id) AS broadcaster_name
FROM podcasts p
JOIN podcast_series s ON s.id = p.series_id
WHERE p.is_published = 1
ORDER BY p.created_at DESC, p.id DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedPodcasts :one
SELECT COUNT(*) FROM podcasts WHERE is_published = 1;

-- name: ListPublishedPodcastsBySeriesSlug :many
SELECT p.*,
  s.name AS series_name, s.slug AS series_slug,
  (SELECT GROUP_CONCAT(b.name ORDER BY b.sort_order, b.name SEPARATOR ', ')
     FROM broadcasters b
     JOIN podcast_broadcasters pb ON pb.broadcaster_id = b.id
    WHERE pb.podcast_id = p.id) AS broadcaster_name
FROM podcasts p
JOIN podcast_series s ON s.id = p.series_id
WHERE p.is_published = 1 AND s.slug = ?
ORDER BY p.created_at DESC, p.id DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedPodcastsBySeriesSlug :one
SELECT COUNT(*) FROM podcasts p
JOIN podcast_series s ON s.id = p.series_id
WHERE p.is_published = 1 AND s.slug = ?;

-- name: GetPublishedPodcastBySlug :one
SELECT * FROM podcasts WHERE slug = ? AND is_published = 1;

-- name: ListPodcasts :many
SELECT p.*,
  s.name AS series_name, s.slug AS series_slug,
  (SELECT GROUP_CONCAT(b.name ORDER BY b.sort_order, b.name SEPARATOR ', ')
     FROM broadcasters b
     JOIN podcast_broadcasters pb ON pb.broadcaster_id = b.id
    WHERE pb.podcast_id = p.id) AS broadcaster_name
FROM podcasts p
JOIN podcast_series s ON s.id = p.series_id
WHERE p.title LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'asc' THEN p.title END ASC,
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'desc' THEN p.title END DESC,
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'asc' THEN p.created_at END ASC,
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'desc' THEN p.created_at END DESC,
  p.created_at DESC, p.id DESC
LIMIT ? OFFSET ?;

-- name: CountPodcasts :one
SELECT COUNT(*) FROM podcasts WHERE title LIKE sqlc.arg(search);

-- name: GetPodcast :one
SELECT * FROM podcasts WHERE id = ?;

-- name: GetPodcastBySlug :one
SELECT * FROM podcasts WHERE slug = ?;

-- name: CreatePodcast :execresult
INSERT INTO podcasts (title, slug, series_id, description, spotify_url, thumb_url, is_published)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdatePodcast :exec
UPDATE podcasts
SET title = ?, slug = ?, series_id = ?, description = ?, spotify_url = ?, thumb_url = ?, is_published = ?
WHERE id = ?;

-- name: DeletePodcast :exec
DELETE FROM podcasts WHERE id = ?;

-- name: SetPodcastPublished :exec
UPDATE podcasts SET is_published = ? WHERE id = ?;

-- Broadcaster assignments are written clear-then-insert, so there is no update query;
-- the junction rows go away with their parent via ON DELETE CASCADE.

-- name: ListPodcastBroadcasters :many
SELECT b.* FROM podcast_broadcasters pb
JOIN broadcasters b ON b.id = pb.broadcaster_id
WHERE pb.podcast_id = ?
ORDER BY b.sort_order ASC, b.name ASC;

-- name: AddPodcastBroadcaster :exec
INSERT INTO podcast_broadcasters (podcast_id, broadcaster_id) VALUES (?, ?);

-- name: ClearPodcastBroadcasters :exec
DELETE FROM podcast_broadcasters WHERE podcast_id = ?;
