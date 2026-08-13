-- Drop the search-engine verification codes from the single-row SEO settings.
-- Site verification is now handled via DNS TXT records, so the app no longer
-- renders google-site-verification / msvalidate.01 meta tags and the columns are
-- unused. The other 0039 columns (default_description, og_image_url) are kept.
ALTER TABLE seo_settings
  DROP COLUMN google_verification,
  DROP COLUMN bing_verification;
