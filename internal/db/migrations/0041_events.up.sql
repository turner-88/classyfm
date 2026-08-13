-- Events are admin-authored entries for station events and promotional offers. Like
-- podcasts (0029) they get a dedicated table with their own list/detail/CRUD, but the
-- banner image is uploaded to disk (image_url is a /uploads/... path resolved at save
-- time via saveUploadedImage), not derived from a third-party link, so there is no
-- broadcaster junction and no external URL as the source of truth.
--
-- category splits the two kinds the section covers and drives the public filter chips;
-- it is a fixed two-value ENUM rather than a lookup table. event_date/location/link_url
-- are all optional: a promo may run without a fixed date or venue, and the CTA link is
-- shown only when set. image_url is nullable because the banner is optional (the list and
-- detail fall back to a placeholder).

CREATE TABLE events (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  title        VARCHAR(500)  NOT NULL,
  slug         VARCHAR(255)  NOT NULL,
  category     ENUM('event','promo') NOT NULL DEFAULT 'event',
  description  TEXT          NOT NULL,
  image_url    VARCHAR(1000) NULL,
  event_date   DATETIME      NULL,
  location     VARCHAR(255)  NULL,
  link_url     VARCHAR(1000) NULL,
  is_published TINYINT(1)    NOT NULL DEFAULT 1,
  created_at   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_events_slug (slug),
  KEY idx_events_published_created (is_published, created_at),
  KEY idx_events_category (category)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
