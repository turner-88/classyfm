-- name: ListFeedSources :many
SELECT * FROM feed_sources ORDER BY source ASC;

-- name: GetFeedSource :one
SELECT * FROM feed_sources WHERE source = ?;

-- name: UpdateFeedSourceConfig :exec
UPDATE feed_sources SET is_enabled = ?, endpoint = ? WHERE source = ?;

-- name: UpdateFeedSourceStatus :exec
UPDATE feed_sources
SET last_fetched_at = ?, last_status = ?, item_count = ?
WHERE source = ?;
