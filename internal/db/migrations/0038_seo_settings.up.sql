-- SEO keywords for the public site's <meta name="keywords"> tag, editable by a
-- superadmin in the CMS. A typed single-row table matching contact_settings /
-- hero_settings rather than the unused k/v `settings` table (see 0027/0036). id
-- is TINYINT UNSIGNED, not TINYINT(1), so sqlc.yaml doesn't map it to a Go bool.
CREATE TABLE seo_settings (
  id         TINYINT UNSIGNED PRIMARY KEY DEFAULT 1,
  keywords   VARCHAR(1000) NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed the single row with the station's initial keyword set.
INSERT INTO seo_settings (id, keywords) VALUES
(1, 'Classy FM Padang, Classy 103.4 FM, Radio Padang, Radio FM Sumatera Barat, Live Streaming Classy FM, Program Radio Classy FM, News Padang, Radio urang awak, Radio Minang, Radio Minangkabau');
