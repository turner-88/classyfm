DELETE FROM media_links WHERE platform = 'tiktok';

ALTER TABLE media_links MODIFY COLUMN platform ENUM('instagram','facebook','x','youtube','spotify') NOT NULL;
