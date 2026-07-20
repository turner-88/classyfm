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
WHERE source = 'hot_release' AND title LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'asc' THEN title END ASC,
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'desc' THEN title END DESC,
  CASE WHEN sqlc.arg(sort) = 'published_at' AND sqlc.arg(dir) = 'asc' THEN published_at END ASC,
  CASE WHEN sqlc.arg(sort) = 'published_at' AND sqlc.arg(dir) = 'desc' THEN published_at END DESC,
  published_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountAllHotRelease :one
SELECT COUNT(*) FROM news_items WHERE source = 'hot_release' AND title LIKE sqlc.arg(search);

-- name: GetNewsItem :one
SELECT * FROM news_items WHERE id = ?;

-- name: GetPublishedNewsItemBySlug :one
SELECT * FROM news_items WHERE slug = ? AND is_published = 1;

-- name: CreateHotRelease :execresult
INSERT INTO news_items (source, title, slug, excerpt, content, image_url, thumb_url, published_at, is_published, is_featured)
VALUES ('hot_release', ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateHotReleaseImported :execresult
-- Same as CreateHotRelease but also records the source article's URL on the old
-- site (classyfm.co.id), used by cmd/importhotrelease to dedupe on re-runs.
INSERT INTO news_items (source, title, slug, excerpt, content, url, image_url, thumb_url, published_at, is_published, is_featured)
VALUES ('hot_release', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateHotRelease :exec
UPDATE news_items
SET title = ?, slug = ?, excerpt = ?, content = ?, image_url = ?, thumb_url = ?, published_at = ?, is_published = ?, is_featured = ?
WHERE id = ? AND source = 'hot_release';

-- name: GetNewsItemImages :one
-- Used by the feed worker to check what's already stored before overwriting
-- image_url/thumb_url on a refresh, so a transient resolution failure can't
-- downgrade an already-upgraded image (see feeds.PreferImage).
SELECT image_url, thumb_url FROM news_items WHERE source = ? AND external_id = ?;

-- name: UpdateNewsItemImages :exec
-- Used by one-off backfill tools (e.g. cmd/upgradeimages) to swap in a
-- higher-resolution image (and/or its list-sized thumbnail) for an
-- already-aggregated item without touching anything else about the row.
UPDATE news_items SET image_url = ?, thumb_url = ? WHERE id = ?;

-- name: DeleteNewsItem :exec
DELETE FROM news_items WHERE id = ?;

-- name: UpsertNewsItem :exec
-- Inserts a new aggregated item as published, or refreshes content fields on an
-- existing one. is_published/is_featured are intentionally left untouched on
-- conflict so an admin's publish/feature decision survives the next fetch.
INSERT INTO news_items (source, external_id, title, excerpt, url, image_url, thumb_url, published_at, is_published)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
ON DUPLICATE KEY UPDATE
  title = VALUES(title),
  excerpt = VALUES(excerpt),
  url = VALUES(url),
  image_url = VALUES(image_url),
  thumb_url = VALUES(thumb_url),
  published_at = VALUES(published_at);

-- name: ListAggregatedNews :many
SELECT * FROM news_items
WHERE source != 'hot_release'
  AND (sqlc.arg(source) = '' OR source = sqlc.arg(source))
  AND title LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'asc' THEN title END ASC,
  CASE WHEN sqlc.arg(sort) = 'title' AND sqlc.arg(dir) = 'desc' THEN title END DESC,
  CASE WHEN sqlc.arg(sort) = 'source' AND sqlc.arg(dir) = 'asc' THEN source END ASC,
  CASE WHEN sqlc.arg(sort) = 'source' AND sqlc.arg(dir) = 'desc' THEN source END DESC,
  CASE WHEN sqlc.arg(sort) = 'published_at' AND sqlc.arg(dir) = 'asc' THEN published_at END ASC,
  CASE WHEN sqlc.arg(sort) = 'published_at' AND sqlc.arg(dir) = 'desc' THEN published_at END DESC,
  published_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountAggregatedNews :one
SELECT COUNT(*) FROM news_items
WHERE source != 'hot_release'
  AND (sqlc.arg(source) = '' OR source = sqlc.arg(source))
  AND title LIKE sqlc.arg(search);

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
