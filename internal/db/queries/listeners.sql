-- Listener history: written by internal/listeners.Sampler, read by the admin
-- dashboard's listener chart (internal/handlers/admin/dashboard_charts.go).
-- See migration 0028 for why the daily rollup and the raw trail are separate.

-- name: InsertListenerSample :exec
INSERT INTO listener_samples (sampled_at, listeners, is_live) VALUES (?, ?, ?);

-- name: UpsertListenerDay :exec
-- Folds one sample into its day's row, monotonic in peak_listeners so a quiet
-- afternoon can never lower the morning's peak.
--
-- peak_at is assigned before peak_listeners on purpose: MySQL evaluates the SET
-- list left to right, so the comparison has to run while peak_listeners still
-- holds the old value.
INSERT INTO listener_stats (stat_date, peak_listeners, peak_at, last_listeners, sample_count)
VALUES (sqlc.arg(stat_date), sqlc.arg(listeners), sqlc.arg(sampled_at), sqlc.arg(listeners), 1)
ON DUPLICATE KEY UPDATE
  peak_at        = IF(sqlc.arg(listeners) > peak_listeners, sqlc.arg(sampled_at), peak_at),
  peak_listeners = GREATEST(peak_listeners, sqlc.arg(listeners)),
  last_listeners = sqlc.arg(listeners),
  sample_count   = sample_count + 1;

-- name: ListListenerStats :many
SELECT * FROM listener_stats WHERE stat_date >= ? ORDER BY stat_date ASC;

-- name: GetListenerDay :one
SELECT * FROM listener_stats WHERE stat_date = ?;

-- name: ListListenerSamples :many
-- Backs the dashboard chart's intraday groupings. Bounded by the caller's window
-- (24h at most, so ~288 rows at the default sampling interval) and covered end to
-- end by idx_listener_samples_sampled_at.
SELECT * FROM listener_samples WHERE sampled_at >= ? ORDER BY sampled_at ASC;

-- name: PruneListenerSamples :exec
DELETE FROM listener_samples WHERE sampled_at < ?;
