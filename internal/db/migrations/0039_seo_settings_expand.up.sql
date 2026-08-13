-- Expand the single-row SEO settings with the other superadmin-configurable SEO
-- surfaces: a site-wide default meta description, a default social-share image
-- (og:image), and search-engine verification codes. Empty defaults preserve
-- current behavior — the app keeps its existing station-name description fallback
-- when default_description is blank, and JSON-LD falls back to the static logo
-- when og_image_url is blank.
ALTER TABLE seo_settings
  ADD COLUMN default_description VARCHAR(500) NOT NULL DEFAULT '',
  ADD COLUMN og_image_url        VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN google_verification VARCHAR(191) NOT NULL DEFAULT '',
  ADD COLUMN bing_verification   VARCHAR(191) NOT NULL DEFAULT '';
