-- Hero slideshow. The home page hero used to be exactly "the 3 latest published
-- news items"; it now mixes those with image slides the admin uploads, so the
-- station can put its own promos in the most prominent slot on the site.
--
-- hero_slides holds the image slides, with full field parity against a news
-- slide (badge, title, excerpt, link) so both sides can render through one
-- view-model and cannot drift apart visually. hero_settings is the single row
-- deciding how many of each go in and in what order.
--
-- Note the settings live in a typed single-row table rather than the k/v
-- `settings` table from 0001: every config surface in this app is a typed table
-- (ad_slots, about_page_banner, media_links, feed_sources) and the k/v table has
-- no call sites at all, so using it here would introduce a second, weaker config
-- idiom for one feature. A typed table also gets order_mode a real ENUM.
CREATE TABLE hero_settings (
  -- TINYINT UNSIGNED, deliberately not TINYINT(1): sqlc.yaml maps tinyint(1) to
  -- Go bool, which would make the primary key a bool. Same shape as
  -- about_page_banner.id.
  id         TINYINT UNSIGNED PRIMARY KEY DEFAULT 1,
  order_mode ENUM('random','updated','news_first','images_first') NOT NULL DEFAULT 'updated',
  news_count INT NOT NULL DEFAULT 3,
  max_images INT NOT NULL DEFAULT 4,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO hero_settings (id) VALUES (1);

CREATE TABLE hero_slides (
  id              BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
  title           VARCHAR(255) NOT NULL DEFAULT '',
  badge_label     VARCHAR(100) NOT NULL DEFAULT '',
  excerpt         TEXT NULL,
  image_url       VARCHAR(1000) NOT NULL,
  link_url        VARCHAR(1000) NULL,
  open_in_new_tab TINYINT(1) NOT NULL DEFAULT 0,
  sort_order      INT NOT NULL DEFAULT 0,
  is_active       TINYINT(1) NOT NULL DEFAULT 1,
  created_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  -- Covers the public read (ListActiveHeroSlides) end to end.
  KEY idx_hero_slides_active (is_active, sort_order, id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
