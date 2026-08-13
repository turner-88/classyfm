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

-- name: GetNewsItemBySlug :one
SELECT * FROM news_items WHERE slug = ?;

-- name: CreateHotRelease :execresult
INSERT INTO news_items (source, title, slug, excerpt, content, image_url, thumb_url, middle_images, published_at, is_published, is_featured)
VALUES ('hot_release', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: CreateHotReleaseImported :execresult
-- Same as CreateHotRelease but also records the source article's URL on the old
-- site (classyfm.co.id), used by cmd/importhotrelease to dedupe on re-runs.
INSERT INTO news_items (source, title, slug, excerpt, content, url, image_url, thumb_url, published_at, is_published, is_featured)
VALUES ('hot_release', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateHotRelease :exec
UPDATE news_items
SET title = ?, slug = ?, excerpt = ?, content = ?, image_url = ?, thumb_url = ?, middle_images = ?, published_at = ?, is_published = ?, is_featured = ?
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

-- name: NewsStatsBySource :many
-- One row per news source for the admin dashboard: how many items exist, how many
-- are published (the rest are the newsfeed review queue), how many arrived since
-- `since`, and when the newest published one was dated. latest_published_at is the
-- real staleness signal - feed_sources.last_fetched_at only says the worker ran,
-- not that anything new came back.
--
-- Every CAST here is load-bearing, same family of trap as the scalar subqueries in
-- programs.sql: a bare SUM() is DECIMAL and a bare MAX() of a DATETIME is untyped as
-- far as sqlc is concerned, and both come back as interface{} instead of a number /
-- time.Time. CAST(... AS SIGNED) and CAST(... AS DATETIME) pin them down. The
-- DATETIME cast can't produce a NULL that breaks the time.Time scan: published_at is
-- NOT NULL and GROUP BY never yields an empty group.
SELECT source,
       COUNT(*) AS total,
       CAST(SUM(is_published = 1) AS SIGNED) AS published,
       CAST(SUM(created_at >= sqlc.arg(since)) AS SIGNED) AS recent,
       CAST(MAX(published_at) AS DATETIME) AS latest_published_at
FROM news_items
GROUP BY source;

-- name: ListRecentNewsArrivals :many
-- Raw arrival timestamps for the dashboard's ingest chart, bucketed into days by the
-- caller. Deliberately not a GROUP BY DATE(created_at): that buckets by whatever
-- timezone the MySQL session runs in - the host's - while the chart has to read in the
-- station's (Asia/Jakarta). Grouping in Go against schedule.Loc is correct by
-- construction, and two weeks of two columns is a few hundred rows.
-- hot_release is excluded because those rows arrive in one historical import (a
-- thousand-plus on a single day), which would flatten every real day to nothing.
SELECT source, created_at FROM news_items
WHERE source <> 'hot_release' AND created_at >= sqlc.arg(since)
ORDER BY created_at;
