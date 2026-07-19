-- name: GetSetting :one
SELECT k, v FROM settings WHERE k = ?;

-- name: ListSettings :many
SELECT k, v FROM settings ORDER BY k;

-- name: UpsertSetting :exec
INSERT INTO settings (k, v) VALUES (?, ?)
ON DUPLICATE KEY UPDATE v = VALUES(v);
