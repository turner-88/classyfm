-- Per-page ad targeting. A banner with no rows here shows on every public page,
-- which is exactly what every banner created before this migration does - so no
-- backfill is needed and existing banners keep their behavior.
--
-- Keys must stay in sync with models.AdPages (internal/models/adpages.go).
CREATE TABLE ad_banner_pages (
  banner_id BIGINT UNSIGNED NOT NULL,
  page      ENUM('home','about','program','program_detail','live','news','news_detail','broadcasters','broadcaster_detail') NOT NULL,
  PRIMARY KEY (banner_id, page),
  CONSTRAINT fk_ad_banner_pages_banner FOREIGN KEY (banner_id) REFERENCES ad_banners (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
