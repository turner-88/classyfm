-- name: GetLegalPage :one
SELECT * FROM legal_pages WHERE slug = ?;

-- name: ListLegalPages :many
SELECT * FROM legal_pages ORDER BY slug;

-- name: UpdateLegalPage :exec
UPDATE legal_pages SET title = ?, intro = ?, body = ? WHERE slug = ?;
