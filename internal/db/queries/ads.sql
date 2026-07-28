-- name: ListAdSlots :many
SELECT * FROM ad_slots ORDER BY slot ASC;

-- name: UpdateAdSlot :exec
UPDATE ad_slots
SET display_mode = ?, rotate_secs = ?, show_placeholder = ?, placeholder_text = ?, is_active = ?
WHERE slot = ?;

-- name: ListActiveAdBannersForPage :many
SELECT b.* FROM ad_banners b
WHERE b.is_active = 1
  AND (
    NOT EXISTS (SELECT 1 FROM ad_banner_pages p WHERE p.banner_id = b.id)
    OR EXISTS (SELECT 1 FROM ad_banner_pages p WHERE p.banner_id = b.id AND p.page = ?)
  )
ORDER BY b.slot ASC, b.sort_order ASC, b.id ASC;

-- name: ListAdBanners :many
SELECT * FROM ad_banners ORDER BY slot ASC, sort_order ASC, id ASC;

-- name: GetAdBanner :one
SELECT * FROM ad_banners WHERE id = ?;

-- name: CreateAdBanner :execresult
INSERT INTO ad_banners (slot, title, alt_text, image_url, link_url, sort_order, is_active)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: UpdateAdBanner :exec
UPDATE ad_banners
SET slot=?, title=?, alt_text=?, image_url=?, link_url=?, sort_order=?, is_active=?
WHERE id=?;

-- name: DeleteAdBanner :exec
DELETE FROM ad_banners WHERE id = ?;

-- name: ListAdBannerPages :many
SELECT banner_id, page FROM ad_banner_pages;

-- name: GetAdBannerPages :many
SELECT page FROM ad_banner_pages WHERE banner_id = ?;

-- name: DeleteAdBannerPages :exec
DELETE FROM ad_banner_pages WHERE banner_id = ?;

-- name: CreateAdBannerPage :exec
INSERT INTO ad_banner_pages (banner_id, page) VALUES (?, ?);
