-- name: GetAboutBanner :one
SELECT * FROM about_page_banner WHERE id = 1;

-- name: UpdateAboutBanner :exec
UPDATE about_page_banner SET media_type = ?, image_url = ?, video_url = ? WHERE id = 1;

-- name: ListAboutSegments :many
SELECT * FROM about_page_segments ORDER BY segment ASC;

-- name: UpdateAboutSegment :exec
UPDATE about_page_segments SET title = ?, body = ? WHERE segment = ?;
