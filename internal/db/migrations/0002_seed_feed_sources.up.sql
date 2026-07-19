-- Seed one row per aggregation source. `endpoint` is admin-editable (channel id for
-- YouTube, RSS feed URL for the others); the worker falls back to built-in defaults
-- when a row's endpoint is empty.
INSERT INTO feed_sources (source, is_enabled, endpoint) VALUES
  ('youtube', 1, 'UC7yseGthOK8sOtahtxvxDbg'),
  ('klikpositif', 1, 'https://klikpositif.com/feed/'),
  ('katasumbar', 1, 'https://katasumbar.com/feed/');
