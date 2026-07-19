-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, user_email_snapshot, action, entity_type, entity_id, detail, ip_address)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: ListAuditLogs :many
SELECT * FROM audit_logs ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?;

-- name: CountAuditLogs :one
SELECT COUNT(*) FROM audit_logs;
