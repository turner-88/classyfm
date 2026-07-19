-- name: ListMediaLinks :many
SELECT * FROM media_links
ORDER BY platform ASC;

-- name: UpdateMediaLinkURL :exec
UPDATE media_links SET url = ? WHERE platform = ?;
