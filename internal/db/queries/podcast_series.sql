-- name: ListPodcastSeries :many
SELECT * FROM podcast_series
ORDER BY sort_order ASC, name ASC;

-- name: ListActivePodcastSeries :many
SELECT * FROM podcast_series
WHERE is_active = 1
ORDER BY sort_order ASC, name ASC;

-- name: GetPodcastSeries :one
SELECT * FROM podcast_series WHERE id = ?;

-- name: GetPodcastSeriesBySlug :one
SELECT * FROM podcast_series WHERE slug = ?;

-- name: CreatePodcastSeries :execresult
INSERT INTO podcast_series (name, slug, sort_order, is_active) VALUES (?, ?, ?, ?);

-- name: UpdatePodcastSeries :exec
UPDATE podcast_series SET name = ?, slug = ?, sort_order = ?, is_active = ? WHERE id = ?;

-- name: DeletePodcastSeries :exec
DELETE FROM podcast_series WHERE id = ?;
