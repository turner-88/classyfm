-- Ad banners: admin-managed sponsor creatives rendered by the public layout on
-- every page. ad_slots holds the fixed set of placements (one row per render
-- site in layouts/base.html) plus how that placement presents its banners;
-- ad_banners holds the creatives themselves.
CREATE TABLE ad_slots (
  slot             ENUM('top','bottom') NOT NULL PRIMARY KEY,
  label            VARCHAR(100) NOT NULL DEFAULT '',
  display_mode     ENUM('stacked','slideshow') NOT NULL DEFAULT 'stacked',
  rotate_secs      INT NOT NULL DEFAULT 6,
  show_placeholder TINYINT(1) NOT NULL DEFAULT 0,
  placeholder_text VARCHAR(255) NOT NULL DEFAULT 'Ad space available',
  is_active        TINYINT(1) NOT NULL DEFAULT 1,
  updated_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO ad_slots (slot, label) VALUES
  ('top', 'Top — below the header'),
  ('bottom', 'Bottom — above the footer');

CREATE TABLE ad_banners (
  id         BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  slot       ENUM('top','bottom') NOT NULL,
  title      VARCHAR(255) NOT NULL DEFAULT '',
  alt_text   VARCHAR(255) NOT NULL DEFAULT '',
  image_url  VARCHAR(1000) NOT NULL,
  link_url   VARCHAR(1000) NULL,
  sort_order INT NOT NULL DEFAULT 0,
  is_active  TINYINT(1) NOT NULL DEFAULT 1,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_ad_banners_slot_active (slot, is_active, sort_order)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
