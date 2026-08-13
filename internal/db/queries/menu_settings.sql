-- name: GetMenuSettings :one
SELECT * FROM menu_settings WHERE id = 1;

-- name: UpdateMenuSettings :exec
UPDATE menu_settings
SET show_news = ?, show_podcast = ?, show_event = ?
WHERE id = 1;
