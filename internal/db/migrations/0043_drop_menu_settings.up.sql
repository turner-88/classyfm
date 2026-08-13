-- Remove the public-nav visibility toggle feature (added in 0042). The News /
-- Podcast / Event menu items are now always shown, so the settings table is gone.
DROP TABLE IF EXISTS menu_settings;
