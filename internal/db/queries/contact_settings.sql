-- name: GetContactSettings :one
SELECT * FROM contact_settings WHERE id = 1;

-- name: UpdateContactSettings :exec
UPDATE contact_settings
SET whatsapp_number = ?, whatsapp_message = ?, phone = ?, email = ?
WHERE id = 1;
