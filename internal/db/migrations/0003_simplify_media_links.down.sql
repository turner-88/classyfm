DELETE FROM media_links;

ALTER TABLE media_links
  DROP KEY uq_platform,
  ADD COLUMN label VARCHAR(255) NULL AFTER platform,
  ADD COLUMN handle VARCHAR(255) NULL AFTER url,
  ADD COLUMN embed_html TEXT NULL AFTER handle,
  ADD COLUMN sort_order INT NOT NULL DEFAULT 0 AFTER embed_html,
  ADD COLUMN is_active TINYINT(1) NOT NULL DEFAULT 1 AFTER sort_order;
