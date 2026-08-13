-- Public-nav visibility toggles editable by a superadmin in the CMS: which of the
-- News / Podcast / Event top-level menu items are shown. A typed single-row table
-- matching seo_settings / contact_settings rather than the unused k/v `settings`
-- table (see 0027/0036/0038). id is TINYINT UNSIGNED, not TINYINT(1), so sqlc.yaml
-- doesn't map it to a Go bool; the show_* flags are TINYINT(1) so they do.
CREATE TABLE menu_settings (
  id           TINYINT UNSIGNED PRIMARY KEY DEFAULT 1,
  show_news    TINYINT(1) NOT NULL DEFAULT 1,
  show_podcast TINYINT(1) NOT NULL DEFAULT 1,
  show_event   TINYINT(1) NOT NULL DEFAULT 1,
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed the single row with every section visible (current behaviour).
INSERT INTO menu_settings (id) VALUES (1);
