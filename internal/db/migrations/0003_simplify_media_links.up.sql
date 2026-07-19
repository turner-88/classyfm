-- Media badges are exactly one per platform (Instagram/Facebook/X/YouTube) — no
-- label/handle/embed content, just the account URL. An empty url means "not
-- configured, don't show the badge".
ALTER TABLE media_links
  DROP COLUMN label,
  DROP COLUMN handle,
  DROP COLUMN embed_html,
  DROP COLUMN sort_order,
  DROP COLUMN is_active,
  MODIFY COLUMN url VARCHAR(1000) NOT NULL DEFAULT '',
  ADD UNIQUE KEY uq_platform (platform);

INSERT INTO media_links (platform, url) VALUES
  ('instagram', 'https://www.instagram.com/classyfm/'),
  ('facebook', 'https://www.facebook.com/classyfmpadang'),
  ('x', 'https://twitter.com/classyfm'),
  ('youtube', 'https://www.youtube.com/user/classyfm')
ON DUPLICATE KEY UPDATE url = VALUES(url);
