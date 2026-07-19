-- name: CreatePasswordResetToken :exec
INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
VALUES (?, ?, ?);

-- name: GetValidPasswordResetToken :one
SELECT * FROM password_reset_tokens
WHERE token_hash = ? AND used_at IS NULL AND expires_at > NOW();

-- name: MarkPasswordResetTokenUsed :exec
UPDATE password_reset_tokens SET used_at = NOW() WHERE token_hash = ?;
