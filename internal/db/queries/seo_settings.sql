-- name: GetSeoSettings :one
SELECT * FROM seo_settings WHERE id = 1;

-- name: UpdateSeoSettings :exec
UPDATE seo_settings
SET keywords = ?, default_description = ?, og_image_url = ?, google_verification = ?, bing_verification = ?
WHERE id = 1;
