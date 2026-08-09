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

-- name: UnhideChatMessage :exec
-- Reverses SoftDeleteChatMessage: an admin un-hides a previously moderated message.
UPDATE chat_messages SET is_deleted = 0 WHERE id = ?;

-- name: BanChatUser :exec
UPDATE chat_users SET is_banned = 1 WHERE id = ?;

-- name: UnbanChatUser :exec
-- Reverses BanChatUser: an admin lifts a chat user's posting ban.
UPDATE chat_users SET is_banned = 0 WHERE id = ?;

-- name: GetChatMessage :one
SELECT id, chat_user_id, body, is_deleted, created_at FROM chat_messages WHERE id = ?;

-- Admin moderation list. Unlike the public reads, this INCLUDES soft-deleted rows
-- (is_deleted is selected so the panel can mark and un-hide them) and searches both the
-- body and the author name. Sort/dir are bound params (never interpolated); the trailing
-- id DESC is the stable tiebreak.
-- name: ListChatMessagesAdmin :many
SELECT
  m.id, m.chat_user_id, m.body, m.is_deleted, m.created_at,
  u.name AS author_name, u.avatar_url AS author_avatar,
  u.is_admin AS author_is_admin, u.is_banned AS author_is_banned
FROM chat_messages m
JOIN chat_users u ON u.id = m.chat_user_id
WHERE (m.body LIKE sqlc.arg(search) OR u.name LIKE sqlc.arg(search))
ORDER BY
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'asc' THEN m.id END ASC,
  CASE WHEN sqlc.arg(sort) = 'created_at' AND sqlc.arg(dir) = 'desc' THEN m.id END DESC,
  m.id DESC
LIMIT ? OFFSET ?;

-- name: CountChatMessagesAdmin :one
SELECT COUNT(*)
FROM chat_messages m
JOIN chat_users u ON u.id = m.chat_user_id
WHERE (m.body LIKE sqlc.arg(search) OR u.name LIKE sqlc.arg(search));

-- Live-mode polling delta for the admin panel: newer-than-id, INCLUDING soft-deleted rows
-- (so a moderator watching live still sees what was hidden and by-whom context is intact).
-- name: ListChatMessagesAdminSince :many
SELECT
  m.id, m.chat_user_id, m.body, m.is_deleted, m.created_at,
  u.name AS author_name, u.avatar_url AS author_avatar,
  u.is_admin AS author_is_admin, u.is_banned AS author_is_banned
FROM chat_messages m
JOIN chat_users u ON u.id = m.chat_user_id
WHERE m.id > ?
ORDER BY m.id ASC
LIMIT ?;

-- name: ListChatUsers :many
SELECT id, google_sub, email, name, avatar_url, is_admin, is_banned, created_at, updated_at
FROM chat_users
WHERE (name LIKE sqlc.arg(search) OR email LIKE sqlc.arg(search))
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: CountChatUsers :one
SELECT COUNT(*) FROM chat_users
WHERE (name LIKE sqlc.arg(search) OR email LIKE sqlc.arg(search));
