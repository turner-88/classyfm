-- name: ListHotRelease :many
SELECT * FROM news_items
WHERE source = 'hot_release' AND is_published = 1
ORDER BY published_at DESC
LIMIT ?;

-- name: ListLatestPublished :many
SELECT * FROM news_items
WHERE is_published = 1
ORDER BY published_at DESC
LIMIT ?;

-- name: ListAllHotRelease :many
SELECT * FROM news_items
WHERE source = 'hot_release'
ORDER BY published_at DESC;

-- name: GetNewsItem :one
SELECT * FROM news_items WHERE id = ?;

-- name: GetPublishedNewsItemBySlug :one
SELECT * FROM news_items WHERE slug = ? AND is_published = 1;

-- name: CreateHotRelease :execresult
INSERT INTO news_items (source, title, slug, excerpt, content, image_url, published_at, is_published, is_featured)
VALUES ('hot_release', ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateHotRelease :exec
UPDATE news_items
SET title = ?, slug = ?, excerpt = ?, content = ?, image_url = ?, published_at = ?, is_published = ?, is_featured = ?
WHERE id = ? AND source = 'hot_release';

-- name: DeleteNewsItem :exec
DELETE FROM news_items WHERE id = ?;

-- name: UpsertNewsItem :exec
-- Inserts a new aggregated item as published, or refreshes content fields on an
-- existing one. is_published/is_featured are intentionally left untouched on
-- conflict so an admin's publish/feature decision survives the next fetch.
INSERT INTO news_items (source, external_id, title, excerpt, url, image_url, published_at, is_published)
VALUES (?, ?, ?, ?, ?, ?, ?, 1)
ON DUPLICATE KEY UPDATE
  title = VALUES(title),
  excerpt = VALUES(excerpt),
  url = VALUES(url),
  image_url = VALUES(image_url),
  published_at = VALUES(published_at);

-- name: ListAggregatedNews :many
SELECT * FROM news_items
WHERE source != 'hot_release'
ORDER BY published_at DESC
LIMIT 200;

-- name: SetNewsItemPublished :exec
UPDATE news_items SET is_published = ? WHERE id = ?;

-- name: SetNewsItemFeatured :exec
UPDATE news_items SET is_featured = ? WHERE id = ?;

-- name: ListPublishedNews :many
SELECT * FROM news_items
WHERE is_published = 1
ORDER BY published_at DESC
LIMIT ? OFFSET ?;

-- name: ListPublishedNewsBySource :many
SELECT * FROM news_items
WHERE is_published = 1 AND source = ?
ORDER BY published_at DESC
LIMIT ? OFFSET ?;

-- name: CountPublishedNews :one
SELECT COUNT(*) FROM news_items WHERE is_published = 1;

-- name: CountPublishedNewsBySource :one
SELECT COUNT(*) FROM news_items WHERE is_published = 1 AND source = ?;
