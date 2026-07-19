-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, user_email_snapshot, action, entity_type, entity_id, detail, ip_address)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs
WHERE action LIKE sqlc.arg(search) OR entity_type LIKE sqlc.arg(search) OR detail LIKE sqlc.arg(search) OR user_email_snapshot LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'asc' THEN created_at END ASC,
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'desc' THEN created_at END DESC,
  CASE WHEN sqlc.arg(sort) = 'action' AND sqlc.arg(dir) = 'asc' THEN action END ASC,
  CASE WHEN sqlc.arg(sort) = 'action' AND sqlc.arg(dir) = 'desc' THEN action END DESC,
  CASE WHEN sqlc.arg(sort) = 'entity_type' AND sqlc.arg(dir) = 'asc' THEN entity_type END ASC,
  CASE WHEN sqlc.arg(sort) = 'entity_type' AND sqlc.arg(dir) = 'desc' THEN entity_type END DESC,
  created_at DESC, id DESC
LIMIT ? OFFSET ?;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs
WHERE action LIKE sqlc.arg(search) OR entity_type LIKE sqlc.arg(search) OR detail LIKE sqlc.arg(search) OR user_email_snapshot LIKE sqlc.arg(search);
