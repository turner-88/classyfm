-- Recreate the menu_settings table (mirrors 0042_menu_settings.up.sql).
CREATE TABLE menu_settings (
  id           TINYINT UNSIGNED PRIMARY KEY DEFAULT 1,
  show_news    TINYINT(1) NOT NULL DEFAULT 1,
  show_podcast TINYINT(1) NOT NULL DEFAULT 1,
  show_event   TINYINT(1) NOT NULL DEFAULT 1,
  updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO menu_settings (id) VALUES (1);
