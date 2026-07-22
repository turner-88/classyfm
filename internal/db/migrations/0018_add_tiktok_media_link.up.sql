ALTER TABLE media_links MODIFY COLUMN platform ENUM('instagram','facebook','x','youtube','spotify','tiktok') NOT NULL;

INSERT INTO media_links (platform, url) VALUES
  ('tiktok', 'https://www.tiktok.com/place/Classy-FM-Radio-Padang-21568226304698972')
ON DUPLICATE KEY UPDATE url = VALUES(url);
