-- Home page hero slideshow. The public side reads exactly two of these
-- (settings + active slides) and mixes the result with the latest news; see
-- internal/handlers/public/hero.go. The rest serve /admin/hero.

-- name: GetHeroSettings :one
SELECT * FROM hero_settings WHERE id = 1;

-- name: UpdateHeroSettings :exec
UPDATE hero_settings SET order_mode = ?, news_count = ?, max_images = ? WHERE id = 1;

-- The admin form requires an image, but the image_url <> '' guard keeps a
-- hand-edited or otherwise blank row from reaching the template, where it would
-- emit <img src=""> - which browsers resolve against the page URL and actually
-- fetch.
-- name: ListActiveHeroSlides :many
SELECT * FROM hero_slides
WHERE is_active = 1 AND image_url <> ''
ORDER BY sort_order ASC, id ASC
LIMIT ?;

-- name: ListHeroSlides :many
SELECT * FROM hero_slides
WHERE title LIKE sqlc.arg(search) OR badge_label LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'asc' THEN title END ASC,
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'desc' THEN title END DESC,
  CASE WHEN sqlc.arg(sort) = 'badge_label' AND sqlc.arg(dir) = 'asc' THEN badge_label END ASC,
  CASE WHEN sqlc.arg(sort) = 'badge_label' AND sqlc.arg(dir) = 'desc' THEN badge_label END DESC,
  CASE WHEN sqlc.arg(sort) = 'sort_order' AND sqlc.arg(dir) = 'asc' THEN sort_order END ASC,
  CASE WHEN sqlc.arg(sort) = 'sort_order' AND sqlc.arg(dir) = 'desc' THEN sort_order END DESC,
  CASE WHEN sqlc.arg(sort) = 'updated_at' AND sqlc.arg(dir) = 'asc' THEN updated_at END ASC,
  CASE WHEN sqlc.arg(sort) = 'updated_at' AND sqlc.arg(dir) = 'desc' THEN updated_at END DESC,
  sort_order ASC, id ASC
LIMIT ? OFFSET ?;

-- name: CountHeroSlides :one
SELECT COUNT(*) FROM hero_slides
WHERE title LIKE sqlc.arg(search) OR badge_label LIKE sqlc.arg(search);

-- name: GetHeroSlide :one
SELECT * FROM hero_slides WHERE id = ?;

-- name: CreateHeroSlide :execresult
INSERT INTO hero_slides (title, badge_label, excerpt, image_url, link_url, open_in_new_tab, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateHeroSlide :exec
UPDATE hero_slides
SET title=?, badge_label=?, excerpt=?, image_url=?, link_url=?, open_in_new_tab=?, sort_order=?, is_active=?
WHERE id=?;

-- name: DeleteHeroSlide :exec
DELETE FROM hero_slides WHERE id = ?;
