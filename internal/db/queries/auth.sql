-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = ?;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = ?;

-- name: CreateUser :execresult
INSERT INTO users (email, password_hash, name, role, is_active)
VALUES (?, ?, ?, ?, ?);

-- name: CountUsers :one
SELECT COUNT(*) FROM users
WHERE name LIKE sqlc.arg(search) OR email LIKE sqlc.arg(search);

-- name: ListUsers :many
SELECT * FROM users
WHERE name LIKE sqlc.arg(search) OR email LIKE sqlc.arg(search)
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'name' AND sqlc.arg(dir) = 'asc' THEN name END ASC,
  CASE WHEN sqlc.arg(sort) = 'name' AND sqlc.arg(dir) = 'desc' THEN name END DESC,
  CASE WHEN sqlc.arg(sort) = 'email' AND sqlc.arg(dir) = 'asc' THEN email END ASC,
  CASE WHEN sqlc.arg(sort) = 'email' AND sqlc.arg(dir) = 'desc' THEN email END DESC,
  name ASC, id ASC
LIMIT ? OFFSET ?;

-- name: UpdateUser :exec
UPDATE users SET name = ?, email = ?, role = ?, is_active = ? WHERE id = ?;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = ? WHERE id = ?;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = ?;

-- name: CreateSession :exec
INSERT INTO sessions (token, user_id, expires_at)
VALUES (?, ?, ?);

-- name: GetSession :one
SELECT
  s.token, s.user_id, s.expires_at,
  u.email, u.name, u.role
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token = ? AND s.expires_at > NOW() AND u.is_active = 1;

-- name: DeleteSession :exec
DELETE FROM sessions WHERE token = ?;

-- name: DeleteSessionsByUserID :exec
DELETE FROM sessions WHERE user_id = ?;

-- name: DeleteExpiredSessions :exec
DELETE FROM sessions WHERE expires_at <= NOW();
