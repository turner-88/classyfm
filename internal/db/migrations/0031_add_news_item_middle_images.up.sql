-- middle_images holds a JSON array of upload URLs for the mid-article gallery on
-- admin-authored Hot Release items (one image renders as a single plate, several
-- as a slideshow). Kept as a JSON-encoded TEXT column rather than a child table
-- because every image elsewhere is a scalar URL string and Hot Release is
-- low-volume, manually authored. NULL / empty array when the article has none.
ALTER TABLE news_items ADD COLUMN middle_images MEDIUMTEXT NULL AFTER thumb_url;
