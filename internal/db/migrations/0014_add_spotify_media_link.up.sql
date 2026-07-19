ALTER TABLE media_links MODIFY COLUMN platform ENUM('instagram','facebook','x','youtube','spotify') NOT NULL;

INSERT INTO media_links (platform, url) VALUES
  ('spotify', 'https://open.spotify.com/show/32heKQCREVb5GeJZZtkv8b')
ON DUPLICATE KEY UPDATE url = VALUES(url);
