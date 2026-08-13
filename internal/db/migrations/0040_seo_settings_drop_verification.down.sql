-- Restore the search-engine verification columns (mirrors 0039).
ALTER TABLE seo_settings
  ADD COLUMN google_verification VARCHAR(191) NOT NULL DEFAULT '',
  ADD COLUMN bing_verification   VARCHAR(191) NOT NULL DEFAULT '';
