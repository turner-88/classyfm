-- name: UpsertChatUser :execresult
-- Called on every Google login: creates the chat user on first sign-in, and on return
-- refreshes profile fields plus the is_admin badge snapshot (recomputed by the handler).
-- created_at/updated_at are driven by the column defaults, not listed here.
INSERT INTO chat_users (google_sub, email, name, avatar_url, is_admin)
VALUES (?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
  email = VALUES(email),
  name = VALUES(name),
  avatar_url = VALUES(avatar_url),
  is_admin = VALUES(is_admin);

-- name: GetChatUserByGoogleSub :one
SELECT * FROM chat_users WHERE google_sub = ?;

-- name: GetChatUserByID :one
SELECT * FROM chat_users WHERE id = ?;

-- name: CreateChatMessage :execresult
INSERT INTO chat_messages (chat_user_id, body) VALUES (?, ?);

-- name: ListRecentChatMessages :many
SELECT
  m.id, m.chat_user_id, m.body, m.created_at,
  u.name AS author_name, u.avatar_url AS author_avatar, u.is_admin AS author_is_admin
FROM chat_messages m
JOIN chat_users u ON u.id = m.chat_user_id
WHERE m.is_deleted = 0
ORDER BY m.id DESC
LIMIT ?;

-- name: ListChatMessagesSince :many
SELECT
  m.id, m.chat_user_id, m.body, m.created_at,
  u.name AS author_name, u.avatar_url AS author_avatar, u.is_admin AS author_is_admin
FROM chat_messages m
JOIN chat_users u ON u.id = m.chat_user_id
WHERE m.is_deleted = 0 AND m.id > ?
ORDER BY m.id ASC
LIMIT ?;

-- name: SoftDeleteChatMessage :exec
UPDATE chat_messages SET is_deleted = 1 WHERE id = ?;

-- name: BanChatUser :exec
UPDATE chat_users SET is_banned = 1 WHERE id = ?;
